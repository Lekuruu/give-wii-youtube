package routes

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Lekuruu/give-wii-youtube/internal/app"
)

type LeanbackSet struct {
	Title       string `json:"title"`
	ListID      string `json:"list_id"`
	VideoCount  int    `json:"video_count"`
	Thumbnail   string `json:"thumbnail"`
	GDataListID string `json:"gdata_list_id"`
	GDataURL    string `json:"gdata_url"`
}

type LeanbackResponse struct {
	Sets []LeanbackSet `json:"sets"`
}

// SWF file paths relative to static directory
const (
	LoaderSWF    = "loader.swf"
	APIPlayerSWF = "apiplayer.swf"
	LeanbackSWF  = "leanbacklite_wii.swf"
)

func RegisterStaticRoutes(server *app.Server, staticDir string) {
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

	// Serve thumbnail cache files
	fs := http.FileServer(http.Dir(filepath.Join(server.State.Config.Storage.Path, "cache", "thumbnails")))
	server.Router.PathPrefix("/dl/").Handler(http.StripPrefix("/dl/", fs))

	// /leanback_ajax
	server.Router.HandleFunc("/leanback_ajax", server.ContextMiddleware(ServeLeanbackAjax)).Methods("GET")
}

func ServeSWF(ctx *app.Context, staticDir, filename string) {
	filePath := filepath.Join(staticDir, filename)
	ctx.Response.Header().Set("Content-Type", "application/x-shockwave-flash")
	ctx.Response.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(ctx.Response, ctx.Request, filePath)
}

func ServeLeanbackAjax(ctx *app.Context) {
	ctx.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response := LeanbackResponse{Sets: make([]LeanbackSet, 0)}

	for _, category := range ctx.State.Categories.Entries {
		set := LeanbackSet{
			Title:       category.Name,
			ListID:      strings.ToLower(category.Name),
			VideoCount:  25, // TODO: Make this dynamic somehow
			Thumbnail:   ctx.State.Config.Server.Url + "/dl/" + strings.ToLower(category.Name) + ".jpg",
			GDataURL:    ctx.State.Config.Server.Url + category.Route,
			GDataListID: category.Name,
		}
		response.Sets = append(response.Sets, set)
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		ctx.State.Logger.Errorf("Failed to marshal leanback ajax response: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Response.Write(jsonData)
}
