package main

import (
	"sync"
	"time"
)

// LogRing is a fixed-size circular buffer of log lines that implements io.Writer.
type LogRing struct {
	mu    sync.RWMutex
	lines []string
	pos   int
	size  int
}

// NewLogRing creates a LogRing with the given capacity.
func NewLogRing(size int) *LogRing {
	return &LogRing{
		lines: make([]string, size),
		size:  size,
	}
}

// Write implements io.Writer — appends the line to the ring buffer.
func (r *LogRing) Write(p []byte) (int, error) {
	text := string(p)
	// log.Logger appends \n; trim trailing whitespace before storing
	for len(text) > 0 && (text[len(text)-1] == '\n' || text[len(text)-1] == '\r') {
		text = text[:len(text)-1]
	}
	if text == "" {
		return len(p), nil
	}
	r.mu.Lock()
	r.lines[r.pos] = text
	r.pos = (r.pos + 1) % r.size
	r.mu.Unlock()
	return len(p), nil
}

// Lines returns up to n most-recent log lines in chronological order.
func (r *LogRing) Lines(n int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if n > r.size {
		n = r.size
	}
	out := make([]string, 0, n)
	for i := 1; i <= r.size && len(out) < n; i++ {
		idx := (r.pos - i + r.size) % r.size
		if r.lines[idx] != "" {
			out = append([]string{r.lines[idx]}, out...)
		}
	}
	return out
}

// FetchState tracks the state of ongoing/completed fetch operations.
type FetchState struct {
	mu        sync.RWMutex
	running   bool
	lastRun   time.Time
	lastError string
}

// TryStart atomically marks a fetch as running.
// Returns false if a fetch is already in progress.
func (s *FetchState) TryStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	s.lastError = ""
	return true
}

// Finish marks the fetch as complete and records any error.
func (s *FetchState) Finish(errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	s.lastRun = time.Now()
	s.lastError = errMsg
}

// Snapshot returns a point-in-time copy of the fetch state.
func (s *FetchState) Snapshot() (running bool, lastRun time.Time, lastError string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running, s.lastRun, s.lastError
}
