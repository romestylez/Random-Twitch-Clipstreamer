//go:build headless

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger, _, srv, _, _, _ := commonSetup()

	// Run HTTP server in background
	go func() {
		port := GetConfig().Port
		if err := srv.ListenAndServe(port); err != nil {
			logger.Printf("❌ HTTP server error: %v", err)
		}
	}()

	// No system tray in headless/Docker mode — wait for SIGTERM or SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	logger.Printf("👋 Received signal %s — shutting down.", sig)
}
