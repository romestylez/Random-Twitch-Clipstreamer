package main

import "testing"

func TestActiveClipFile(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected string
	}{
		{
			name:     "defaults to current channel",
			cfg:      Config{ChannelName: "benjamool"},
			expected: "benjamool_mp4_urls.json",
		},
		{
			name:     "uses admin selection",
			cfg:      Config{ChannelName: "benjamool", PlayerClipFile: "other_mp4_urls.json"},
			expected: "other_mp4_urls.json",
		},
		{
			name:     "rejects path traversal",
			cfg:      Config{ChannelName: "benjamool", PlayerClipFile: "../other_mp4_urls.json"},
			expected: "benjamool_mp4_urls.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ActiveClipFile(); got != tt.expected {
				t.Fatalf("ActiveClipFile() = %q, want %q", got, tt.expected)
			}
		})
	}
}
