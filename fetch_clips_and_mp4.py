import requests
import json
import asyncio
import time
from datetime import datetime, timedelta, timezone
from playwright.async_api import async_playwright

# === Configuration ===
client_id = "your-client-id"                     # Replace with your Twitch client ID
client_secret = "your-client-secret"             # Replace with your Twitch client secret
target_channel = "target_channel"                # Twitch channel to fetch clips from
days_back = 30                                   # Only fetch clips from the last N days
min_views = 250                                  # Minimum number of views per clip
concurrency = 10                                 # Number of parallel browser sessions
output_file = f"{target_channel}_mp4_urls.json"  # Output file for MP4 links
log_file = "fetch_clips_and_mp4.log"             # Log file for status and errors

# === Twitch Access Token ===
token_res = requests.post("https://id.twitch.tv/oauth2/token", data={
    "client_id": client_id,
    "client_secret": client_secret,
    "grant_type": "client_credentials"
})
access_token = token_res.json()["access_token"]
headers = {
    "Client-ID": client_id,
    "Authorization": f"Bearer {access_token}"
}

# === Get broadcaster ID ===
user_res = requests.get(f"https://api.twitch.tv/helix/users?login={target_channel}", headers=headers)
user_id = user_res.json()["data"][0]["id"]

# === Get clip URLs ===
ended_at = datetime.now(timezone.utc)
started_at = ended_at - timedelta(days=days_back)
started_at_str = started_at.isoformat(timespec='seconds').replace("+00:00", "Z")
ended_at_str = ended_at.isoformat(timespec='seconds').replace("+00:00", "Z")

clip_url = "https://api.twitch.tv/helix/clips"
params = {
    "broadcaster_id": user_id,
    "started_at": started_at_str,
    "ended_at": ended_at_str,
    "first": 100
}

filtered_clips = []
while True:
    res = requests.get(clip_url, headers=headers, params=params)
    data = res.json()
    if "data" not in data:
        print("Error fetching clips:", json.dumps(data, indent=2))
        break
    for clip in data["data"]:
        if clip.get("view_count", 0) >= min_views:
            filtered_clips.append(clip["url"])
    if "pagination" in data and data["pagination"].get("cursor"):
        params["after"] = data["pagination"]["cursor"]
    else:
        break

# === MP4 Extraction ===
def format_date_de(dt):
    return dt.strftime("%d.%m.%Y %H:%M:%S")

async def process_clip(clip_url, index, browser, results, errors):
    mp4_url = None
    context = await browser.new_context(
        viewport={"width": 1280, "height": 720},
        user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
    )
    page = await context.new_page()

    def handle_response(response):
        nonlocal mp4_url
        try:
            ct = response.headers.get("content-type", "")
            if ct.startswith("video/mp4") and ".mp4" in response.url:
                mp4_url = response.url
        except Exception:
            pass

    page.on("response", handle_response)

    try:
        await page.goto(clip_url, wait_until="networkidle", timeout=30000)

        for _ in range(6):  # bis zu 6 Sekunden nach networkidle
            if mp4_url:
                break
            await page.wait_for_timeout(1000)

    except Exception:
        errors.append(f"[{index+1}] Failed to load: {clip_url}")

    if mp4_url:
        results.append(mp4_url)
    else:
        errors.append(f"[{index+1}] No MP4 found: {clip_url}")
    await context.close()

async def main():
    start_time = time.time()
    start_dt = datetime.now()
    with open(log_file, "w", encoding="utf-8") as log:
        log.write(f"Start: {format_date_de(start_dt)}\n")

    results = []
    errors = []

    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        sem = asyncio.Semaphore(concurrency)

        async def sem_task(i, url):
            async with sem:
                await process_clip(url, i, browser, results, errors)

        await asyncio.gather(*(sem_task(i, url) for i, url in enumerate(filtered_clips)))
        await browser.close()

    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(results, f, indent=2, ensure_ascii=False)

    end_dt = datetime.now()
    duration = round(time.time() - start_time, 1)
    summary = f"\n✅ Done! Found {len(results)} of {len(filtered_clips)} MP4 links.\n"
    summary += f"⏱️  Duration: {duration} seconds\n"

    log_text = f"End: {format_date_de(end_dt)}\nDuration: {duration} seconds\n"
    log_text += f"Found MP4 links: {len(results)} of {len(filtered_clips)}\n"
    if errors:
        log_text += f"\nErrors ({len(errors)}):\n" + "\n".join(errors) + "\n"
    log_text += summary

    with open(log_file, "a", encoding="utf-8") as log:
        log.write(log_text)

    print(summary)

if __name__ == "__main__":
    asyncio.run(main())
