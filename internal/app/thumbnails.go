package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ThumbnailCache handles category thumbnail caching
type ThumbnailCache struct {
	CacheDir     string
	Logger       *Logger
	thumbnailUrl string
	interval     time.Duration
	stopChan     chan struct{}
}

// NewThumbnailCache creates a new thumbnail cache
func NewThumbnailCache(cacheDir string, logger *Logger, interval time.Duration, thumbnailUrl string) *ThumbnailCache {
	os.MkdirAll(cacheDir, 0755)
	return &ThumbnailCache{
		CacheDir:     cacheDir,
		Logger:       logger,
		thumbnailUrl: thumbnailUrl,
		interval:     interval,
		stopChan:     make(chan struct{}),
	}
}

// Start begins the thumbnail update scheduler
func (tc *ThumbnailCache) Start(categories []string, getFirstVideoID func(category string) string) {
	go func() {
		ticker := time.NewTicker(tc.interval)
		defer ticker.Stop()

		// Initial update
		tc.updateThumbnails(categories, getFirstVideoID)

		for {
			select {
			case <-ticker.C:
				tc.updateThumbnails(categories, getFirstVideoID)
			case <-tc.stopChan:
				return
			}
		}
	}()
}

// Stop stops the thumbnail update scheduler
func (tc *ThumbnailCache) Stop() {
	close(tc.stopChan)
}

// updateThumbnails updates thumbnails for all categories
func (tc *ThumbnailCache) updateThumbnails(categories []string, getFirstVideoID func(category string) string) {
	for _, category := range categories {
		videoID := getFirstVideoID(category)
		if videoID == "" {
			continue
		}

		thumbnailUrl := fmt.Sprintf(tc.thumbnailUrl, videoID)
		if err := tc.downloadThumbnail(thumbnailUrl, category); err != nil {
			tc.Logger.Errorf("Failed to download thumbnail for %s: %v", category, err)
		} else {
			tc.Logger.Logf("Updated thumbnail for category: %s", category)
		}
	}
}

// downloadThumbnail downloads a thumbnail image
func (tc *ThumbnailCache) downloadThumbnail(url, category string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	filePath := filepath.Join(tc.CacheDir, category+".jpg")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
