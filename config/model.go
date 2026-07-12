package config

import "strings"

var ModelReverseMap = map[string]string{}
var ModelMap = map[string]string{
	"claude-4.6-sonnet":       "claude46sonnet",
	"claude-4.6-sonnet-think": "claude46sonnetthinking",
	"claude-5.0-sonnet":       "claude50sonnet",
	"claude-5.0-sonnet-think": "claude50sonnetthinking",
	"gpt-5.4":                 "gpt54",
	"gpt-5.4-think":           "gpt54_thinking",
	"gpt-5.5":                 "gpt55",
	"gpt-5.5-think":           "gpt55_thinking",
	"gpt-5.6-sol":             "gpt56_sol",
	"gpt-5.6-sol-think":       "gpt56_sol_thinking",
	"gpt-5.6-terra":           "gpt56_terra",
	"gpt-5.6-terra-think":     "gpt56_terra_thinking",
	"gemini-3.1-pro":          "gemini31pro_high",
	"gemini-2.5-pro":          "gemini25pro",
	"glm-5.2":                 "glm_5_2",
	"turbo":                   "turbo",
}
var MaxModelMap = map[string]string{
	"claude-4.6-opus":       "claude46opus",
	"claude-4.6-opus-think": "claude46opusthinking",
}
var ImageModelMap = map[string]string{
	"gpt-4o-image": "gpt-4o-image",
	"gemini-flash": "gemini-flash",
}

var dashLookup map[string]string

func ModelMapGet(key string, defaultValue string) string {
	if value, exists := ModelMap[key]; exists {
		return value
	}
	if value, exists := dashLookup[key]; exists {
		return value
	}
	return defaultValue
}

func ModelReverseMapGet(key string, defaultValue string) string {
	if value, exists := ModelReverseMap[key]; exists {
		return value
	}
	return defaultValue
}

var ResponseModels []map[string]string

func init() {
	for k, v := range MaxModelMap {
		ModelMap[k] = v
	}

	dashLookup = make(map[string]string)
	for k, v := range ModelMap {
		dashKey := strings.ReplaceAll(k, ".", "-")
		if dashKey != k {
			dashLookup[dashKey] = v
		}
	}

	for k, v := range ModelMap {
		ModelReverseMap[v] = k
	}
	for k, v := range ImageModelMap {
		ModelReverseMap[v] = k
	}
	buildResponseModels()
}

func IsImageModel(modelID string) bool {
	_, exists := ImageModelMap[modelID]
	return exists
}

func buildResponseModels() {
	ResponseModels = make([]map[string]string, 0, (len(ModelMap)+len(ImageModelMap))*2)

	for modelID := range ModelMap {
		if _, isDashAlias := dashLookup[modelID]; isDashAlias {
			continue
		}
		if !ConfigInstance.IsMaxSubscribe {
			if _, isMaxModel := MaxModelMap[modelID]; isMaxModel {
				continue
			}
		}

		ResponseModels = append(ResponseModels, map[string]string{"id": modelID})
		ResponseModels = append(ResponseModels, map[string]string{"id": modelID + "-search"})
	}

	for modelID := range ImageModelMap {
		ResponseModels = append(ResponseModels, map[string]string{"id": modelID})
	}
}
