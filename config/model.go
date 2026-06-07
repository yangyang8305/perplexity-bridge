package config

import "strings"

var ModelReverseMap = map[string]string{}
var ModelMap = map[string]string{
	"claude-4.6-sonnet":       "claude46sonnet",
	"claude-4.6-sonnet-think": "claude46sonnetthinking",
	"gemini-3.1-pro":          "gemini31pro_high",
	"gpt-5.4":                 "gpt54",
	"gpt-5.4-think":           "gpt54_thinking",
	"gemini-3-pro":            "gemini3pro",
	"grok-4.1":                "grok41",
	"sonar":                   "sonar",
	"sonar-pro":               "sonar-pro",
	"sonar-reasoning":         "sonar_reasoning",
	"sonar-reasoning-pro":     "sonar-reasoning-pro",
	"sonar-deep-research":     "sonar_deep_research",
	"kimi":                    "kimi",
	"kimi-k2":                 "kimi-k2",
	"kimi-k2-think":           "kimi_k2_thinking",
	"gpt-5.2":                 "gpt52",
	"claude-4.5-sonnet":       "claude45sonnet",
	"claude-4.0-sonnet":       "claude-4.0-sonnet",
	"o4-mini":                 "o4-mini",
	"gpt-4o":                  "gpt-4o",
	"gpt-4.1":                 "gpt-4.1",
	"deepseek-r1":             "deepseek-r1",
}
var MaxModelMap = map[string]string{
	"claude-4.6-opus":       "claude46opus",
	"claude-4.6-opus-think": "claude46opusthinking",
}
var ImageModelMap = map[string]string{
	"gpt-4o-image":   "gpt-4o-image",
	"gemini-flash":   "gemini-flash",
	"nano-banana-2":  "nano-banana-2",
	"nano-banana-pro": "nano-banana-pro",
}

// Get returns the value for the given key from the ModelMap.
// If the key doesn't exist, it returns the provided default value.
func ModelMapGet(key string, defaultValue string) string {
	if value, exists := ModelMap[key]; exists {
		return value
	}
	return defaultValue
}

// GetReverse returns the value for the given key from the ModelReverseMap.
// If the key doesn't exist, it returns the provided default value.
func ModelReverseMapGet(key string, defaultValue string) string {
	if value, exists := ModelReverseMap[key]; exists {
		return value
	}
	return defaultValue
}

var ResponseModels []map[string]string

func init() {
	//给modelmap添加max模型
	for k, v := range MaxModelMap {
		ModelMap[k] = v
	}
	// 生成短线别名（opencode等工具会把.转成-）
	dashAliases := map[string]string{}
	for k, v := range ModelMap {
		dashKey := strings.ReplaceAll(k, ".", "-")
		if dashKey != k {
			dashAliases[dashKey] = v
		}
	}
	for k, v := range dashAliases {
		ModelMap[k] = v
	}
	// 构建反向映射
	for k, v := range ModelMap {
		ModelReverseMap[v] = k
	}
	for k, v := range ImageModelMap {
		ModelReverseMap[v] = k
	}
	buildResponseModels()
}

// IsImageModel checks if the given model ID is an image generation model
func IsImageModel(modelID string) bool {
	_, exists := ImageModelMap[modelID]
	return exists
}

// buildResponseModels 构建响应模型列表
func buildResponseModels() {
	ResponseModels = make([]map[string]string, 0, (len(ModelMap)+len(ImageModelMap))*2)

	for modelID := range ModelMap {
		// 如果不是最大订阅用户，跳过最大模型
		if !ConfigInstance.IsMaxSubscribe {
			if _, isMaxModel := MaxModelMap[modelID]; isMaxModel {
				continue
			}
		}

		// 添加普通模型
		ResponseModels = append(ResponseModels, map[string]string{
			"id": modelID,
		})

		// 添加搜索模型
		ResponseModels = append(ResponseModels, map[string]string{
			"id": modelID + "-search",
		})
	}

	// 添加图像生成模型
	for modelID := range ImageModelMap {
		ResponseModels = append(ResponseModels, map[string]string{
			"id": modelID,
		})
	}
}
