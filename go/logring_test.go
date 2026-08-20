package main

import (
	"testing"
	"time"
)

func TestFetchStateCancel(t *testing.T) {
	state := &FetchState{}
	ctx, started := state.TryStart()
	if !started {
		t.Fatal("first fetch did not start")
	}
	if _, started := state.TryStart(); started {
		t.Fatal("second concurrent fetch unexpectedly started")
	}
	if !state.Cancel() {
		t.Fatal("running fetch was not cancelled")
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("fetch context was not cancelled")
	}

	state.Finish("")
	if state.Cancel() {
		t.Fatal("completed fetch reported as cancellable")
	}
}
