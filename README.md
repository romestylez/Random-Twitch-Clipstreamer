# 🎲 Random Twitch Clipstreamer

A lightweight tool to collect Twitch clips based on view count and date range, extract `.mp4` links or download the videos directly, and play them randomly in a browser player with cooldown logic.

## 🧩 Components

- `fetch_clips_and_mp4.py`: Twitch API clip fetcher with two modes:
  - `/local`: Extracts only the `.mp4` links and stores them in a JSON file
  - `/download`: Downloads clips using `yt-dlp` and stores local file paths in the JSON

- `player.html`: HTML5-based random clip player
- `get_lastplayed.php` / `save_lastplayed.php`: Handle playback history server-side

## 🛠 Requirements

- Python 3.8+
- Twitch Developer Application (`client_id`, `client_secret`)
- `yt-dlp` (for downloading):
  ```bash
  pip install yt-dlp
  ```

- Playwright (only needed for `/local` mode to extract `.mp4` links):
  ```bash
  pip install playwright
  python -m playwright install
  ```
- Python Requests
  ```bash
     pip install requests
  ```


## ⚙️ Usage

### Mode 1 – Extract mp4 links only

```bash
python fetch_clips_and_mp4.py /local
```

- Uses Playwright to extract `.mp4` links
- Saves results to a JSON file

### Mode 2 – Download clips

```bash
python fetch_clips_and_mp4.py /download
```

- Downloads all clips as `.mp4` files using `yt-dlp`
- Generates a JSON file with the local file paths

> If no parameter is passed, `/download` is used by default.

## 🧹 Auto Cleanup

In `/download` mode, any local clip that is **no longer within the defined timeframe** (e.g. older than 30 days) will be automatically deleted. This keeps your `Twitch_Clips` folder small and up to date.

## 🔧 Configuration

In `fetch_clips_and_mp4.py`:

```python
CHANNEL_NAME = "smtxlost"
DAYS_BACK = 30
MIN_VIEWS = 250
```

- `DAYS_BACK`: Time window in days (e.g. 730 = 2 years)
- `MIN_VIEWS`: Minimum number of views for a clip to be considered

## 📦 Output

- `smtxlost_mp4_urls.json`: List of `.mp4` links or file paths
- `fetch_clips_and_mp4.log`: Detailed log of API responses, downloads, and deletions

## 🎬 HTML Player

Use a local webserver (e.g. Laragon) to serve the following files:

- `player.html`
- `get_lastplayed.php`
- `save_lastplayed.php`
- `smtxlost_mp4_urls.json`

### Start the player:

```text
http://localhost/player.html?clips=smtxlost_mp4_urls.json
```

- Cooldown is enforced per clip (default: 240 minutes)
- History is saved in `clip_history.json`

## 📄 License

MIT – free for personal and commercial use.
