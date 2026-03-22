package main

import (
	"io"
	"log"
	"os"
)

// commonSetup initialises working directory, config, logger, scheduler, HTTP
// server and the shared fetch-job closure. It is called by both the tray
// variant and the headless variant so the startup logic is not duplicated.
func commonSetup() (logger *log.Logger, ring *LogRing, srv *Server, sched *Scheduler, state *FetchState, fetchJob func()) {
	// Determine working directory:
	// If DATA_DIR is set (e.g. in Docker), use that so config.json and
	// Twitch_Clips/ end up on the mounted volume. Otherwise fall back to
	// the directory containing the executable (original desktop behaviour).
	if dataDir := os.Getenv("DATA_DIR"); dataDir != "" {
		_ = os.MkdirAll(dataDir, 0755)
		_ = os.Chdir(dataDir)
	} else if exePath, err := os.Executable(); err == nil {
		_ = os.Chdir(dirOf(exePath))
	}

	// Load configuration
	cfg := LoadConfig()

	// Set up fanout logger: writes to both log file and in-memory ring buffer
	ring = NewLogRing(200)
	logFile, err := os.OpenFile("clipstreamer.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logFile = os.Stderr
	}
	multi := io.MultiWriter(logFile, ring, os.Stdout)
	logger = log.New(multi, "", log.LstdFlags)

	logger.Println("🚀 Twitch Clipstreamer starting...")

	// Shared fetch state
	state = &FetchState{}

	// Fetch function used by scheduler and tray menu
	fetchJob = func() {
		if !state.TryStart() {
			logger.Println("⚠️  Fetch already running, skipping.")
			return
		}
		c := LoadConfig() // always re-read from disk so direct file edits are honoured
		err := FetchAndWrite(c, logger)
		errStr := ""
		if err != nil {
			errStr = err.Error()
			logger.Printf("❌ Fetch error: %v", err)
		}
		state.Finish(errStr)
	}

	// Scheduler
	sched = NewScheduler(logger, fetchJob)
	sched.Start(cfg)

	// HTTP server
	srv = newServer(logger, ring, sched, fetchJob, state)

	return
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
