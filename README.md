# 🎲 Random Twitch Clipstreamer

A lightweight toolset to fetch Twitch clips by view count and time range, 
extract their direct `.mp4` URLs, and stream them randomly with cooldown logic 
to avoid repeated playback.

Includes:
1. One Python script that fetches and extracts MP4 URLs in one step
2. A customizable HTML5 player that shuffles clips and prevents repeats (via server-side cooldown)

---

## 🎯 Features

- Uses the official Twitch API (Helix) to retrieve clips
- Filters clips by minimum view count and time range
- Extracts direct `.mp4` links using Playwright (headless Chromium)
- Saves links to a JSON file for use in HTML players
- Detailed logging of all steps

---

## 🔧 Requirements

- Python 3.8+ (Download for Windows: https://www.python.org/downloads/windows/)
  - ⚠️ Be sure to check **"Add Python to PATH"** during installation
- Internet connection (API + browser scraping)
- Twitch Developer Application (for `client_id` and `client_secret`)

---

## 📦 Installation

```bash
git clone https://github.com/yourname/Random-Twitch-Clipstreamer.git
cd Random-Twitch-Clipstreamer
pip install requests playwright
python -m playwright install
```

---

## 📂 Files Overview

| File                          | Purpose                                                  |
|-------------------------------|----------------------------------------------------------|
| `fetch_clips_and_mp4.py`      | Combines fetching + MP4 extraction in one efficient run  |
| `*_mp4_urls.json`             | Output list of `.mp4` URLs (named after the channel)     |
| `player.html`                 | Optional HTML5 player using cooldown logic via PHP       |
| `get_lastplayed.php`          | Reads playback history                                   |
| `save_lastplayed.php`         | Writes new playback timestamps                           |

---

## 🔑 Twitch API Setup

1. Go to [Twitch Developer Console](https://dev.twitch.tv/console/apps)
2. Register a new app (OAuth redirect: `http://localhost`)
3. Copy your `client_id` and `client_secret`
4. Edit them inside `fetch_clips_and_mp4.py`:
```python
client_id = "your-client-id"
client_secret = "your-client-secret"
```

---

## ⚙️ Configuration

Edit `fetch_clips_and_mp4.py`:
```python
target_channel = "your_channel"
days_back = 30
min_views = 250
concurrency = 10
```

---

## 🚀 Usage

Run everything in one step:
```bash
python fetch_clips_and_mp4.py
```

This will:
- Fetch Twitch clips for the target channel
- Open each clip in headless browser
- Extract `.mp4` links
- Save them to `<channel>_mp4_urls.json`

---

## 🕸️ Hosting & Clip Cooldown (player.html)

To use `player.html` with cooldown functionality, you need a local or remote **web server with PHP support**.

### Why a web server?

The clip player tracks recently played clips to prevent repetitions (e.g. no replays within 4 hours).  
This information is stored in a file called `clip_history.json` and managed via:

- `get_lastplayed.php` – loads the current play history
- `save_lastplayed.php` – saves updated playback times

JavaScript in the browser alone can't write files, so this requires a small PHP backend.

### Setup Instructions

1. Install a PHP web server (e.g. [Laragon](https://laragon.org) for Windows or Apache with PHP)
2. Place these files in your web root:
   - `player.html`
   - `get_lastplayed.php`
   - `save_lastplayed.php`
   - `clip_mp4_urls.json` (your clip list)
3. When loading `player.html`, it will:
   - Shuffle the clip list
   - Skip clips that were played recently (based on cooldown time)
   - Save new playback timestamps to `clip_history.json`

### 🔗 Player URL with custom clip list

You can load a custom MP4 list in `player.html` by passing a query parameter:

```
player.html?clips=yourchannel_mp4_urls.json
```

**Example:**

```
https://yourdomain.com/player.html?clips=yourchannel_mp4_urls.json
```

If no `clips` parameter is provided, the default `clip_mp4_urls.json` will be used.


### Cooldown Configuration

Inside `player.html`, adjust this line to control how long a clip is blocked after being shown:

```js
const cooldownMinutes = 240; // prevent replay for 4 hours
```

Older entries are automatically removed from `clip_history.json` to keep it small and efficient.

---

## 🖥️ Optional: Play MP4s in Browser

Open `player.html` in a browser via a local server:

```bash
# Python 3.x simple HTTP server (Linux/macOS/WSL)
python3 -m http.server 8000
```

Then visit [http://localhost:8000/player.html](http://localhost:8000/player.html)

Make sure `clip_mp4_urls.json` is in the same directory.

---

## 📝 Log Files

- `get_clip_links.log`: Created fresh on each run, shows how many clips were saved and from which date range.
- `get_mp4_links.log`: Appends on each run, showing duration, found `.mp4` links, and any errors.

---

## 📄 License

MIT – feel free to modify, share, or extend this project.

---

## 🙋‍♂️ Support

Issues, suggestions, or improvements? Feel free to open a GitHub issue or pull request.

