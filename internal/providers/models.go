package providers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Lekuruu/give-wii-youtube/internal/templates"
)

// VideoInfo contains detailed information about a single video
type VideoInfo struct {
	VideoID       string      `json:"videoId"`
	Title         string      `json:"title"`
	Author        string      `json:"author"`
	AuthorID      string      `json:"authorId"`
	AuthorURL     string      `json:"authorUrl"`
	LengthSeconds int         `json:"lengthSeconds"`
	ViewCount     int64       `json:"viewCount"`
	LikeCount     int64       `json:"likeCount"`
	Description   string      `json:"description"`
	PublishedText string      `json:"publishedText"`
	Keywords      []string    `json:"keywords"`
	IsLive        bool        `json:"isLive"`
	Thumbnails    []Thumbnail `json:"thumbnails"`
}

// Thumbnail represents a video thumbnail
type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// SearchResult represents a video in search results
type SearchResult struct {
	Type           string      `json:"type"`
	Title          string      `json:"title"`
	VideoID        string      `json:"videoId"`
	Author         string      `json:"author"`
	AuthorID       string      `json:"authorId"`
	AuthorURL      string      `json:"authorUrl"`
	AuthorVerified bool        `json:"authorVerified"`
	Description    string      `json:"description"`
	ViewCount      int64       `json:"viewCount"`
	ViewCountText  string      `json:"viewCountText"`
	PublishedText  string      `json:"publishedText"`
	LengthSeconds  int         `json:"lengthSeconds"`
	LengthText     string      `json:"lengthText"`
	IsLive         bool        `json:"liveNow"`
	Thumbnails     []Thumbnail `json:"videoThumbnails"`
}

// SearchSuggestion represents a search autocomplete suggestion
type SearchSuggestion struct {
	Query       string   `json:"query"`
	Suggestions []string `json:"suggestions"`
}

// Category represents a content category (trending, music, etc.)
type Category struct {
	Name   string
	Param  string
	Videos []SearchResult
}

// FeedTemplateData is the data structure for feed.xml template
type FeedTemplateData struct {
	Results []SearchResultTemplateData
}

// SearchResultTemplateData is the data structure for search result entries in templates
type SearchResultTemplateData struct {
	VideoID       string
	Title         string
	Author        string
	AuthorID      string
	Description   string
	PublishedText string
	LengthSeconds int
	ViewCount     int64
	ThumbnailURL  string
}

// VideoEntryTemplateData is the data structure for video_entry.xml template
type VideoEntryTemplateData struct {
	VideoID       string
	Title         string
	Author        string
	AuthorID      string
	PublishedText string
	LengthSeconds int
	ViewCount     int64
	LikeCount     int64
	ThumbnailURL  string
}

// SuggestionsTemplateData is the data structure for suggestions.xml template
type SuggestionsTemplateData struct {
	Query       string
	Suggestions []string
}

// ToTemplateData converts a SearchResult to template data
func (s *SearchResult) ToTemplateData(thumbnailFormat string) SearchResultTemplateData {
	thumbnailURL := fmt.Sprintf(thumbnailFormat, s.VideoID)
	if len(s.Thumbnails) > 0 {
		thumbnailURL = s.Thumbnails[0].URL
	}

	return SearchResultTemplateData{
		VideoID:       s.VideoID,
		Title:         s.Title,
		Author:        s.Author,
		AuthorID:      s.AuthorID,
		Description:   s.Description,
		PublishedText: s.PublishedText,
		LengthSeconds: s.LengthSeconds,
		ViewCount:     s.ViewCount,
		ThumbnailURL:  strings.Replace(thumbnailURL, "https://", "http://", 1),
	}
}

// ToTemplateData converts VideoInfo to template data
func (v *VideoInfo) ToTemplateData(thumbnailFormat string) VideoEntryTemplateData {
	thumbnailURL := fmt.Sprintf(thumbnailFormat, v.VideoID)
	if len(v.Thumbnails) > 0 {
		thumbnailURL = v.Thumbnails[0].URL
	}

	return VideoEntryTemplateData{
		VideoID:       v.VideoID,
		Title:         v.Title,
		Author:        v.Author,
		AuthorID:      v.AuthorID,
		PublishedText: v.PublishedText,
		LengthSeconds: v.LengthSeconds,
		ViewCount:     v.ViewCount,
		LikeCount:     v.LikeCount,
		ThumbnailURL:  strings.Replace(thumbnailURL, "https://", "http://", 1),
	}
}

