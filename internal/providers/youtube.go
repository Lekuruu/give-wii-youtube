package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// YouTubeProvider implements Provider using YouTube's Innertube API
type YouTubeProvider struct {
	ApiKey         string
	Http           *http.Client
	TrendingParams map[string]string
}

// Innertube API endpoints
const (
	InnertubePlayerUrl = "https://www.youtube.com/youtubei/v1/player"
	InnertubeSearchUrl = "https://www.youtube.com/youtubei/v1/search"
	InnertubeGroupUrl  = "https://www.youtube.com/youtubei/v1/browse"
	SuggestUrl         = "https://suggestqueries-clients6.youtube.com/complete/search"
)

// Client context for Innertube requests
var clientContext = map[string]interface{}{
	"client": map[string]interface{}{
		"clientName":    "WEB",
		"clientVersion": "2.20231221",
		"hl":            "en",
		"gl":            "US",
	},
}

// NewYouTubeProvider creates a new YouTube provider
func NewYouTubeProvider(apiKey string, trendingParams map[string]string) *YouTubeProvider {
	return &YouTubeProvider{
		Http:           &http.Client{Timeout: 30 * time.Second},
		ApiKey:         apiKey,
		TrendingParams: trendingParams,
	}
}

// performInnertubeRequest sends a POST request to an Innertube endpoint
func (p *YouTubeProvider) performInnertubeRequest(endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("X-YouTube-Client-Name", "1")
	req.Header.Set("X-YouTube-Client-Version", "2.20231221")

	resp, err := p.Http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// GetVideoInfo retrieves detailed information about a video
func (p *YouTubeProvider) GetVideoInfo(videoID string) (*VideoInfo, error) {
	payload := map[string]interface{}{
		"context": clientContext,
		"videoId": videoID,
	}

	data, err := p.performInnertubeRequest(InnertubePlayerUrl, payload)
	if err != nil {
		return nil, err
	}

	videoDetails, ok := data["videoDetails"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("videoDetails not found in response")
	}

	info := &VideoInfo{
		VideoID:       videoID,
		Title:         getString(videoDetails, "title"),
		Author:        getString(videoDetails, "author"),
		AuthorID:      getString(videoDetails, "channelId"),
		LengthSeconds: getInt(videoDetails, "lengthSeconds"),
		ViewCount:     getInt64(videoDetails, "viewCount"),
		Description:   getString(videoDetails, "shortDescription"),
		IsLive:        getBool(videoDetails, "isLive"),
	}

	// Get keywords
	if keywords, ok := videoDetails["keywords"].([]interface{}); ok {
		for _, kw := range keywords {
			if s, ok := kw.(string); ok {
				info.Keywords = append(info.Keywords, s)
			}
		}
	}

	// Get thumbnails
	if thumbnail, ok := videoDetails["thumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := thumbnail["thumbnails"].([]interface{}); ok {
			for _, t := range thumbs {
				if tm, ok := t.(map[string]interface{}); ok {
					info.Thumbnails = append(info.Thumbnails, Thumbnail{
						URL:    getString(tm, "url"),
						Width:  getInt(tm, "width"),
						Height: getInt(tm, "height"),
					})
				}
			}
		}
	}

	// Get published date from microformat
	if microformat, ok := data["microformat"].(map[string]interface{}); ok {
		if renderer, ok := microformat["playerMicroformatRenderer"].(map[string]interface{}); ok {
			info.PublishedText = getString(renderer, "publishDate")
		}
	}

	return info, nil
}

// Search performs a video search
func (p *YouTubeProvider) Search(query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 20
	}

	payload := map[string]interface{}{
		"context": clientContext,
		"query":   query,
		"params":  "",
	}

	data, err := p.performInnertubeRequest(InnertubeSearchUrl, payload)
	if err != nil {
		return nil, err
	}

	return p.parseSearchResults(data, maxResults)
}

// GetTrending retrieves trending videos for a category
func (p *YouTubeProvider) GetTrending(category string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 20
	}

	payload := map[string]interface{}{
		"context":  clientContext,
		"browseId": "FEtrending",
	}

	// Add category-specific params if available
	category = strings.ToLower(category)
	if param, ok := p.TrendingParams[category]; ok {
		payload["params"] = param
	}

	data, err := p.performInnertubeRequest(InnertubeGroupUrl, payload)
	if err != nil {
		return nil, err
	}

	return p.parseTrendingResults(data, maxResults)
}

