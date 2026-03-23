package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

const configFile = "config.json"

// Config holds all application settings.
type Config struct {
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	ChannelName     string `json:"channel_name"`
	DaysBack        int    `json:"days_back"`
	MinViews        int    `json:"min_views"`
	Whitelist       string `json:"whitelist"`
	Blacklist       string `json:"blacklist"`
	ScheduleEnabled bool   `json:"schedule_enabled"`
	ScheduleHour    int    `json:"schedule_hour"`
	ScheduleMinute  int    `json:"schedule_minute"`
	DownloadMode      string `json:"download_mode"` // "download" or "local"
	Port              int    `json:"port"`
	DownloadDir       string `json:"download_dir"`
	ClipPauseSeconds  float64 `json:"clip_pause_seconds"`
}

var (
	globalConfig Config
	configMu     sync.RWMutex
)

// LoadConfig reads config.json, falling back to .env for missing values.
func LoadConfig() Config {
	cfg := defaultConfig()

	// Try config.json first.
	// We decode twice: once into a raw map to know which keys are actually
	// present in the file (so 0 can be distinguished from "key absent"),
	// and once into the typed Config struct.
	if data, err := os.ReadFile(configFile); err == nil {
		var loaded Config
		var raw map[string]json.RawMessage
		if json.Unmarshal(data, &loaded) == nil && json.Unmarshal(data, &raw) == nil {
			fileKeys := make(map[string]bool, len(raw))
			for k := range raw {
				fileKeys[k] = true
			}
			mergeConfig(&cfg, loaded, fileKeys)
		}
	}

	// Fall back to .env for any empty required fields
	_ = godotenv.Load(".env")
	applyEnvFallback(&cfg)

	configMu.Lock()
	globalConfig = cfg
	configMu.Unlock()

	return cfg
}

// GetConfig returns a copy of the current config (thread-safe).
func GetConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}

// SaveConfig persists the config to config.json atomically and updates the global.
func SaveConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := configFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, configFile); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}

	configMu.Lock()
	globalConfig = cfg
	configMu.Unlock()
	return nil
}

func defaultConfig() Config {
	return Config{
		DaysBack:     1095,
		MinViews:     300,
		DownloadMode: "download",
		Port:         42069,
		DownloadDir:  "Twitch_Clips",
	}
}

// mergeConfig copies fields from src into dst.
// String fields: only overridden when non-empty (so old config files without a
// field keep the default).  Numeric/bool fields from the file are always
// applied — 0 is a valid user choice (e.g. no minimum-view filter) and must
// not silently revert to the compiled default on the next restart.
func mergeConfig(dst *Config, src Config, fileKeys map[string]bool) {
	if src.ClientID != "" {
		dst.ClientID = src.ClientID
	}
	if src.ClientSecret != "" {
		dst.ClientSecret = src.ClientSecret
	}
	if src.ChannelName != "" {
		dst.ChannelName = src.ChannelName
	}
	if fileKeys["days_back"] {
		dst.DaysBack = src.DaysBack
	}
	if fileKeys["min_views"] {
		dst.MinViews = src.MinViews
	}
	if src.Whitelist != "" {
		dst.Whitelist = src.Whitelist
	}
	if src.Blacklist != "" {
		dst.Blacklist = src.Blacklist
	}
	dst.ScheduleEnabled = src.ScheduleEnabled
	dst.ScheduleHour = src.ScheduleHour
	dst.ScheduleMinute = src.ScheduleMinute
	if src.DownloadMode != "" {
		dst.DownloadMode = src.DownloadMode
	}
	if fileKeys["port"] {
		dst.Port = src.Port
	}
	if src.DownloadDir != "" {
		dst.DownloadDir = src.DownloadDir
	}
	if fileKeys["clip_pause_seconds"] {
		dst.ClipPauseSeconds = src.ClipPauseSeconds
	}
}

func applyEnvFallback(cfg *Config) {
	if cfg.ClientID == "" {
		cfg.ClientID = os.Getenv("CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = os.Getenv("CLIENT_SECRET")
	}
	if cfg.ChannelName == "" {
		cfg.ChannelName = os.Getenv("CHANNEL_NAME")
	}
	if cfg.DaysBack == 0 {
		if v := envInt("DAYS_BACK", 0); v != 0 {
			cfg.DaysBack = v
		}
	}
	if cfg.MinViews == 0 {
		if v := envInt("MIN_VIEWS", 0); v != 0 {
			cfg.MinViews = v
		}
	}
	if cfg.Whitelist == "" {
		cfg.Whitelist = strings.TrimSpace(os.Getenv("WHITELIST"))
	}
	if cfg.Blacklist == "" {
		cfg.Blacklist = strings.TrimSpace(os.Getenv("BLACKLIST"))
	}
}

func envInt(key string, def int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// OutputFile returns the JSON output filename for a given channel.
func (c Config) OutputFile() string {
	if c.ChannelName != "" {
		return c.ChannelName + "_mp4_urls.json"
	}
	return "mp4_urls.json"
}

// BinaryDir returns the directory of the running executable.
func BinaryDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
