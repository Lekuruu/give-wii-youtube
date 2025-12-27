package templates

import "bytes"

// RenderFeed renders the feed template with the given data
func (t *Templates) RenderFeed(data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := t.Feed.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderVideoEntry renders the video entry template with the given data
func (t *Templates) RenderVideoEntry(data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := t.VideoEntry.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderSuggestions renders the suggestions template with the given data
func (t *Templates) RenderSuggestions(data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := t.Suggestions.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderError renders the error template with the given message
func (t *Templates) RenderError(message string) (string, error) {
	var buf bytes.Buffer
	data := struct{ Message string }{Message: message}
	if err := t.Error.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
