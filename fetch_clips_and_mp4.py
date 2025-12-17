import sys
import os
import json
import subprocess
import requests
import asyncio
from datetime import datetime, timedelta, UTC
from pathlib import Path
import logging
from dotenv import load_dotenv  # <--- NEU

# --- .env laden ---
# pip install python-dotenv
load_dotenv()

# Optional: Playwright für /local
try:
    from playwright.async_api import async_playwright
except ImportError:
    async_playwright = None

# --- Logging setup ---
log_file = "fetch_clips_and_mp4.log"
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[
        logging.FileHandler(log_file, mode="w", encoding="utf-8"),
        logging.StreamHandler(sys.stdout)
    ]
)
log = logging.getLogger()

# --- Configuration ---
CLIENT_ID = os.getenv("CLIENT_ID")
CLIENT_SECRET = os.getenv("CLIENT_SECRET")
CHANNEL_NAME = os.getenv("CHANNEL_NAME")

DAYS_BACK = 1095
MIN_VIEWS = 300
DOWNLOAD_DIR = "Twitch_Clips"
OUTPUT_FILE = f"{CHANNEL_NAME}_mp4_urls.json" if CHANNEL_NAME else "mp4_urls.json"
CONCURRENCY = 5

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

async def fetch_mp4(clip_url, slug, semaphore):
    if async_playwright is None:
        log.error("❌ Playwright is not installed. Use 'pip install playwright' and run 'playwright install'")
        return None
    async with semaphore:
        log.info(f"🔍 Extracting MP4 URL for: {slug}")
        try:
            async with async_playwright() as p:
                browser = await p.firefox.launch(headless=True)
                context = await browser.new_context()
                page = await context.new_page()
                try:
                    await page.goto(clip_url, timeout=10000)
                    mp4_url = await page.evaluate("document.querySelector('video')?.src")
                    if not mp4_url:
                        await page.wait_for_selector("video", timeout=4000)
                        mp4_url = await page.eval_on_selector("video", "el => el.src")
                    return mp4_url
                finally:
                    await context.close()
                    await browser.close()
        except Exception as e:
            log.warning(f"⚠️  {slug} failed: {e}")
            return None

async def main_async():
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

    if is_local:
        semaphore = asyncio.Semaphore(CONCURRENCY)
        tasks = [fetch_mp4(clip["url"], clip["id"], semaphore) for clip in clips]
        results = await asyncio.gather(*tasks)
        downloaded = [r for r in results if r]
    else:
        for clip in clips:
            slug = clip["id"]
            planned_ids.add(slug)
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

    final_data = []

    for clip in clips:
        slug = clip["id"]
        date_str = clip.get("created_at", "")[:10]  # "2023-07-05"
        upload_date = date_str.replace("-", "")
        filename = f"{upload_date}_{slug}.mp4"
        filepath = str(Path(DOWNLOAD_DIR) / filename).replace("\\", "/")

        if filepath in downloaded:
            final_data.append({
                "url": filepath,
                "date": date_str
            })

    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        json.dump(final_data, f, indent=2, ensure_ascii=False)

    log.info(f"✅ Done. {len(final_data)} files listed.")

def main():
    asyncio.run(main_async())

if __name__ == "__main__":
    main()