// ToVideoInfoResponse generates the legacy get_video_info format response
func (v *VideoInfo) ToVideoInfoResponse(thumbnailFormat string) string {
	thumbnailURL := fmt.Sprintf(thumbnailFormat, v.VideoID)
	fmtList := "43/854x480/9/0/115"
	fmtStreamMap := "43|"
	fmtMap := "43/0/7/0/0"

	params := []string{
		fmt.Sprintf("status=%s", "ok"),
		fmt.Sprintf("length_seconds=%d", v.LengthSeconds),
		fmt.Sprintf("keywords=%s", "a"),
		fmt.Sprintf("vq=%s", "None"),
		fmt.Sprintf("muted=%d", 0),
		fmt.Sprintf("avg_rating=%.1f", 5.0),
		fmt.Sprintf("thumbnailUrl=%s", thumbnailURL),
		fmt.Sprintf("allow_ratings=%s", "1"),
		fmt.Sprintf("hl=%s", "en"),
		fmt.Sprintf("ftoken=%s", ""),
		fmt.Sprintf("allow_embed=%s", "1"),
		fmt.Sprintf("fmtMap=%s", fmtMap),
		fmt.Sprintf("fmt_url_map=%s", fmtStreamMap),
		fmt.Sprintf("token=%s", "null"),
		fmt.Sprintf("plid=%s", "null"),
		fmt.Sprintf("track_embed=%d", 0),
		fmt.Sprintf("author=%s", v.Author),
		fmt.Sprintf("title=%s", v.Title),
		fmt.Sprintf("videoId=%s", v.VideoID),
		fmt.Sprintf("fmtList=%s", fmtList),
		fmt.Sprintf("fmtStreamMap=%s", fmtStreamMap),
	}

	return strings.Join(params, "&")
}

// ToXMLEntry generates an Atom entry for the video using templates
func (v *VideoInfo) ToXMLEntry(thumbnailFormat string) (string, error) {
	templates := templates.GetInstance()
	if templates == nil {
		return "", fmt.Errorf("templates not initialized")
	}
	return templates.RenderVideoEntry(v.ToTemplateData(thumbnailFormat))
}

// GenerateFeedXML generates a complete Atom feed from search results using templates
func GenerateFeedXML(results []SearchResult, thumbnailFormat string) (string, error) {
	templates := templates.GetInstance()
	if templates == nil {
		return "", fmt.Errorf("templates not initialized")
	}

	// Convert results to template data
	templateResults := make([]SearchResultTemplateData, len(results))
	for i, r := range results {
		templateResults[i] = r.ToTemplateData(thumbnailFormat)
	}

	data := FeedTemplateData{Results: templateResults}
	return templates.RenderFeed(data)
}

// GenerateSuggestionsXML generates XML for search suggestions using templates
func GenerateSuggestionsXML(query string, suggestions []string) (string, error) {
	templates := templates.GetInstance()
	if templates == nil {
		return "", fmt.Errorf("templates not initialized")
	}

	data := SuggestionsTemplateData{
		Query:       query,
		Suggestions: suggestions,
	}
	return templates.RenderSuggestions(data)
}

// GenerateErrorXML generates an XML error response using templates
func GenerateErrorXML(message string) (string, error) {
	templates := templates.GetInstance()
	if templates == nil {
		return "", fmt.Errorf("templates not initialized")
	}
	return templates.RenderError(message)
}

// ParseViewCount parses view count text like "1.5M views" to an integer
func ParseViewCount(viewCountText string) int64 {
	text := strings.ToLower(viewCountText)
	text = strings.ReplaceAll(text, "views", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.TrimSpace(text)

	multiplier := int64(1)
	if strings.Contains(text, "k") {
		multiplier = 1000
		text = strings.ReplaceAll(text, "k", "")
	} else if strings.Contains(text, "m") {
		multiplier = 1000000
		text = strings.ReplaceAll(text, "m", "")
	} else if strings.Contains(text, "b") {
		multiplier = 1000000000
		text = strings.ReplaceAll(text, "b", "")
	}

	text = strings.TrimSpace(text)
	if f, err := strconv.ParseFloat(text, 64); err == nil {
		return int64(f * float64(multiplier))
	}

	// Try parsing as plain integer
	var result int64
	for _, r := range text {
		if r >= '0' && r <= '9' {
			result = result*10 + int64(r-'0')
		}
	}
	return result
}

// ParseDuration parses duration text like "3:45" or "1:23:45" to seconds
func ParseDuration(durationText string) int {
	parts := strings.Split(durationText, ":")
	if len(parts) == 0 {
		return 0
	}

	var seconds int
	for i, part := range parts {
		val, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			continue
		}

		switch len(parts) - i {
		case 1: // seconds
			seconds += val
		case 2: // minutes
			seconds += val * 60
		case 3: // hours
			seconds += val * 3600
		}
	}

	return seconds
}
