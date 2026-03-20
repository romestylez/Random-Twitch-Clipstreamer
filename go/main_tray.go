//go:build !headless

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

func main() {
	logger, _, srv, _, _, fetchJob := commonSetup()

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
