package module

// cfgString extracts a string value from a raw config map, returning defaultVal if absent or wrong type.
func cfgString(cfg map[string]interface{}, key, defaultVal string) string {
	v, ok := cfg[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}

// cfgStrings extracts a string slice from a raw config map.
// Handles []interface{}, []string, or a bare string (treated as single-element slice).
func cfgStrings(cfg map[string]interface{}, key string) []string {
	v, ok := cfg[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []interface{}:
		result := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		return []string{t}
	}
	return nil
}

// cfgInt extracts an integer from a raw config map.
// Handles both int and float64 (YAML numbers decode as int or float64 depending on context).
func cfgInt(cfg map[string]interface{}, key string, defaultVal int) int {
	v, ok := cfg[key]
	if !ok {
		return defaultVal
	}
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	}
	return defaultVal
}
