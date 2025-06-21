import asyncio
import json
import time
from pathlib import Path
from datetime import datetime
from playwright.async_api import async_playwright

# === Configuration ===
target_channel = "smtxlost"  # <-- set your Twitch channel here

input_file = Path(f"{target_channel}_clips.json")
output_file = Path(f"{target_channel}_mp4_urls.json")
log_file = Path("get_mp4_links.log")
concurrency = 10

# === Helper ===
def format_date_de(dt):
    return dt.strftime("%d.%m.%Y %H:%M:%S")

def load_urls_from_json(path):
    with open(path, "r", encoding="utf-8") as f:
        raw = json.load(f)
        return [entry if isinstance(entry, str) else entry.get("url") for entry in raw if entry]

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
        await page.goto(clip_url, wait_until="networkidle", timeout=40000)
        await page.evaluate("window.scrollBy(0, window.innerHeight)")
        await page.wait_for_timeout(4000)
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

    clip_urls = load_urls_from_json(input_file)
    results = []
    errors = []

    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        sem = asyncio.Semaphore(concurrency)

        async def sem_task(i, url):
            async with sem:
                await process_clip(url, i, browser, results, errors)

        await asyncio.gather(*(sem_task(i, url) for i, url in enumerate(clip_urls)))
        await browser.close()

    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(results, f, indent=2, ensure_ascii=False)

    end_dt = datetime.now()
    duration = round(time.time() - start_time, 1)

    # === Console summary (no error count here) ===
    summary = f"\n✅ Done! Found {len(results)} of {len(clip_urls)} MP4 links.\n"
    summary += f"⏱️  Duration: {duration} seconds\n"

    # === Log output with full error info ===
    log_text = f"End: {format_date_de(end_dt)}\nDuration: {duration} seconds\n"
    log_text += f"Found MP4 links: {len(results)} of {len(clip_urls)}\n"
    if errors:
        log_text += f"\nErrors ({len(errors)}):\n" + "\n".join(errors) + "\n"
    log_text += summary

    with open(log_file, "a", encoding="utf-8") as log:
        log.write(log_text)

    print(summary)

if __name__ == "__main__":
    asyncio.run(main())
