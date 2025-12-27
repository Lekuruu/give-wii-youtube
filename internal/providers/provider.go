package providers

// Provider interface for video providers, e.g. youtube
type Provider interface {
	GetVideoInfo(videoId string) (*VideoInfo, error)
	Search(query string, maxResults int) ([]SearchResult, error)
	GetTrending(category string, maxResults int) ([]SearchResult, error)
	GetSearchSuggestions(query string) (*SearchSuggestion, error)
	GetVideoUrlFormat() string
	GetThumbnailUrlFormat() string
}
