import sys
import os
import json
import subprocess
import requests
from datetime import datetime, timedelta, UTC
from pathlib import Path
import logging

# Optional: Playwright für /local
try:
    from playwright.sync_api import sync_playwright
except ImportError:
    sync_playwright = None

# --- Logging setup ---
log_file = "fetch_clips_and_mp4.log"
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.FileHandler(log_file, encoding="utf-8"),
        logging.StreamHandler(sys.stdout)
    ]
)
log = logging.getLogger()

# --- Configuration ---
CLIENT_ID = "YOUR-CLIENT-ID"
CLIENT_SECRET = "YOUR-CLIENT-SECRET"
CHANNEL_NAME = "YOUR-CHANNEL"
DAYS_BACK = 30
MIN_VIEWS = 500
DOWNLOAD_DIR = "Twitch_Clips"
OUTPUT_FILE = f"{CHANNEL_NAME}_mp4_urls.json"

def get_oauth_token(client_id, client_secret):
    url = 'https://id.twitch.tv/oauth2/token'
    params = {
        'client_id': client_id,
        'client_secret': client_secret,
        'grant_type': 'client_credentials'
    }
    response = requests.post(url, params=params)
    response.raise_for_status()
    return response.json()['access_token']

def get_user_id(channel_name, client_id, access_token):
    url = f'https://api.twitch.tv/helix/users?login={channel_name}'
    headers = {
        'Client-ID': client_id,
        'Authorization': f'Bearer {access_token}'
    }
    response = requests.get(url, headers=headers)
    response.raise_for_status()
    data = response.json().get('data')
    return data[0]['id'] if data else None

def get_channel_clips(broadcaster_id, client_id, access_token, started_at, ended_at):
    url = 'https://api.twitch.tv/helix/clips'
    headers = {
        'Client-ID': client_id,
        'Authorization': f'Bearer {access_token}'
    }
    params = {
        'broadcaster_id': broadcaster_id,
        'started_at': started_at,
        'ended_at': ended_at,
        'first': 100
    }

    clips = []
    while True:
        response = requests.get(url, headers=headers, params=params)
        response.raise_for_status()
        data = response.json()
        clips.extend(data.get('data', []))
        cursor = data.get('pagination', {}).get('cursor')
        if not cursor:
            break
        params['after'] = cursor
    return [clip for clip in clips if clip.get("view_count", 0) >= MIN_VIEWS]

def fetch_mp4_url_with_playwright(clip_url):
    if sync_playwright is None:
        log.error("❌ Playwright is not installed. Use 'pip install playwright' and run 'playwright install'")
        return None

    with sync_playwright() as p:
        browser = p.firefox.launch(headless=True)
        context = browser.new_context()
        page = context.new_page()
        try:
            page.goto(clip_url, timeout=15000)
            page.wait_for_selector("video", timeout=15000)
            mp4_url = page.eval_on_selector("video", "el => el.src")
            return mp4_url
        except Exception as e:
            log.warning(f"⚠️  Failed to get MP4 URL from {clip_url}: {e}")
            return None
        finally:
            context.close()
            browser.close()

def main():
    mode = sys.argv[1] if len(sys.argv) > 1 else "/local"
    is_download = mode.lower() == "/download"
    is_local = mode.lower() == "/local"

    if is_download and not os.path.exists(DOWNLOAD_DIR):
        os.makedirs(DOWNLOAD_DIR)

    log.info("🔑 Getting OAuth token...")
    token = get_oauth_token(CLIENT_ID, CLIENT_SECRET)

    log.info(f"👤 Getting user ID for {CHANNEL_NAME}...")
    user_id = get_user_id(CHANNEL_NAME, CLIENT_ID, token)
    if not user_id:
        log.error("❌ Failed to get user ID.")
        return

    now = datetime.now(UTC)
    start_time = (now - timedelta(days=DAYS_BACK)).isoformat("T")
    end_time = now.isoformat("T")

    log.info(f"🎯 Fetching clips from the last {DAYS_BACK} days...")
    clips = get_channel_clips(user_id, CLIENT_ID, token, start_time, end_time)
    log.info(f"✅ Found {len(clips)} clips with at least {MIN_VIEWS} views.")

    downloaded = []
    planned_ids = set()

    for clip in clips:
        slug = clip["id"]
        planned_ids.add(slug)
        if is_local:
            log.info(f"🔍 Extracting MP4 URL for: {slug}")
            mp4_url = fetch_mp4_url_with_playwright(clip["url"])
            if mp4_url:
                downloaded.append(mp4_url)
            continue

        # /download-Modus
        upload_date = clip.get("created_at", "")[:10].replace("-", "")
        filename = f"{upload_date}_{slug}.mp4"
        filepath = Path(DOWNLOAD_DIR) / filename
        if filepath.exists():
            log.info(f"✔️  Already exists: {filename}")
            downloaded.append(str(filepath).replace("\\", "/"))
            continue

        log.info(f"⬇️  Downloading: {slug}")
        yt_dlp_cmd = [
            'yt-dlp',
            '--paths', DOWNLOAD_DIR,
            '-o', filename,
            '--no-warnings',
            '--continue',
            '--ignore-errors',
            clip["url"]
        ]
        try:
            subprocess.run(yt_dlp_cmd, check=True, capture_output=True, text=True)
            if filepath.exists():
                downloaded.append(str(filepath).replace("\\", "/"))
            else:
                log.warning(f"⚠️  Could not confirm download of: {slug}")
        except subprocess.CalledProcessError as e:
            log.warning(f"⚠️  Failed to download {slug}: {e}")

    if is_download:
        log.info("🧹 Removing obsolete files...")
        all_files = list(Path(DOWNLOAD_DIR).glob("*.mp4"))
        for file in all_files:
            if not any(file.stem.endswith(clip_id) for clip_id in planned_ids):
                try:
                    file.unlink()
                    log.info(f"🗑️  Deleted: {file.name}")
                except Exception as e:
                    log.warning(f"⚠️  Could not delete {file.name}: {e}")

    log.info(f"💾 Writing output to {OUTPUT_FILE}...")
    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        json.dump(downloaded, f, indent=2, ensure_ascii=False)

    log.info(f"✅ Done. {len(downloaded)} files listed.")

if __name__ == "__main__":
    main()
