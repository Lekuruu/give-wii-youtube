package routes

import (
	"net/http"
	"os"
	"strings"

	"github.com/Lekuruu/give-wii-youtube/internal/providers"
)

func writeXMLError(w http.ResponseWriter, message string) {
	xml, err := providers.GenerateErrorXML(message)
	if err != nil {
		// Fallback to simple error if templates fail
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><error>` + message + `</error>`))
		return
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(xml))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func resolveLocationMetadata(request *http.Request) (string, string) {
	country := "US"
	language := "en"

	// Check for Accept-Language header, usually just "de" or "en"
	if al := request.Header.Get("Accept-Language"); al != "" {
		parts := strings.Split(al, ",")

		if len(parts) > 0 {
			langPart := strings.TrimSpace(parts[0])
			if len(langPart) >= 2 {
				language = langPart[0:2]
			}
		}
	}

	// Check for CF-IPCountry header from Cloudflare
	if cc := request.Header.Get("CF-IPCountry"); cc != "" && len(cc) == 2 {
		country = strings.ToUpper(cc)
	}

	// Override with query parameters if provided
	if qCountry := request.URL.Query().Get("country"); qCountry != "" && len(qCountry) == 2 {
		country = strings.ToUpper(qCountry)
	}
	if qLanguage := request.URL.Query().Get("language"); qLanguage != "" && len(qLanguage) >= 2 {
		language = qLanguage[0:2]
	}

	return country, language
}
