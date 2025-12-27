package routes

import (
	"net/http"
	"os"

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
