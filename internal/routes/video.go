package routes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	ffmpeg "github.com/Lekuruu/ffmpeg-go"
	"github.com/Lekuruu/give-wii-youtube/internal/app"
	"github.com/lrstanley/go-ytdlp"
)

// VideoStreamer handles video downloading and streaming
type VideoStreamer struct {
	DownloadDir string
	CacheDir    string
	Logger      *app.Logger
	Quality     string

	// Track videos being processed
	processing sync.Map
}

// NewVideoStreamer creates a new video streamer
func NewVideoStreamer(downloadDir, cacheDir, quality string, logger *app.Logger) *VideoStreamer {
	// Ensure directories exist
	os.MkdirAll(downloadDir, 0755)
	os.MkdirAll(cacheDir, 0755)

	// Default to 360p, if not specified
	if quality == "" {
		quality = "360"
	}

	return &VideoStreamer{
		DownloadDir: downloadDir,
		CacheDir:    cacheDir,
		Logger:      logger,
		Quality:     quality,
	}
}

func RegisterVideoRoutes(server *app.Server, streamer *VideoStreamer) {
	server.Router.HandleFunc("/get_video", server.ContextMiddleware(streamer.HandleGetVideo)).Methods("GET")
	server.Router.HandleFunc("/git_video", server.ContextMiddleware(streamer.HandleGitVideo)).Methods("GET")
	server.Router.HandleFunc("/videos/{filename}", server.ContextMiddleware(streamer.HandleServeVideo)).Methods("GET")
}

// HandleGetVideo downloads video and converts to webm for Wii playback
func (vs *VideoStreamer) HandleGetVideo(ctx *app.Context) {
	videoID := ctx.Request.URL.Query().Get("video_id")
	videoUrl := fmt.Sprintf(ctx.State.Provider.GetVideoUrlFormat(), videoID)
	if videoID == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		ctx.Response.Write([]byte("Missing video_id parameter"))
		return
	}

	// Check if webm already exists
	webmPath := filepath.Join(vs.CacheDir, videoID+".webm")
	if fileExists(webmPath) {
		vs.serveFile(ctx, webmPath, "video/webm")
		return
	}

	// Check if already processing
	if _, processing := vs.processing.LoadOrStore(videoID, true); processing {
		ctx.Response.WriteHeader(http.StatusAccepted)
		ctx.Response.Write([]byte("Video is being processed, please try again later"))
		return
	}
	defer vs.processing.Delete(videoID)

	// Download video using yt-dlp
	mp4Path := filepath.Join(vs.DownloadDir, videoID+".mp4")

	if !fileExists(mp4Path) {
		if err := vs.downloadVideo(videoUrl, mp4Path); err != nil {
			vs.Logger.Errorf("Failed to download video %s: %v", videoID, err)
			ctx.Response.WriteHeader(http.StatusInternalServerError)
			ctx.Response.Write([]byte("Failed to download video"))
			return
		}
	}

	// Convert to webm
	if err := vs.convertToWebm(mp4Path, webmPath); err != nil {
		vs.Logger.Errorf("Failed to convert video %s: %v", videoID, err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Write([]byte("Failed to convert video"))
		return
	}

	vs.serveFile(ctx, webmPath, "video/webm")
}

// HandleGitVideo streams video as flv with real-time transcoding
func (vs *VideoStreamer) HandleGitVideo(ctx *app.Context) {
	videoID := ctx.Request.URL.Query().Get("video_id")
	if videoID == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		ctx.Response.Write([]byte("Missing video_id parameter"))
		return
	}

	// Get direct video url using yt-dlp
	videoUrl := fmt.Sprintf(ctx.State.Provider.GetVideoUrlFormat(), videoID)
	streamUrl, err := vs.getVideoUrl(videoUrl)
	if err != nil {
		vs.Logger.Errorf("Failed to get video url for %s: %v", videoID, err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Write([]byte("Failed to get video url"))
		return
	}

	// Parse range header for seeking
	rangeHeader := ctx.Request.Header.Get("Range")
	rangeStart := 0
	if rangeHeader != "" {
		if strings.HasPrefix(rangeHeader, "bytes=") {
			parts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
			if len(parts) > 0 {
				rangeStart, _ = strconv.Atoi(parts[0])
			}
		}
	}

	// Calculate start time based on byte position
	// Approximate bitrate: 500kbps video + 96kbps audio
	totalBitrate := float64(500000 + 96000)
	bytesPerSecond := totalBitrate / 8
	startTime := float64(rangeStart) / bytesPerSecond

	// Set headers for streaming
	ctx.Response.Header().Set("Content-Type", "video/x-flv")
	ctx.Response.Header().Set("Accept-Ranges", "bytes")
	ctx.Response.Header().Set("Cache-Control", "no-cache")
	ctx.Response.Header().Set("Transfer-Encoding", "chunked")

	// Stream with ffmpeg
	vs.streamWithFFmpeg(ctx, streamUrl, startTime)
}

