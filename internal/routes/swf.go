package routes

import (
	"net/http"
	"path/filepath"

	"github.com/Lekuruu/give-wii-youtube/internal/app"
)

// SWF file paths relative to static directory
const (
	LoaderSWF    = "loader.swf"
	APIPlayerSWF = "apiplayer.swf"
	LeanbackSWF  = "leanbacklite_wii.swf"
)

func RegisterSWFRoutes(server *app.Server, staticDir string) {
	// /apiplayer-loader -> static/loader.swf
	server.Router.HandleFunc("/apiplayer-loader", server.ContextMiddleware(func(ctx *app.Context) {
		ServeSWF(ctx, staticDir, LoaderSWF)
	})).Methods("GET", "HEAD")

	// /videoplayback -> apiplayer.swf
	server.Router.HandleFunc("/videoplayback", server.ContextMiddleware(func(ctx *app.Context) {
		ServeSWF(ctx, staticDir, APIPlayerSWF)
	})).Methods("GET", "HEAD")

	// /wiitv -> leanbacklite_wii.swf
	server.Router.HandleFunc("/wiitv", server.ContextMiddleware(func(ctx *app.Context) {
		ServeSWF(ctx, staticDir, LeanbackSWF)
	})).Methods("GET", "HEAD")

	// /player_204 -> empty response (tracking endpoint)
	server.Router.HandleFunc("/player_204", server.ContextMiddleware(func(ctx *app.Context) {
		ctx.Response.WriteHeader(http.StatusNoContent)
	})).Methods("GET")
}

func ServeSWF(ctx *app.Context, staticDir, filename string) {
	filePath := filepath.Join(staticDir, filename)
	ctx.Response.Header().Set("Content-Type", "application/x-shockwave-flash")
	ctx.Response.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(ctx.Response, ctx.Request, filePath)
}
