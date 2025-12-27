package routes

import (
	"github.com/Lekuruu/give-wii-youtube/internal/app"
	"github.com/Lekuruu/give-wii-youtube/internal/providers"
)

func RegisterCategoryRoutes(server *app.Server) {
	for _, category := range server.State.Categories.Entries {
		categoryInstance := category
		categoryHandler := func(ctx *app.Context) { HandleCategory(ctx, &categoryInstance) }
		server.Router.HandleFunc(categoryInstance.Route, server.ContextMiddleware(categoryHandler)).Methods("GET")
	}
}

// HandleCategory handles requests for trending categories
func HandleCategory(ctx *app.Context, category *app.Category) {
	var results []providers.SearchResult
	var err error

	// Use trending param if available, otherwise use search fallback
	if category.TrendingParam != "" {
		results, err = ctx.State.Provider.GetTrending(category.TrendingParam, 20)
	} else {
		results, err = ctx.State.Provider.GetTrending("", 20)
	}

	if err != nil || len(results) == 0 {
		// Fallback to search
		ctx.State.Logger.Errorf("Failed to get trending for %s: %v", category.Name, err)
		HandleCategorySearch(ctx, category.SearchFallback)
		return
	}

	xml, err := providers.GenerateFeedXML(results, ctx.State.Provider.GetThumbnailUrlFormat())
	if err != nil {
		ctx.State.Logger.Errorf("Failed to generate feed XML: %v", err)
		writeXMLError(ctx.Response, err.Error())
		return
	}

	ctx.Response.Header().Set("Content-Type", "text/xml; charset=utf-8")
	ctx.Response.Write([]byte(xml))
}

// HandleCategorySearch uses search as fallback for categories not in trending API
func HandleCategorySearch(ctx *app.Context, query string) {
	results, err := ctx.State.Provider.Search(query, 20)
	if err != nil {
		ctx.State.Logger.Errorf("Category search failed for '%s': %v", query, err)
		writeXMLError(ctx.Response, err.Error())
		return
	}

	xml, err := providers.GenerateFeedXML(results, ctx.State.Provider.GetThumbnailUrlFormat())
	if err != nil {
		ctx.State.Logger.Errorf("Failed to generate feed XML: %v", err)
		writeXMLError(ctx.Response, err.Error())
		return
	}

	ctx.Response.Header().Set("Content-Type", "text/xml; charset=utf-8")
	ctx.Response.Write([]byte(xml))
}
