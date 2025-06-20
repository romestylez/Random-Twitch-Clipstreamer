import requests
import json
from datetime import datetime, timedelta, timezone

# === Configuration ===
client_id = "your-client-id"                          # Your Twitch App's Client ID
client_secret = "your-secret-id"                      # Your Twitch App's Client Secret
target_channel = "channel_name"                       # Twitch channel to fetch clips from
days_back = 730                                       # Number of days to look back for clips
min_views = 250                                       # Only include clips with at least this many views
json_filename = f"{target_channel}_clips.json"        # Output JSON file with filtered clip URLs

# === Get Access Token using Client Credentials Flow ===
token_res = requests.post("https://id.twitch.tv/oauth2/token", data={
    "client_id": client_id,
    "client_secret": client_secret,
    "grant_type": "client_credentials"
})
access_token = token_res.json()["access_token"]

# === Prepare headers for Twitch API requests ===
headers = {
    "Client-ID": client_id,
    "Authorization": f"Bearer {access_token}"
}

# === Get user ID of the target channel ===
user_res = requests.get(f"https://api.twitch.tv/helix/users?login={target_channel}", headers=headers)
user_id = user_res.json()["data"][0]["id"]

# === Define time window for clip search ===
ended_at = datetime.now(timezone.utc)
started_at = ended_at - timedelta(days=days_back)

# Format timestamps to Twitch-compatible ISO strings (UTC)
started_at_str = started_at.isoformat(timespec='seconds').replace("+00:00", "Z")
ended_at_str = ended_at.isoformat(timespec='seconds').replace("+00:00", "Z")

# === Fetch clips from the Twitch API ===
clip_url = "https://api.twitch.tv/helix/clips"
params = {
    "broadcaster_id": user_id,
    "started_at": started_at_str,
    "ended_at": ended_at_str,
    "first": 100  # Max items per request
}

filtered_clips = []
while True:
    res = requests.get(clip_url, headers=headers, params=params)
    data = res.json()

    # Handle potential API errors
    if "data" not in data:
        print("Error fetching clips:", json.dumps(data, indent=2))
        break

    # Filter by minimum view count
    for clip in data["data"]:
        if clip.get("view_count", 0) >= min_views:
            filtered_clips.append(clip["url"])

    # Pagination handling
    if "pagination" in data and data["pagination"].get("cursor"):
        params["after"] = data["pagination"]["cursor"]
    else:
        break

# === Save filtered clip URLs to JSON file ===
with open(json_filename, "w", encoding="utf-8") as f:
    json.dump(filtered_clips, f, ensure_ascii=False, indent=2)

# === Print result summary and write to log ===
summary = (f"{len(filtered_clips)} clip URLs saved to {json_filename} "
           f"(min. {min_views} views, last {days_back} days: "
           f"from {started_at.date()} to {ended_at.date()})")

print(summary)

# === Write summary to log file ===
with open("get_clip_links.log", "w", encoding="utf-8") as log_file:
    log_file.write(summary + "\n")
