package lang

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed *.txt *.json
var langFS embed.FS

var langTexts map[string]interface{}

func init() {
	langTexts = make(map[string]interface{})
	data, err := langFS.ReadFile("en.json")
	if err == nil {
		json.Unmarshal(data, &langTexts)
	}
}

// GetText retrieves text by key (supports dot notation for nested keys)
// Examples: "welcome_banner", "intro.title", "menu.main.create_note"
func GetText(key string) string {
	// Support both flat keys (backwards compatibility) and dot-notation
	parts := strings.Split(key, ".")
	current := langTexts

	for i, part := range parts {
		if val, ok := current[part]; ok {
			// If it's a string, return it
			if str, ok := val.(string); ok {
				return str
			}
			// If it's a nested map and not the last part, continue
			if m, ok := val.(map[string]interface{}); ok && i < len(parts)-1 {
				current = m
				continue
			}
		}
	}

	// Fallback: return the key itself if not found
	return key
}

// GetTextf retrieves text and formats it with fmt.Sprintf
func GetTextf(key string, args ...interface{}) string {
	return fmt.Sprintf(GetText(key), args...)
}

// GetBanner returns the ASCII art banner
func GetBanner() string {
	data, err := langFS.ReadFile("banner.txt")
	if err == nil {
		return string(data)
	}
	return "Flip - Your Intelligent Assistant"
}

// GetTemplate returns a complete text template file (for longer content like intro, help, etc.)
// Examples: "intro" -> reads intro.en.txt, "help" -> reads help.en.txt
func GetTemplate(name string) string {
	filename := fmt.Sprintf("%s.en.txt", name)
	data, err := langFS.ReadFile(filename)
	if err == nil {
		return string(data)
	}
	// Fallback: return error message
	return fmt.Sprintf("[Template '%s' not found]", name)
}
