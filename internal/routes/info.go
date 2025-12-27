package routes

import (
	"net/http"

	"github.com/Lekuruu/give-wii-youtube/internal/app"
)

func RegisterInfoRoutes(server *app.Server) {
	server.Router.HandleFunc("/get_video_info", server.ContextMiddleware(HandleGetVideoInfo)).Methods("GET")
	server.Router.HandleFunc("/feeds/api/videos/{video_id}", server.ContextMiddleware(HandleVideoFeed)).Methods("GET")
}

// HandleGetVideoInfo returns video info in the legacy url-encoded format
func HandleGetVideoInfo(ctx *app.Context) {
	videoID := ctx.Request.URL.Query().Get("video_id")
	if videoID == "" {
		// NOTE: These error responses are made up, no idea how well they match the real ones
		ctx.Response.WriteHeader(http.StatusBadRequest)
		ctx.Response.Write([]byte("status=fail&errorcode=2&reason=Invalid+video+id"))
		return
	}

	info, err := ctx.State.Provider.GetVideoInfo(videoID, "US", "en")
	if err != nil {
		ctx.State.Logger.Errorf("Failed to get video info for %s: %v", videoID, err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Write([]byte("status=fail&errorcode=100&reason=Unable+to+fetch+video+info"))
		return
	}

	ctx.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	ctx.Response.Write([]byte(info.ToVideoInfoResponse(ctx.State.Provider.GetThumbnailUrlFormat())))
}

// HandleVideoFeed returns video info as an atom xml entry
func HandleVideoFeed(ctx *app.Context) {
	videoID := ctx.Vars["video_id"]
	if videoID == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		writeXMLError(ctx.Response, "Invalid video ID")
		return
	}

	info, err := ctx.State.Provider.GetVideoInfo(videoID, "US", "en")
	if err != nil {
		ctx.State.Logger.Errorf("Failed to get video feed for %s: %v", videoID, err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		writeXMLError(ctx.Response, err.Error())
		return
	}

	xml, err := info.ToXMLEntry(ctx.State.Provider.GetThumbnailUrlFormat())
	if err != nil {
		ctx.State.Logger.Errorf("Failed to generate video XML: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		writeXMLError(ctx.Response, err.Error())
		return
	}

	ctx.Response.Header().Set("Content-Type", "text/xml; charset=utf-8")
	ctx.Response.Write([]byte(xml))
}
