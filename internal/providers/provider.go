package providers

// Provider interface for video providers, e.g. youtube
type Provider interface {
	GetVideoInfo(videoId string, country string, language string) (*VideoInfo, error)
	Search(query string, maxResults int, country string, language string) ([]SearchResult, error)
	GetTrending(category string, maxResults int, country string, language string) ([]SearchResult, error)
	GetSearchSuggestions(query string, country string, language string) (*SearchSuggestion, error)
	GetVideoUrlFormat() string
	GetThumbnailUrlFormat() string
}
