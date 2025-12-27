package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Lekuruu/give-wii-youtube/internal/app"
	"github.com/Lekuruu/give-wii-youtube/internal/providers"
	"github.com/Lekuruu/give-wii-youtube/internal/routes"
	"github.com/Lekuruu/give-wii-youtube/internal/templates"
)

func main() {
	state := app.NewState()
	if state == nil {
		os.Exit(1)
	}

	// Ensure required folders exist
	if err := state.Storage.Setup(); err != nil {
		state.Logger.Errorf("Failed to setup storage: %v", err)
		os.Exit(1)
	}

	server := app.NewServer(
		state.Config.Server.Host,
		state.Config.Server.Port,
		"wii-youtube",
		state,
	)

	// Initialize provider
	state.Provider = providers.NewYouTubeProvider(state.Categories.GetTrendingParameters())

	// Initialize paths
	staticDir, downloadDir, templatesDir, cacheDir := initializePaths(state)

	// Initialize templates
	if err := templates.Initialize(templatesDir); err != nil {
		state.Logger.Errorf("Failed to initialize templates: %v", err)
		os.Exit(1)
	}

	// Create video streamer
	videoStreamer := routes.NewVideoStreamer(
		downloadDir,
		filepath.Join(cacheDir, "videos"),
		state.Config.Video.Quality,
		state.Logger,
	)

	// Register all routes
	registerRoutes(server, videoStreamer, staticDir)

	// Launch thumbnail cache scheduler
	thumbnailCache := setupThumbnailCache(state)

	// Handle graceful shutdown
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChannel
		state.Logger.Log("Shutting down...")
		thumbnailCache.Stop()
		os.Exit(0)
	}()

	server.Serve()
}

func registerRoutes(server *app.Server, streamer *routes.VideoStreamer, staticDir string) {
	routes.RegisterStaticRoutes(server, staticDir)
	routes.RegisterInfoRoutes(server)
	routes.RegisterSearchRoutes(server)
	routes.RegisterCategoryRoutes(server)
	routes.RegisterVideoRoutes(server, streamer)

	// Health check endpoint
	server.Router.HandleFunc("/health", server.ContextMiddleware(func(ctx *app.Context) {
		ctx.Response.Header().Set("Content-Type", "application/json")
		ctx.Response.Write([]byte(`{"status":"ok"}`))
	})).Methods("GET")
}

func initializePaths(state *app.State) (staticDir, downloadDir, templatesDir, cacheDir string) {
	// Get paths for static files and downloads
	execPath, _ := os.Executable()
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
	staticDir = filepath.Join(baseDir, "static")

	// Handle running from source directory
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		// Use current working directory
		baseDir, _ = os.Getwd()
	}

	downloadDir = state.Config.Video.DownloadFolder
	staticDir = filepath.Join(baseDir, "static")
	templatesDir = filepath.Join(baseDir, "templates")
	cacheDir = filepath.Join(state.Config.Storage.Path, "cache")

	// Ensure directories exist
	os.MkdirAll(cacheDir, 0755)
	os.MkdirAll(staticDir, 0755)
	os.MkdirAll(downloadDir, 0755)

	return staticDir, downloadDir, templatesDir, cacheDir
}

func setupThumbnailCache(state *app.State) *app.ThumbnailCache {
	cacheDuration := time.Duration(state.Config.Cache.Duration) * time.Second
	if cacheDuration <= 0 {
		cacheDuration = 10 * time.Minute
	}
	thumbnailCache := app.NewThumbnailCache(
		filepath.Join(state.Config.Storage.Path, "cache", "thumbnails"),
		state.Logger,
		cacheDuration,
		state.Provider.GetThumbnailUrlFormat(),
	)

	// Get category names from config
	categoryNames := state.Categories.GetCategoryNames()

	thumbnailCache.Start(categoryNames, func(categoryName string) string {
		// Get first video from category to use its thumbnail
		category := state.Categories.GetCategory(categoryName)
		if category == nil {
			return ""
		}

		var results []providers.SearchResult
		var err error

		if category.TrendingParam != "" {
			results, err = state.Provider.GetTrending(category.TrendingParam, 1)
		} else {
			results, err = state.Provider.GetTrending("", 1)
		}

		if err != nil || len(results) == 0 {
			// Fallback to search
			results, err = state.Provider.Search(category.SearchFallback, 1)
			if err != nil || len(results) == 0 {
				return ""
			}
		}

		return results[0].VideoID
	})
	return thumbnailCache
}
