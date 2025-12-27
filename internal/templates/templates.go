package templates

import (
	"html"
	"strings"
	"text/template"
)

// Templates holds all parsed XML templates
type Templates struct {
	Feed        *template.Template
	VideoEntry  *template.Template
	Suggestions *template.Template
	Error       *template.Template
}

// Global template instance
var templates *Templates

// templateFuncs provides custom functions for templates
var templateFuncs = template.FuncMap{
	"escape": func(s string) string {
		return html.EscapeString(s)
	},
	"escapeAttr": func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&apos;")
		return s
	},
}

// Initialize initializes all XML templates from embedded files
func Initialize(templatesDir string) error {
	var err error
	templates = &Templates{}

	templates.Feed, err = template.New("feed.xml").Funcs(templateFuncs).ParseFiles(templatesDir + "/feed.xml")
	if err != nil {
		return err
	}

	templates.VideoEntry, err = template.New("video_entry.xml").Funcs(templateFuncs).ParseFiles(templatesDir + "/video_entry.xml")
	if err != nil {
		return err
	}

	templates.Suggestions, err = template.New("suggestions.xml").Funcs(templateFuncs).ParseFiles(templatesDir + "/suggestions.xml")
	if err != nil {
		return err
	}

	templates.Error, err = template.New("error.xml").Funcs(templateFuncs).ParseFiles(templatesDir + "/error.xml")
	if err != nil {
		return err
	}

	return nil
}

// GetInstance returns the global templates instance
func GetInstance() *Templates {
	return templates
}