// GetSearchSuggestions retrieves search autocomplete suggestions
func (p *YouTubeProvider) GetSearchSuggestions(query string) (*SearchSuggestion, error) {
	params := url.Values{}
	params.Set("client", "youtube")
	params.Set("ds", "yt")
	params.Set("hl", "en")
	params.Set("gl", "US")
	params.Set("q", query)

	reqUrl := fmt.Sprintf("%s?%s", SuggestUrl, params.Encode())
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := p.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSONP response
	suggestions := &SearchSuggestion{
		Query:       query,
		Suggestions: []string{},
	}

	// Extract JSON from JSONP response: window.google.ac.h([...])
	re := regexp.MustCompile(`\[.*\]`)
	match := re.Find(body)
	if match == nil {
		return suggestions, nil
	}

	var data []interface{}
	if err := json.Unmarshal(match, &data); err != nil {
		return suggestions, nil
	}

	// Parse suggestions from the response structure
	if len(data) > 1 {
		if suggList, ok := data[1].([]interface{}); ok {
			for _, item := range suggList {
				if arr, ok := item.([]interface{}); ok && len(arr) > 0 {
					if s, ok := arr[0].(string); ok {
						suggestions.Suggestions = append(suggestions.Suggestions, s)
					}
				}
			}
		}
	}

	return suggestions, nil
}

// GetVideoUrlFormat returns the url format of the video page
func (p *YouTubeProvider) GetVideoUrlFormat() string {
	return "https://www.youtube.com/watch?v=%s"
}

// GetThumbnailUrlFormat returns the url format of the default thumbnail for a video
func (p *YouTubeProvider) GetThumbnailUrlFormat() string {
	return "http://i.ytimg.com/vi/%s/hqdefault.jpg"
}

