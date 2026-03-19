package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	ytdlpURL        = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	concurrencyLimit = 5
)

// ClipEntry is one entry in the output JSON file consumed by player.html.
type ClipEntry struct {
	URL  string `json:"url"`
	Date string `json:"date"`
}

// FetchAndWrite is the main orchestration function: fetches clips from Twitch,
// downloads or extracts URLs, cleans up obsolete files, and writes the output JSON.
func FetchAndWrite(cfg Config, logger *log.Logger) error {
	ytdlp, err := resolveYtDlp()
	if err != nil {
		return fmt.Errorf("yt-dlp not available: %w", err)
	}

	logger.Println("🔑 Getting OAuth token...")
	token, err := GetOAuthToken(cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		return fmt.Errorf("oauth: %w", err)
	}

	logger.Printf("👤 Getting user ID for %s...", cfg.ChannelName)
	userID, err := GetUserID(cfg.ChannelName, cfg.ClientID, token)
	if err != nil {
		return fmt.Errorf("user id: %w", err)
	}

	now := time.Now().UTC()
	startedAt := now.AddDate(0, 0, -cfg.DaysBack)

	logger.Printf("🎯 Fetching clips from the last %d days (min views: %d)...", cfg.DaysBack, cfg.MinViews)
	clips, err := GetClips(userID, cfg.ClientID, token, startedAt, now,
		cfg.MinViews,
		splitCategories(cfg.Whitelist),
		splitCategories(cfg.Blacklist),
	)
	if err != nil {
		return fmt.Errorf("fetch clips: %w", err)
	}
	logger.Printf("✅ Found %d clips.", len(clips))

	var entries []ClipEntry

	if cfg.DownloadMode == "local" {
		entries, err = runLocalMode(cfg, clips, ytdlp, logger)
	} else {
		entries, err = runDownloadMode(cfg, clips, ytdlp, logger)
	}
	if err != nil {
		return err
	}

	outFile := cfg.OutputFile()
	logger.Printf("💾 Writing %d entries to %s...", len(entries), outFile)
	if err := writeJSON(outFile, entries); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	logger.Printf("✅ Done. %d clips listed.", len(entries))
	return nil
}

// runDownloadMode downloads clips via yt-dlp and cleans up obsolete files.
func runDownloadMode(cfg Config, clips []Clip, ytdlp string, logger *log.Logger) ([]ClipEntry, error) {
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return nil, fmt.Errorf("create download dir: %w", err)
	}

	plannedSlugs := make(map[string]bool, len(clips))
	var downloaded []string

	for _, clip := range clips {
		slug := clip.ID
		plannedSlugs[slug] = true
		dateStr := clip.CreatedAt.Format("20060102")
		filename := dateStr + "_" + slug + ".mp4"
		filepath_ := filepath.Join(cfg.DownloadDir, filename)

		if _, err := os.Stat(filepath_); err == nil {
			logger.Printf("✔️  Already exists: %s", filename)
			downloaded = append(downloaded, toSlash(filepath_))
			continue
		}

		logger.Printf("⬇️  Downloading: %s", slug)
		cmd := exec.Command(ytdlp,
			"--paths", cfg.DownloadDir,
			"-o", filename,
			"--no-warnings",
			"--continue",
			"--ignore-errors",
			clip.URL,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Printf("⚠️  Failed to download %s: %v\n%s", slug, err, out)
			continue
		}
		if _, err := os.Stat(filepath_); err == nil {
			downloaded = append(downloaded, toSlash(filepath_))
		} else {
			logger.Printf("⚠️  Could not confirm download of: %s", slug)
		}
	}

	// Cleanup obsolete files
	logger.Println("🧹 Removing obsolete files...")
	entries, _ := filepath.Glob(filepath.Join(cfg.DownloadDir, "*.mp4"))
	for _, f := range entries {
		base := filepath.Base(f)
		name := strings.TrimSuffix(base, ".mp4")
		// filename format: YYYYMMDD_slug
		parts := strings.SplitN(name, "_", 2)
		slug := name
		if len(parts) == 2 {
			slug = parts[1]
		}
		if !plannedSlugs[slug] {
			if err := os.Remove(f); err != nil {
				logger.Printf("⚠️  Could not delete %s: %v", base, err)
			} else {
				logger.Printf("🗑️  Deleted: %s", base)
			}
		}
	}

	// Build output entries preserving clip metadata order
	downloadedSet := make(map[string]bool, len(downloaded))
	for _, d := range downloaded {
		downloadedSet[d] = true
	}

	var result []ClipEntry
	for _, clip := range clips {
		dateStr := clip.CreatedAt.Format("20060102")
		filename := dateStr + "_" + clip.ID + ".mp4"
		fp := toSlash(filepath.Join(cfg.DownloadDir, filename))
		if downloadedSet[fp] {
			result = append(result, ClipEntry{
				URL:  fp,
				Date: clip.CreatedAt.Format("2006-01-02"),
			})
		}
	}
	return result, nil
}

// runLocalMode extracts direct MP4 URLs using yt-dlp --print url (no download).
func runLocalMode(cfg Config, clips []Clip, ytdlp string, logger *log.Logger) ([]ClipEntry, error) {
	type result struct {
		entry ClipEntry
		ok    bool
	}

	results := make([]result, len(clips))
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup

	for i, clip := range clips {
		i, clip := i, clip
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			logger.Printf("🔍 Extracting MP4 URL for: %s", clip.ID)
			cmd := exec.Command(ytdlp, "--print", "url", "--no-warnings", clip.URL)
			out, err := cmd.Output()
			if err != nil {
				logger.Printf("⚠️  %s failed: %v", clip.ID, err)
				return
			}
			mp4URL := strings.TrimSpace(string(out))
			if mp4URL == "" || !strings.HasPrefix(mp4URL, "http") {
				logger.Printf("⚠️  %s: unexpected output: %q", clip.ID, mp4URL)
				return
			}
			results[i] = result{
				entry: ClipEntry{
					URL:  mp4URL,
					Date: clip.CreatedAt.Format("2006-01-02"),
				},
				ok: true,
			}
		}()
	}
	wg.Wait()

	var entries []ClipEntry
	for _, r := range results {
		if r.ok {
			entries = append(entries, r.entry)
		}
	}
	return entries, nil
}

// resolveYtDlp finds yt-dlp in PATH or next to the binary,
// downloading it automatically if not found (Windows only).
func resolveYtDlp() (string, error) {
	// 1. Check PATH
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p, nil
	}

	// 2. Check next to binary
	binDir := BinaryDir()
	candidates := []string{
		filepath.Join(binDir, "yt-dlp.exe"),
		filepath.Join(binDir, "yt-dlp"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	// 3. Auto-download yt-dlp.exe next to binary
	dest := filepath.Join(binDir, "yt-dlp.exe")
	log.Printf("📥 yt-dlp not found — downloading from GitHub to %s ...", dest)
	if err := downloadFile(dest, ytdlpURL); err != nil {
		return "", fmt.Errorf("auto-download yt-dlp: %w", err)
	}
	log.Printf("✅ yt-dlp downloaded successfully.")
	return dest, nil
}

// downloadFile downloads url to dest, showing basic progress.
func downloadFile(dest, srcURL string) error {
	resp, err := http.Get(srcURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, srcURL)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// writeJSON marshals data to a JSON file atomically.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func toSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
