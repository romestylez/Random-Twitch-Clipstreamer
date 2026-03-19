package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

func main() {
	// Change working directory to the directory containing the executable,
	// so all relative file paths (config.json, Twitch_Clips/, etc.) work
	// regardless of how the binary is launched.
	if exePath, err := os.Executable(); err == nil {
		_ = os.Chdir(dirOf(exePath))
	}

	// Load configuration
	cfg := LoadConfig()

	// Set up fanout logger: writes to both log file and in-memory ring buffer
	ring := NewLogRing(200)
	logFile, err := os.OpenFile("clipstreamer.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logFile = os.Stderr
	}
	multi := io.MultiWriter(logFile, ring, os.Stdout)
	logger := log.New(multi, "", log.LstdFlags)

	logger.Println("🚀 Twitch Clipstreamer starting...")

	// Shared fetch state
	state := &FetchState{}

	// Fetch function used by scheduler and tray menu
	fetchJob := func() {
		if !state.TryStart() {
			logger.Println("⚠️  Fetch already running, skipping.")
			return
		}
		c := GetConfig()
		err := FetchAndWrite(c, logger)
		errStr := ""
		if err != nil {
			errStr = err.Error()
			logger.Printf("❌ Fetch error: %v", err)
		}
		state.Finish(errStr)
	}

	// Scheduler
	sched := NewScheduler(logger, fetchJob)
	sched.Start(cfg)

	// HTTP server
	srv := newServer(logger, ring, sched, fetchJob, state)

	// Run HTTP server in background
	go func() {
		port := GetConfig().Port
		if err := srv.ListenAndServe(port); err != nil {
			logger.Printf("❌ HTTP server error: %v", err)
		}
	}()

	// System tray (blocks until quit)
	systray.Run(onTrayReady(logger, fetchJob), onTrayExit)
}

func onTrayReady(logger *log.Logger, fetchJob func()) func() {
	return func() {
		systray.SetTitle("Twitch Clipstreamer")
		systray.SetTooltip(fmt.Sprintf("Twitch Clipstreamer — Port :%d", GetConfig().Port))

		// Try to set an icon (embedded PNG/ICO); fall back gracefully if missing
		systray.SetIcon(trayIcon)

		mAdmin := systray.AddMenuItem("Admin öffnen", "Öffnet die Admin-UI im Browser")
		mFetch := systray.AddMenuItem("Clips herunterladen", "Clips von Twitch herunterladen und aktualisieren")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Beenden", "Twitch Clipstreamer beenden")

		go func() {
			for {
				select {
				case <-mAdmin.ClickedCh:
					url := fmt.Sprintf("http://localhost:%d/admin", GetConfig().Port)
					openBrowser(url)
				case <-mFetch.ClickedCh:
					go fetchJob()
				case <-mQuit.ClickedCh:
					logger.Println("👋 Shutting down...")
					systray.Quit()
				}
			}
		}()
	}
}

func onTrayExit() {
	os.Exit(0)
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