// parseSearchResults parses search results from Innertube response
func (p *YouTubeProvider) parseSearchResults(data map[string]interface{}, maxResults int) ([]SearchResult, error) {
	var results []SearchResult

	contents, ok := data["contents"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	twoColumn, ok := contents["twoColumnSearchResultsRenderer"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	primaryContents, ok := twoColumn["primaryContents"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	sectionList, ok := primaryContents["sectionListRenderer"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	sections, ok := sectionList["contents"].([]interface{})
	if !ok {
		return results, nil
	}

	for _, section := range sections {
		sectionMap, ok := section.(map[string]interface{})
		if !ok {
			continue
		}

		itemSection, ok := sectionMap["itemSectionRenderer"].(map[string]interface{})
		if !ok {
			continue
		}

		items, ok := itemSection["contents"].([]interface{})
		if !ok {
			continue
		}

		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			videoRenderer, ok := itemMap["videoRenderer"].(map[string]interface{})
			if !ok {
				continue
			}

			result := p.parseVideoRenderer(videoRenderer)
			results = append(results, result)

			if len(results) >= maxResults {
				return results, nil
			}
		}
	}

	return results, nil
}

// parseTrendingResults parses trending results from Innertube browse response
func (p *YouTubeProvider) parseTrendingResults(data map[string]interface{}, maxResults int) ([]SearchResult, error) {
	var results []SearchResult
	seen := make(map[string]bool)

	contents, ok := data["contents"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	twoColumn, ok := contents["twoColumnBrowseResultsRenderer"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	tabs, ok := twoColumn["tabs"].([]interface{})
	if !ok || len(tabs) == 0 {
		return results, nil
	}

	firstTab, ok := tabs[0].(map[string]interface{})
	if !ok {
		return results, nil
	}

	tabRenderer, ok := firstTab["tabRenderer"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	content, ok := tabRenderer["content"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	sectionList, ok := content["sectionListRenderer"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	sections, ok := sectionList["contents"].([]interface{})
	if !ok {
		return results, nil
	}

	// Extract videos from all sections
	videos := p.extractVideosFromItems(sections)

	for _, video := range videos {
		if seen[video.VideoID] {
			continue
		}
		seen[video.VideoID] = true
		results = append(results, video)

		if len(results) >= maxResults {
			break
		}
	}

	return results, nil
}

// extractVideosFromItems recursively extracts videos from various item types
func (p *YouTubeProvider) extractVideosFromItems(items []interface{}) []SearchResult {
	var results []SearchResult

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Direct video renderer
		if videoRenderer, ok := itemMap["videoRenderer"].(map[string]interface{}); ok {
			results = append(results, p.parseVideoRenderer(videoRenderer))
			continue
		}

		// Item section renderer
		if itemSection, ok := itemMap["itemSectionRenderer"].(map[string]interface{}); ok {
			if contents, ok := itemSection["contents"].([]interface{}); ok {
				results = append(results, p.extractVideosFromItems(contents)...)
			}
			continue
		}

		// Shelf renderer
		if shelf, ok := itemMap["shelfRenderer"].(map[string]interface{}); ok {
			if content, ok := shelf["content"].(map[string]interface{}); ok {
				if expanded, ok := content["expandedShelfContentsRenderer"].(map[string]interface{}); ok {
					if items, ok := expanded["items"].([]interface{}); ok {
						results = append(results, p.extractVideosFromItems(items)...)
					}
				}
			}
			continue
		}

		// Rich section renderer
		if richSection, ok := itemMap["richSectionRenderer"].(map[string]interface{}); ok {
			if content, ok := richSection["content"].(map[string]interface{}); ok {
				if richShelf, ok := content["richShelfRenderer"].(map[string]interface{}); ok {
					if contents, ok := richShelf["contents"].([]interface{}); ok {
						results = append(results, p.extractVideosFromItems(contents)...)
					}
				}
			}
			continue
		}

		// Rich item renderer
		if richItem, ok := itemMap["richItemRenderer"].(map[string]interface{}); ok {
			if content, ok := richItem["content"].(map[string]interface{}); ok {
				if videoRenderer, ok := content["videoRenderer"].(map[string]interface{}); ok {
					results = append(results, p.parseVideoRenderer(videoRenderer))
				}
			}
			continue
		}
	}

	return results
}

// parseVideoRenderer parses a videoRenderer object into a SearchResult
func (p *YouTubeProvider) parseVideoRenderer(vr map[string]interface{}) SearchResult {
	result := SearchResult{
		Type:    "video",
		VideoID: getString(vr, "videoId"),
	}

	// Title
	if title, ok := vr["title"].(map[string]interface{}); ok {
		if runs, ok := title["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				result.Title = getString(run, "text")
			}
		}
	}

	// Author
	if ownerText, ok := vr["ownerText"].(map[string]interface{}); ok {
		if runs, ok := ownerText["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				result.Author = getString(run, "text")

				if navEndpoint, ok := run["navigationEndpoint"].(map[string]interface{}); ok {
					if browseEndpoint, ok := navEndpoint["browseEndpoint"].(map[string]interface{}); ok {
						result.AuthorID = getString(browseEndpoint, "browseId")
						result.AuthorURL = getString(browseEndpoint, "canonicalBaseUrl")
					}
				}
			}
		}
	}

	// Thumbnails
	if thumbnail, ok := vr["thumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := thumbnail["thumbnails"].([]interface{}); ok {
			for _, t := range thumbs {
				if tm, ok := t.(map[string]interface{}); ok {
					result.Thumbnails = append(result.Thumbnails, Thumbnail{
						URL:    getString(tm, "url"),
						Width:  getInt(tm, "width"),
						Height: getInt(tm, "height"),
					})
				}
			}
		}
	}

	// Description
	if descSnippet, ok := vr["descriptionSnippet"].(map[string]interface{}); ok {
		if runs, ok := descSnippet["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				result.Description = getString(run, "text")
			}
		}
	}

	// View count
	if viewCountText, ok := vr["viewCountText"].(map[string]interface{}); ok {
		result.ViewCountText = getString(viewCountText, "simpleText")
		result.ViewCount = ParseViewCount(result.ViewCountText)
	}

	// Published time
	if publishedText, ok := vr["publishedTimeText"].(map[string]interface{}); ok {
		result.PublishedText = getString(publishedText, "simpleText")
	}

	// Duration
	if lengthText, ok := vr["lengthText"].(map[string]interface{}); ok {
		result.LengthText = getString(lengthText, "simpleText")
		result.LengthSeconds = ParseDuration(result.LengthText)
	}

	// Live status
	if badges, ok := vr["badges"].([]interface{}); ok {
		for _, badge := range badges {
			if b, ok := badge.(map[string]interface{}); ok {
				if metaBadge, ok := b["metadataBadgeRenderer"].(map[string]interface{}); ok {
					label := getString(metaBadge, "label")
					if strings.Contains(strings.ToUpper(label), "LIVE") {
						result.IsLive = true
						break
					}
				}
			}
		}
	}

	// Author verified
	if ownerBadges, ok := vr["ownerBadges"].([]interface{}); ok {
		for _, badge := range ownerBadges {
			if b, ok := badge.(map[string]interface{}); ok {
				if metaBadge, ok := b["metadataBadgeRenderer"].(map[string]interface{}); ok {
					style := getString(metaBadge, "style")
					if style == "BADGE_STYLE_TYPE_VERIFIED" {
						result.AuthorVerified = true
						break
					}
				}
			}
		}
	}

	return result
}
