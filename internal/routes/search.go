package routes

import (
	"net/http"

	"github.com/Lekuruu/give-wii-youtube/internal/app"
	"github.com/Lekuruu/give-wii-youtube/internal/providers"
)

func RegisterSearchRoutes(server *app.Server) {
	server.Router.HandleFunc("/complete/search", server.ContextMiddleware(HandleSearchSuggestions)).Methods("GET")
	server.Router.HandleFunc("/feeds/api/videos", server.ContextMiddleware(HandleVideoSearch)).Methods("GET")
}

// HandleSearchSuggestions handles search suggestion requests
func HandleSearchSuggestions(ctx *app.Context) {
	query := ctx.Request.URL.Query().Get("q")
	if query == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestion, err := ctx.State.Provider.GetSearchSuggestions(query)
	if err != nil {
		ctx.State.Logger.Errorf("Failed to fetch suggestions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	xml, err := providers.GenerateSuggestionsXML(query, suggestion.Suggestions)
	if err != nil {
		ctx.State.Logger.Errorf("Failed to generate suggestions XML: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Response.Header().Set("Content-Type", "text/xml; charset=utf-8")
	ctx.Response.Write([]byte(xml))
}

// HandleVideoSearch handles video search requests
func HandleVideoSearch(ctx *app.Context) {
	query := ctx.Request.URL.Query().Get("q")
	if query == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		writeXMLError(ctx.Response, "Missing search query")
		return
	}

	results, err := ctx.State.Provider.Search(query, 20)
	if err != nil {
		ctx.State.Logger.Errorf("Search failed for query '%s': %v", query, err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		writeXMLError(ctx.Response, err.Error())
		return
	}

	xml, err := providers.GenerateFeedXML(results, ctx.State.Provider.GetThumbnailUrlFormat())
	if err != nil {
		ctx.State.Logger.Errorf("Failed to generate feed XML: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		writeXMLError(ctx.Response, err.Error())
		return
	}

	ctx.Response.Header().Set("Content-Type", "text/xml; charset=utf-8")
	ctx.Response.Write([]byte(xml))
}
