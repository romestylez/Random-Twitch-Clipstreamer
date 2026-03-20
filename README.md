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

## 🐳 Docker

Die einfachste Möglichkeit, den Clipstreamer zu betreiben – kein Go, kein yt-dlp, kein ffmpeg manuell installieren. Alles ist im Image enthalten.

### Voraussetzungen

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Windows/macOS) oder Docker + Docker Compose (Linux)

### Schnellstart

1. Einen neuen Ordner anlegen, z. B. `clipstreamer`
2. Darin eine Datei `docker-compose.yml` erstellen mit folgendem Inhalt:

```yaml
services:
  clipstreamer:
    image: ghcr.io/romestylez/random-twitch-clipstreamer:latest
    ports:
      - "42069:42069"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

3. Container starten:

```bash
docker compose up -d
```

Docker lädt das Image automatisch herunter – kein manuelles Bauen nötig.

- **Player:** `http://localhost:42069/`
- **Admin-UI:** `http://localhost:42069/admin`

### Konfiguration

Beim ersten Start ist noch keine `config.json` vorhanden. Einfach die Admin-UI unter `http://localhost:42069/admin` öffnen, dort alle Felder ausfüllen und speichern – die Datei wird automatisch unter `./data/config.json` angelegt.

Alternativ kann die Konfiguration manuell angelegt werden:

```bash
# data-Ordner anlegen
mkdir data

# config.json.example als Vorlage kopieren und anpassen
cp config.json.example data/config.json
```

Alle persistenten Daten (Konfiguration, heruntergeladene Clips, Logs) landen im `./data`-Ordner neben der `docker-compose.yml`.

### Daten & Volumes

| Pfad im Container | Inhalt |
|---|---|
| `/data/config.json` | Konfigurationsdatei |
| `/data/Twitch_Clips/` | Heruntergeladene `.mp4`-Dateien |
| `/data/clipstreamer.log` | Logdatei |
| `/data/*.json` | Clip-Listen, History |

### Nützliche Befehle

```bash
# Container starten
docker compose up -d

# Logs live ansehen
docker compose logs -f

# Container stoppen
docker compose down

# Image auf neueste Version aktualisieren
docker compose pull
docker compose up -d
```

### Image manuell bauen

Wer das Image lieber selbst bauen möchte, einfach `image:` in der `docker-compose.yml` durch `build: .` ersetzen (Dockerfile liegt im Repo):

```yaml
services:
  clipstreamer:
    build: .
    ports:
      - "42069:42069"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

```bash
docker compose up -d --build
```

---

## 🔨 Selbst bauen

```bash
cd go
go build -o twitch-clipstreamer .
```

Voraussetzung: [Go 1.21+](https://go.dev/dl/)

## 📜 Lizenz
Dieses Projekt darf frei verwendet, verändert und weitergegeben werden.

Voraussetzung ist, dass ein Verweis auf dieses Repository (z. B. durch einen Link) erfolgt.
