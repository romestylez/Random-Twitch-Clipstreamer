# 🎲 Random Twitch Clipstreamer

A lightweight toolset for fetching Twitch clips based on view count and time range, 
extracting their direct `.mp4` URLs, and streaming them randomly with cooldown logic 
to avoid repeated playback.

Includes:
1. Python scripts to collect and filter Twitch clips.
2. Automated MP4 URL extraction using headless browser automation.
3. A customizable HTML5 player that shuffles clips and prevents repeats (via server-side cooldown).


## 🎯 Features

- Uses the official Twitch API (Helix) to retrieve clips
- Filters clips by minimum view count and time range
- Extracts direct `.mp4` video links using Playwright (headless Chromium)
- Saves the links to a JSON file for use in HTML/players
- Detailed logging for each step

---

## 🔧 Requirements

- Python 3.8+ (Download for Windows: https://www.python.org/downloads/windows/)
  - ⚠️ Be sure to check **"Add Python to PATH"** during installation
- Internet connection (API + browser scraping)
- Twitch Developer Application (for `client_id` and `client_secret`)

---

## 📦 Installation

### 1. Clone the repository

```bash
git clone https://github.com/romestylez/Random-Twitch-Clipstreamer.git
cd Random-Twitch-Clipstreamer
```

### 2. Install dependencies

```bash
pip install requests playwright
python -m playwright install
```

> This installs the required Chromium browser engine used for scraping.

---

## 📂 Files Overview

| File                  | Purpose                                            |
|-----------------------|----------------------------------------------------|
| `get_clip_links.py`   | Uses the Twitch API to fetch and filter clip URLs |
| `get_mp4_links.py`    | Launches a headless browser to extract MP4 links  |
| `*_clips.json`         | Output list of filtered Twitch clip URLs (named after `target_channel`) |
| `clip_mp4_urls.json`  | Output list of direct MP4 links                   |
| `player.html`         | Optional HTML player that consumes the MP4 list   |

---

## 🔑 Twitch Developer Setup

1. Go to [https://dev.twitch.tv/console/apps](https://dev.twitch.tv/console/apps)
2. Register a new application:
   - **Name:** Anything you like
   - **OAuth Redirect URL:** `http://localhost`
   - **Category:** Website Integration or Application
3. Copy your **Client ID** and **Client Secret**
4. Paste them into `get_clip_links.py`:
5. 
   ```python
   client_id = "your-client-id"
   client_secret = "your-client-secret"
   ```

---

## 🚀 Usage

### Step 1: Fetch Twitch Clip URLs

This script queries the Twitch API and saves clip links that meet your criteria (e.g. views ≥ 250 in the last 7 days).

```bash
python get_clip_links.py
```

Outputs:
- `*_clips.json` — list of valid Twitch clip URLs (named after `target_channel`)
- `get_clip_links.log` — summary of the results

### Step 2: Extract MP4 Links

This script loads each clip page in headless Chromium and captures the `.mp4` stream URL.

```bash
python get_mp4_links.py
```

Outputs:
- `clip_mp4_urls.json` — array of `.mp4` links
- `get_mp4_links.log` — details and errors (if any)

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

## ⚙️ Configuration

Adjust these settings in `get_clip_links.py`:

```python
target_channel = "your_channel_name"   # Twitch channel to fetch clips from
days_back = 7                 # Only clips from the last N days
min_views = 250              # Only include clips with at least N views
```

Adjust these settings in `get_mp4_links.py`:

```python
target_channel = "your_channel_name"   # Twitch channel to fetch clips from
```
---

## 📄 License

MIT – feel free to modify, share, or extend this project.

---

## 🙋‍♂️ Support

Issues, suggestions, or improvements? Feel free to open a GitHub issue or pull request.


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

### Cooldown Configuration

Inside `player.html`, adjust this line to control how long a clip is blocked after being shown:

```js
const cooldownMinutes = 240; // prevent replay for 4 hours
```

Older entries are automatically removed from `clip_history.json` to keep it small and efficient.