// HandleServeVideo serves a cached video file
func (vs *VideoStreamer) HandleServeVideo(ctx *app.Context) {
	filename := ctx.Vars["filename"]
	if filename == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	// Sanitize filename to prevent directory traversal
	filename = filepath.Base(filename)

	// Try cache directory first, then download directory
	filePath := filepath.Join(vs.CacheDir, filename)
	if !fileExists(filePath) {
		filePath = filepath.Join(vs.DownloadDir, filename)
	}

	if !fileExists(filePath) {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	// Determine content type
	contentType := "video/mp4"
	switch {
	case strings.HasSuffix(filename, ".webm"):
		contentType = "video/webm"
	case strings.HasSuffix(filename, ".flv"):
		contentType = "video/x-flv"
	}

	vs.serveFile(ctx, filePath, contentType)
}

// downloadVideo downloads a video using yt-dlp
func (vs *VideoStreamer) downloadVideo(videoUrl, outputPath string) error {
	vs.Logger.Logf("Downloading video %s at quality %s", videoUrl, vs.Quality)

	dl := ytdlp.New().
		FormatSort(fmt.Sprintf("res:%s,ext:mp4:m4a", vs.Quality)).
		NoPlaylist().
		NoOverwrites().
		Continue().
		Output(outputPath)

	_, err := dl.Run(context.Background(), videoUrl)
	if err != nil {
		return fmt.Errorf("yt-dlp failed: %w", err)
	}

	return nil
}

// getVideoUrl gets a direct video url using yt-dlp
func (vs *VideoStreamer) getVideoUrl(videoUrl string) (string, error) {
	dl := ytdlp.New().
		FormatSort(fmt.Sprintf("res:%s,ext:mp4:m4a", vs.Quality)).
		NoPlaylist().
		Print("urls")

	result, err := dl.Run(context.Background(), videoUrl)
	if err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w", err)
	}

	url := strings.TrimSpace(result.Stdout)
	if url == "" {
		return "", fmt.Errorf("no url returned")
	}

	// yt-dlp may return multiple urls, take the first one
	lines := strings.Split(url, "\n")
	return lines[0], nil
}

// convertToWebm converts a video file to webm format optimized for Wii
func (vs *VideoStreamer) convertToWebm(inputPath, outputPath string) error {
	vs.Logger.Logf("Converting video to webm at quality %s: %s", vs.Quality, inputPath)

	// Calculate bitrate based on quality
	bitrate := "300k"
	qualityInt, err := strconv.Atoi(vs.Quality)
	if err == nil {
		if qualityInt >= 720 {
			bitrate = "1000k"
		} else if qualityInt >= 480 {
			bitrate = "500k"
		}
	}

	err = ffmpeg.Input(inputPath).
		Output(outputPath, ffmpeg.KwArgs{
			"vf":       fmt.Sprintf("scale=-1:%s", vs.Quality),
			"c:v":      "libvpx",
			"b:v":      bitrate,
			"cpu-used": "8",
			"pix_fmt":  "yuv420p",
			"c:a":      "libvorbis",
			"b:a":      "128k",
			"r":        "30",
			"g":        "30",
		}).
		OverWriteOutput().
		Run()

	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	return nil
}

// streamWithFFmpeg streams video content through FFmpeg transcoding
func (vs *VideoStreamer) streamWithFFmpeg(ctx *app.Context, streamUrl string, startTime float64) {
	// Calculate bitrate based on quality
	bitrate := "300k"
	switch vs.Quality {
	case "480":
		bitrate = "500k"
	case "720":
		bitrate = "1000k"
	}

	// Build input with optional seek
	inputKwArgs := ffmpeg.KwArgs{}
	if startTime > 0 {
		inputKwArgs["ss"] = fmt.Sprintf("%.2f", startTime)
	}

	// Create the ffmpeg stream
	stream := ffmpeg.Input(streamUrl, inputKwArgs).
		Output("pipe:1", ffmpeg.KwArgs{
			"c:v": "flv1",
			"b:v": bitrate,
			"vf":  fmt.Sprintf("scale=-1:%s", vs.Quality),
			"c:a": "mp3",
			"b:a": "96k",
			"r":   "24",
			"g":   "24",
			"f":   "flv",
		})

	// Get the command to run with pipe output
	cmd := stream.Compile()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		vs.Logger.Errorf("Failed to create FFmpeg stdout pipe: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Capture stderr for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		vs.Logger.Errorf("Failed to start FFmpeg: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Stream output to response
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if _, writeErr := ctx.Response.Write(buf[:n]); writeErr != nil {
				// Client disconnected
				break
			}

			if flusher, ok := ctx.Response.(http.Flusher); ok {
				// Flush if possible
				flusher.Flush()
			}
		}
		if err != nil {
			if err != io.EOF {
				vs.Logger.Errorf("Error reading FFmpeg output: %v, stderr: %s", err, stderr.String())
			}
			break
		}
	}

	// Clean up
	cmd.Wait()
}

// serveFile serves a file with proper headers and range support
func (vs *VideoStreamer) serveFile(ctx *app.Context, filePath, contentType string) {
	file, err := os.Open(filePath)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Response.Header().Set("Content-Type", contentType)
	ctx.Response.Header().Set("Accept-Ranges", "bytes")
	ctx.Response.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

	// Handle range requests
	rangeHeader := ctx.Request.Header.Get("Range")
	if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") {
		ranges := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.Split(ranges, "-")

		start, _ := strconv.ParseInt(parts[0], 10, 64)
		end := stat.Size() - 1

		if len(parts) > 1 && parts[1] != "" {
			end, _ = strconv.ParseInt(parts[1], 10, 64)
		}

		if start > end || start >= stat.Size() {
			ctx.Response.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		contentLength := end - start + 1
		ctx.Response.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
		ctx.Response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size()))
		ctx.Response.WriteHeader(http.StatusPartialContent)

		file.Seek(start, io.SeekStart)
		io.CopyN(ctx.Response, file, contentLength)
		return
	}

	http.ServeContent(ctx.Response, ctx.Request, filepath.Base(filePath), stat.ModTime(), file)
}
