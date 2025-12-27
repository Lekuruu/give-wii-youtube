package providers

import "fmt"

// getString safely extracts a string value
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getInt safely extracts an integer value
func getInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var result int
		fmt.Sscanf(v, "%d", &result)
		return result
	}
	return 0
}

// getInt64 safely extracts an int64 value
func getInt64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		var result int64
		fmt.Sscanf(v, "%d", &result)
		return result
	}
	return 0
}

// getBool safely extracts a boolean value
func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// getMap safely extracts a nested map
func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// getNestedMap traverses a path of keys to get a nested map
func getNestedMap(m map[string]interface{}, keys ...string) map[string]interface{} {
	current := m
	for i, key := range keys {
		// Handle array index access (e.g., "0" for first element)
		if key == "0" {
			if arr, ok := current[keys[i-1]].([]interface{}); ok && len(arr) > 0 {
				if next, ok := arr[0].(map[string]interface{}); ok {
					current = next
					continue
				}
			}
			return nil
		}

		next, ok := current[key].(map[string]interface{})
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

// getNestedString traverses a path of keys to get a string value
func getNestedString(m map[string]interface{}, keys ...string) string {
	if len(keys) == 0 {
		return ""
	}

	current := m
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	return getString(current, keys[len(keys)-1])
}

// getNestedSlice traverses a path of keys to get a slice
func getNestedSlice(m map[string]interface{}, keys ...string) []interface{} {
	if len(keys) == 0 {
		return nil
	}

	current := m
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return nil
		}
		current = next
	}

	if v, ok := current[keys[len(keys)-1]].([]interface{}); ok {
		return v
	}
	return nil
}

// getRunsText extracts text from a "runs" array structure
func getRunsText(m map[string]interface{}, key string) string {
	container, ok := m[key].(map[string]interface{})
	if !ok {
		return ""
	}

	runs, ok := container["runs"].([]interface{})
	if !ok || len(runs) == 0 {
		return ""
	}

	firstRun, ok := runs[0].(map[string]interface{})
	if !ok {
		return ""
	}

	return getString(firstRun, "text")
}

// extractThumbnails extracts thumbnails from a container with a "thumbnail" field
func extractThumbnails(m map[string]interface{}) []Thumbnail {
	var thumbnails []Thumbnail

	thumbnail, ok := m["thumbnail"].(map[string]interface{})
	if !ok {
		return thumbnails
	}

	thumbs, ok := thumbnail["thumbnails"].([]interface{})
	if !ok {
		return thumbnails
	}

	for _, t := range thumbs {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		thumbnails = append(thumbnails, Thumbnail{
			URL:    getString(tm, "url"),
			Width:  getInt(tm, "width"),
			Height: getInt(tm, "height"),
		})
	}

	return thumbnails
}
