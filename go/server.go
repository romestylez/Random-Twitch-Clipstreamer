package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed static
var staticFiles embed.FS

// Server holds state shared across HTTP handlers.
type Server struct {
	logger    *log.Logger
	logRing   *LogRing
	scheduler *Scheduler
	fetchFn   func()
	state     *FetchState
}

func newServer(logger *log.Logger, ring *LogRing, sched *Scheduler, fetchFn func(), state *FetchState) *Server {
	return &Server{
		logger:    logger,
		logRing:   ring,
		scheduler: sched,
		fetchFn:   fetchFn,
		state:     state,
	}
}

// ListenAndServe starts the HTTP server on the given port.
func (s *Server) ListenAndServe(port int) error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	addr := fmt.Sprintf(":%d", port)
	s.logger.Printf("🌐 HTTP server listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Embedded static files (player.html, admin.html)
	staticFS, _ := fs.Sub(staticFiles, "static")
	staticHandler := http.FileServer(http.FS(staticFS))

	// /admin → /admin.html
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/admin.html"
		staticHandler.ServeHTTP(w, r)
	})
	mux.Handle("/player.html", staticHandler)
	mux.Handle("/admin.html", staticHandler)

	// Serve local downloaded clips
	mux.Handle("/Twitch_Clips/", http.StripPrefix("/Twitch_Clips/",
		http.FileServer(http.Dir("Twitch_Clips"))))

	// PHP-alias + clean endpoint pairs
	for _, p := range []string{"/get_lastplayed", "/get_lastplayed.php"} {
		mux.HandleFunc(p, s.handleGetLastPlayed)
	}
	for _, p := range []string{"/save_lastplayed", "/save_lastplayed.php"} {
		mux.HandleFunc(p, s.handleSaveLastPlayed)
	}
	for _, p := range []string{"/write_clipdate", "/write_clipdate.php"} {
		mux.HandleFunc(p, s.handleWriteClipDate)
	}

	// Canonical clip list alias: player.html always fetches /clip_mp4_urls.json
	mux.HandleFunc("/clip_mp4_urls.json", s.handleClipList)

	// Admin API
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/fetch", s.handleFetch)
	mux.HandleFunc("/api/status", s.handleStatus)

	// catch-all: serve clip_date.html and *_mp4_urls.json from working dir,
	// redirect / → /player.html
	mux.HandleFunc("/", s.handleCatchAll)
}

func (s *Server) handleCatchAll(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/player.html", http.StatusFound)
		return
	}
	// Serve files from working directory (clip JSON, clip_date.html, etc.)
	path := strings.TrimPrefix(r.URL.Path, "/")
	http.ServeFile(w, r, path)
}

// ── PHP-compatible handlers ────────────────────────────────────────────────

func (s *Server) handleGetLastPlayed(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, err := os.ReadFile("clip_history.json")
	if err != nil {
		w.Write([]byte("{}"))
		return
	}
	w.Write(data)
}

func (s *Server) handleSaveLastPlayed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	pretty, _ := json.MarshalIndent(raw, "", "  ")
	if err := atomicWriteFile("clip_history.json", pretty); err != nil {
		http.Error(w, "Write failed", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("OK"))
}

func (s *Server) handleWriteClipDate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Date string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Date == "" {
		http.Error(w, "Missing date", http.StatusBadRequest)
		return
	}

	var formatted string
	if t, err := time.Parse("2006-01-02", body.Date); err == nil {
		formatted = t.Format("02.01.2006")
	} else {
		formatted = body.Date
	}

	html := buildClipDateHTML(formatted)
	if err := atomicWriteFile("clip_date.html", []byte(html)); err != nil {
		http.Error(w, "Write failed", http.StatusInternalServerError)
		return
	}
}

// buildClipDateHTML reproduces the exact output of the PHP write_clipdate.php.
func buildClipDateHTML(formatted string) string {
	return `<!DOCTYPE html>
<html lang="de">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="refresh" content="1">
  <style>
    body {
      margin: 0;
      padding: 0;
      background: transparent;
      font-family: "Segoe UI Emoji", sans-serif;
      display: flex;
      justify-content: flex-start;
      align-items: flex-end;
      height: 100vh;
    }
    .bubble {
      margin: 20px;
      background: rgba(0, 0, 0, 0.6);
      color: white;
      font-size: 32px;
      padding: 10px 20px;
      border-radius: 20px;
      max-width: 90%;
      box-shadow: 0 4px 10px rgba(0, 0, 0, 0.3);
    }
  </style>
</head>
<body>
  <div class="bubble">📅 Clip vom ` + formatted + `</div>
</body>
</html>`
}

// ── Admin API handlers ──────────────────────────────────────────────────────

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(GetConfig())

	case http.MethodPost:
		var cfg Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Preserve existing sensitive values if the browser cleared the password fields.
		current := GetConfig()
		if cfg.ClientID == "" {
			cfg.ClientID = current.ClientID
		}
		if cfg.ClientSecret == "" {
			cfg.ClientSecret = current.ClientSecret
		}
		// Enforce sane defaults for zero values
		if cfg.Port == 0 {
			cfg.Port = 42069
		}
		if cfg.DaysBack == 0 {
			cfg.DaysBack = 1095
		}
		if cfg.DownloadDir == "" {
			cfg.DownloadDir = "Twitch_Clips"
		}
		if cfg.DownloadMode == "" {
			cfg.DownloadMode = "download"
		}
		if err := SaveConfig(cfg); err != nil {
			http.Error(w, "Save failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if s.scheduler != nil {
			s.scheduler.Reschedule(cfg)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.state.TryStart() {
		http.Error(w, "Fetch already running", http.StatusConflict)
		return
	}
	go func() {
		cfg := LoadConfig() // always re-read from disk so direct file edits are honoured
		err := FetchAndWrite(cfg, s.logger)
		errStr := ""
		if err != nil {
			errStr = err.Error()
			s.logger.Printf("❌ Fetch error: %v", err)
		}
		s.state.Finish(errStr)
	}()
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	running, lastRun, lastError := s.state.Snapshot()
	type response struct {
		Running   bool      `json:"running"`
		LastRun   time.Time `json:"last_run,omitempty"`
		LastError string    `json:"last_error,omitempty"`
		Logs      []string  `json:"logs"`
	}
	resp := response{
		Running:   running,
		LastRun:   lastRun,
		LastError: lastError,
		Logs:      s.logRing.Lines(50),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleClipList serves the channel's clip JSON under the fixed path
// /clip_mp4_urls.json that player.html expects by default.
func (s *Server) handleClipList(w http.ResponseWriter, r *http.Request) {
	cfg := GetConfig()
	http.ServeFile(w, r, cfg.OutputFile())
}

// atomicWriteFile writes data to path using a temp-file + rename pattern.
func atomicWriteFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
