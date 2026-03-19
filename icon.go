package main

import _ "embed"

// trayIcon is loaded from icon.ico (required by systray on Windows).
// To change the icon: replace go/icon.ico and rebuild.
// Source PNG is kept as go/icon.png for reference.
//
//go:embed icon.ico
var trayIcon []byte
