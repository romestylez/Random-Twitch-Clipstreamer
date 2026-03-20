# 🎥 Random Twitch Clipstreamer

Sammelt Twitch-Clips nach Zeitraum & Mindest-Views, extrahiert `.mp4`-Links **oder lädt die Clips herunter**, und spielt sie zufällig im Browser ab – mit Cooldown, History **und optionaler Anzeige des Clip-Datums**.

> **Neu: vollständig in Go neu geschrieben** – kein Python, kein PHP, kein lokaler Webserver nötig. Alles in einer einzigen ausführbaren Datei.

## 📦 Komponenten

- ⚙️ `twitch-clipstreamer` – einzelnes Binary mit allem drin:
  - Twitch-API-Fetcher (Clip-Links oder Download via [yt-dlp](https://github.com/yt-dlp/yt-dlp))
  - Eingebauter HTTP-Server (Player + Admin-UI + API)
  - System-Tray-Icon (Windows/Linux/macOS)
  - Optionaler Tages-Scheduler
- 🌐 `player.html` – HTML5-Player (eingebettet im Binary)
- 🛠️ `admin.html` – Web-Oberfläche zum Konfigurieren & manuellen Fetchen
- ⚙️ `config.json.example` – Beispiel-Konfiguration

## 🔧 Voraussetzungen

- Twitch Developer App (Client-ID & Client-Secret) → [dev.twitch.tv](https://dev.twitch.tv/console)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) – wird unter Windows automatisch heruntergeladen falls nicht vorhanden, unter Linux/macOS bitte manuell installieren:
  ```bash
  # Linux
  sudo apt install yt-dlp
  # oder pip
  pip install yt-dlp
  ```
- Sonst: **keine weiteren Abhängigkeiten**

## ⚙️ Installation & Konfiguration

1. Binary aus dem [Releases-Bereich](../../releases) herunterladen oder selbst bauen (siehe unten).
2. `config.json.example` nach `config.json` kopieren und anpassen:

```json
{
  "client_id": "DEIN_TWITCH_CLIENT_ID",
  "client_secret": "DEIN_TWITCH_CLIENT_SECRET",
  "channel_name": "kanalname",
  "days_back": 1095,
  "min_views": 300,
  "whitelist": "",
  "blacklist": "",
  "schedule_enabled": false,
  "schedule_hour": 8,
  "schedule_minute": 0,
  "download_mode": "download",
  "port": 42069,
  "download_dir": "Twitch_Clips"
}
```

| Feld | Beschreibung |
|---|---|
| `client_id` / `client_secret` | Twitch-App-Zugangsdaten |
| `channel_name` | Twitch-Kanalname |
| `days_back` | Zeitfenster in Tagen (z. B. 1095 = 3 Jahre) |
| `min_views` | Mindest-Views pro Clip |
| `whitelist` | Nur Clips dieser Kategorie, kommagetrennt (z. B. `irl`) |
| `blacklist` | Kategorien ausschließen, kommagetrennt (z. B. `just chatting,irl`) |
| `schedule_enabled` | Automatischer täglicher Fetch |
| `schedule_hour` / `schedule_minute` | Uhrzeit des automatischen Fetchs |
| `download_mode` | `download` – Clips herunterladen · `local` – nur `.mp4`-Links sammeln |
| `port` | HTTP-Port (Standard: `42069`) |
| `download_dir` | Zielordner für heruntergeladene Clips |

Alternativ können `CLIENT_ID`, `CLIENT_SECRET`, `CHANNEL_NAME`, `DAYS_BACK`, `MIN_VIEWS`, `WHITELIST` und `BLACKLIST` auch per `.env`-Datei gesetzt werden (Fallback).

## ▶️ Starten

```bash
./twitch-clipstreamer
```

Das Binary wechselt automatisch in sein eigenes Verzeichnis, startet den HTTP-Server und öffnet ein System-Tray-Icon.

- **Player:** `http://localhost:42069/`
- **Admin-UI:** `http://localhost:42069/admin`

Über das Tray-Menü lässt sich die Admin-UI öffnen, ein manueller Fetch starten oder das Programm beenden.

## 📂 Output

| Datei | Inhalt |
|---|---|
| `<CHANNEL>_mp4_urls.json` | Clip-Liste für den Player |
| `clip_history.json` | Zuletzt gespielte Clips (Cooldown-Tracking) |
| `clip_date.html` | Aktuelles Clip-Datum (für OBS-Overlay) |
| `clipstreamer.log` | Protokoll (API, Downloads, Fehler) |
| `Twitch_Clips/` | Heruntergeladene `.mp4`-Dateien (nur `download`-Modus) |

**Beispiel-JSON-Schema:**
```json
[
  {
    "url": "https://...mp4",
    "date": "31.12.2024"
  }
]
```

✨ **Auto-Cleanup** (nur `download`-Modus): Clips außerhalb des konfigurierten Zeitfensters werden automatisch gelöscht.

## 🎬 HTML-Player

Der Player läuft direkt unter `http://localhost:42069/` – kein externer Webserver nötig.

### Funktionen

- ⏱️ **Cooldown pro Clip** (Standard: 240 Minuten, anpassbar in `player.html` → `const cooldownMinutes`)
- 📜 **History** in `clip_history.json`
- 🗓️ **Clip-Datum**: Der Player schreibt das aktuelle Clip-Datum per `POST /write_clipdate` → `clip_date.html`
  Diese Datei ist über `http://localhost:42069/clip_date.html` abrufbar und kann in OBS als Browser-Quelle für ein Datum-Overlay eingebunden werden.

## 🛠️ Admin-UI

Erreichbar unter `http://localhost:42069/admin`:

- Konfiguration live anpassen & speichern
- Manuellen Clip-Fetch starten
- Fetch-Status & Logs einsehen

## 🔨 Selbst bauen

```bash
cd go
go build -o twitch-clipstreamer .
```

Voraussetzung: [Go 1.21+](https://go.dev/dl/)

## 📜 Lizenz
Dieses Projekt darf frei verwendet, verändert und weitergegeben werden.

Voraussetzung ist, dass ein Verweis auf dieses Repository (z. B. durch einen Link) erfolgt.
