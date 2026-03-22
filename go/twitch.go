package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Clip represents a Twitch clip from the Helix API.
type Clip struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	ViewCount int       `json:"view_count"`
	GameID    string    `json:"game_id"`
	Title     string    `json:"title"`
}

// GetOAuthToken retrieves a client-credentials OAuth token from Twitch.
func GetOAuthToken(clientID, clientSecret string) (string, error) {
	params := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
	}
	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", params)
	if err != nil {
		return "", fmt.Errorf("oauth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oauth status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode oauth response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}
	return result.AccessToken, nil
}

// GetUserID resolves a channel login name to a Twitch broadcaster ID.
func GetUserID(channelName, clientID, token string) (string, error) {
	req, _ := http.NewRequest("GET",
		"https://api.twitch.tv/helix/users?login="+url.QueryEscape(channelName), nil)
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get user status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode user response: %w", err)
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("channel %q not found", channelName)
	}
	return result.Data[0].ID, nil
}

// GetClips fetches all clips for a broadcaster in the given time window,
// filtered by minimum view count and optional whitelist/blacklist of game names.
func GetClips(broadcasterID, clientID, token string, startedAt, endedAt time.Time, minViews int, whitelist, blacklist []string) ([]Clip, error) {
	baseURL := "https://api.twitch.tv/helix/clips"
	params := url.Values{
		"broadcaster_id": {broadcasterID},
		"started_at":     {startedAt.UTC().Format(time.RFC3339)},
		"ended_at":       {endedAt.UTC().Format(time.RFC3339)},
		"first":          {"100"},
	}

	var clips []Clip
	cursor := ""

	for {
		if cursor != "" {
			params.Set("after", cursor)
		}
		reqURL := baseURL + "?" + params.Encode()

		req, _ := http.NewRequest("GET", reqURL, nil)
		req.Header.Set("Client-ID", clientID)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch clips page: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("clips status %d: %s", resp.StatusCode, body)
		}

		var page struct {
			Data []struct {
				ID        string    `json:"id"`
				URL       string    `json:"url"`
				CreatedAt time.Time `json:"created_at"`
				ViewCount int       `json:"view_count"`
				GameID    string    `json:"game_id"`
				Title     string    `json:"title"`
			} `json:"data"`
			Pagination struct {
				Cursor string `json:"cursor"`
			} `json:"pagination"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode clips page: %w", err)
		}
		resp.Body.Close()

		for _, d := range page.Data {
			if d.ViewCount < minViews {
				continue
			}
			clips = append(clips, Clip{
				ID:        d.ID,
				URL:       d.URL,
				CreatedAt: d.CreatedAt,
				ViewCount: d.ViewCount,
				GameID:    d.GameID,
				Title:     d.Title,
			})
		}

		cursor = page.Pagination.Cursor
		if cursor == "" {
			break
		}
	}

	// Apply whitelist/blacklist only when needed.
	// The clips API does not return game_name, so we resolve game_ids first.
	if len(whitelist) > 0 || len(blacklist) > 0 {
		gameNames, err := resolveGameNames(clips, clientID, token)
		if err != nil {
			return nil, fmt.Errorf("resolve game names: %w", err)
		}
		filtered := clips[:0]
		for _, c := range clips {
			if matchesCategory(gameNames[c.GameID], whitelist, blacklist) {
				filtered = append(filtered, c)
			}
		}
		clips = filtered
	}

	return clips, nil
}

// resolveGameNames calls /helix/games for all unique game_ids found in clips
// and returns a map[gameID]gameName.
func resolveGameNames(clips []Clip, clientID, token string) (map[string]string, error) {
	seen := make(map[string]bool)
	var ids []string
	for _, c := range clips {
		if c.GameID != "" && !seen[c.GameID] {
			seen[c.GameID] = true
			ids = append(ids, c.GameID)
		}
	}
	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(ids))
	for len(ids) > 0 {
		batch := ids
		if len(batch) > 100 {
			batch, ids = ids[:100], ids[100:]
		} else {
			ids = nil
		}

		params := url.Values{}
		for _, id := range batch {
			params.Add("id", id)
		}
		req, _ := http.NewRequest("GET", "https://api.twitch.tv/helix/games?"+params.Encode(), nil)
		req.Header.Set("Client-ID", clientID)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("get games: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("games status %d: %s", resp.StatusCode, body)
		}

		var page struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode games: %w", err)
		}
		resp.Body.Close()

		for _, g := range page.Data {
			result[g.ID] = g.Name
		}
	}
	return result, nil
}

// matchesCategory returns true if the gameName passes whitelist/blacklist filters.
// whitelist: if non-empty, gameName must be in the list.
// blacklist: if non-empty, gameName must NOT be in the list.
func matchesCategory(gameName string, whitelist, blacklist []string) bool {
	name := strings.ToLower(strings.TrimSpace(gameName))

	if len(whitelist) > 0 {
		for _, w := range whitelist {
			if strings.ToLower(strings.TrimSpace(w)) == name {
				return true
			}
		}
		return false
	}

	if len(blacklist) > 0 {
		for _, b := range blacklist {
			if strings.ToLower(strings.TrimSpace(b)) == name {
				return false
			}
		}
	}

	return true
}

// splitCategories splits a comma-separated category string into a slice,
// ignoring empty entries.
func splitCategories(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
