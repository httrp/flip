package lang

import (
	"embed"
	"encoding/json"
)

//go:embed *.txt *.json
var langFS embed.FS

var langTexts map[string]string

func GetText(key string) string {
	if langTexts == nil {
		langTexts = make(map[string]string)
		data, err := langFS.ReadFile("en.json")
		if err == nil {
			json.Unmarshal(data, &langTexts)
		}
	}
	if val, ok := langTexts[key]; ok {
		return val
	}
	return key
}

func GetBanner() string {
	data, err := langFS.ReadFile("banner.txt")
	if err == nil {
		return string(data)
	}
	return "Flip - Your Intelligent Assistant"
}
