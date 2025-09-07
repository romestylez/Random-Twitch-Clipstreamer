# 🎥 Random Twitch Clipstreamer

Sammelt Twitch-Clips nach Zeitraum & Mindest-Views, extrahiert `.mp4`-Links **oder lädt die Clips herunter**, und spielt sie zufällig im Browser ab – mit Cooldown, History **und optionaler Anzeige des Clip-Datums**.

## 📦 Komponenten
- 🐍 `fetch_clips_and_mp4.py` – Clip-Fetcher mit zwei Modi:
  - `/local` – nur `.mp4`-Links sammeln → JSON
  - `/download` – Clips via `yt-dlp` laden → lokale Pfade in JSON  
- 🌐 `player.html` – HTML5-Player mit Zufallsauswahl, Cooldown & History  
- 📜 `get_lastplayed.php` / `save_lastplayed.php` – vom Player genutzte Hilfsskripte zur Speicherung/Abruf des zuletzt gespielten Clips  
- 🗓️ `write_clipdate.php` – schreibt das **aktuelle Clip-Datum** (z. B. für Overlay/OBS) in eine Textdatei  
- ▶️ `start.bat` – Beispiel-Starter für Windows  
- ⚙️ `.env.example` – Beispiel-Konfiguration für API-Keys

## 🔧 Voraussetzungen
- Python 3.8+
- Twitch Developer App (Client-ID/Secret)
- `requests`, ggf. `playwright` (nur `/local`), `yt-dlp` (nur `/download`)
- Optional: lokaler Webserver (z. B. Laragon) für Player & PHP-Endpoints

```bash
pip install requests
# Für /local:
pip install playwright
python -m playwright install
# Für /download:
pip install yt-dlp
```

## ⚙️ Installation & Konfiguration
1. Repo klonen.  
2. `.env.example` nach `.env` kopieren und `CLIENT_ID`, `CLIENT_SECRET` usw. setzen.  
3. In `fetch_clips_and_mp4.py` Basiswerte anpassen:
   ```py
   CHANNEL_NAME = "dein_channel"
   DAYS_BACK = 30
   MIN_VIEWS = 250
   ```
   - `DAYS_BACK`: Zeitfenster in Tagen (z. B. 730 = 2 Jahre)  
   - `MIN_VIEWS`: minimale Views

## ▶️ Nutzung (Fetcher)
```bash
# Nur Links extrahieren:
python fetch_clips_and_mp4.py /local

# Clips herunterladen (Default, wenn kein Parameter):
python fetch_clips_and_mp4.py /download
```

✨ **Auto-Cleanup** (nur `/download`): Clips außerhalb des Zeitfensters werden automatisch gelöscht.

## 📂 Output
- `<CHANNEL>_mp4_urls.json` – Liste mit `.mp4`-Links oder lokalen Pfaden  
- `fetch_clips_and_mp4.log` – Protokoll (API, Downloads, Löschungen)

**Beispiel-JSON-Schema (vereinfacht):**
```json
[
  {
    "id": "ClipID",
    "title": "Clip-Titel",
    "url": "https://...mp4" ,
    "local_path": "Twitch_Clips/...",
    "created_at": "2024-12-31T23:59:59Z",
    "view_count": 1234
  }
]
```

## 🎬 HTML-Player
Dateien per Webserver bereitstellen:
- `player.html`, `get_lastplayed.php`, `save_lastplayed.php`, `<CHANNEL>_mp4_urls.json`

Start:
```
http://localhost/player.html?clips=<CHANNEL>_mp4_urls.json
```

### 🔑 Funktionen
- ⏱️ **Cooldown pro Clip** (Standard: 240 Minuten - anpassbar in der player.htlm -> const cooldownMinutes = 240;)  
- 📜 **History** in `clip_history.json`  
- 🗓️ **Clip-Datum anzeigen**: `player.html` liest `created_at` aus der JSON.  
  Optional schreibt `write_clipdate.php` das Datum in eine Textdatei → per OBS-Textquelle („Read from file“) im Stream einblendbar.

### 🧩 Hilfsskripte (Player-Rückrufe)
Diese PHP-Dateien werden **vom Player per `fetch()`** aufgerufen. Sie sind **keine öffentliche API**, sondern kleine Dateihilfen für den lokalen Player/OBS:
- `GET get_lastplayed.php` – liefert das zuletzt gespielte Element  
- `POST save_lastplayed.php` – speichert das zuletzt gespielte Element  
- `POST write_clipdate.php` – schreibt das aktuelle Clip-Datum in eine Textdatei (für OBS-Overlays nutzbar)

## 🖥️ Windows-Start
```bat
start.bat
```
> Startskript zum schnellen Aufruf der Standard-Befehle (anpassen nach Bedarf).

## 📜 Lizenz
MIT – frei für private & kommerzielle Nutzung.
