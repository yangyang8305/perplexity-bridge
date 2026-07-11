package service

import (
	"fmt"
	"net/http"
	"pplx2api/config"
	"pplx2api/core"
	"pplx2api/logger"
	"pplx2api/model"
	"pplx2api/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

type ChatCompletionRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func ChatCompletionsHandler(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No messages provided"})
		return
	}

	// Guard: refuse early if no sessions are configured (avoids panic in NextIndex / modulo-by-zero).
	if len(config.ConfigInstance.Sessions) == 0 {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "No SESSIONS configured"})
		return
	}

	m := req.Model
	if m == "" {
		m = "claude-4.6-sonnet"
	}
	if idx := strings.LastIndex(m, "/"); idx >= 0 {
		m = m[idx+1:]
	}
	openSearch := false
	if strings.HasSuffix(m, "-search") {
		openSearch = true
		m = strings.TrimSuffix(m, "-search")
	}
	m = config.ModelMapGet(m, m)

	hasTools := len(req.Tools) > 0
	hasToolResults := core.HasToolRoleMessages(req.Messages)

	var prompt strings.Builder
	img_data_list := []string{}
	toolDefs := core.ConvertTools(req.Tools)

	for _, msg := range req.Messages {
		role, roleOk := msg["role"].(string)
		if !roleOk {
			continue
		}
		if role == "system" {
			content, _ := msg["content"].(string)
			prompt.WriteString("System: ")
			prompt.WriteString(content)
			prompt.WriteString("\n\n")
			continue
		}
		if role == "tool" {
			toolCallID := core.ExtractToolCallID(msg)
			content, _ := msg["content"].(string)
			prompt.WriteString(fmt.Sprintf("[Tool result for call %s]\n%s\n\n", toolCallID, content))
			continue
		}
		if role == "assistant" {
			if toolCalls, hasTC := msg["tool_calls"]; hasTC && toolCalls != nil {
				tcList, ok := toolCalls.([]interface{})
				if ok && len(tcList) > 0 {
					prompt.WriteString("Assistant called:")
					for _, tc := range tcList {
						tcMap, ok := tc.(map[string]interface{})
						if !ok {
							continue
						}
						tcID, _ := tcMap["id"].(string)
						funcMap, ok := tcMap["function"].(map[string]interface{})
						if !ok {
							continue
						}
						name, _ := funcMap["name"].(string)
						args, _ := funcMap["arguments"].(string)
						prompt.WriteString(fmt.Sprintf(" %s(id=%s, args=%s)", name, tcID, args))
					}
					prompt.WriteString("\n\n")
				}
			} else {
				prompt.WriteString(utils.GetRolePrefix(role))
				if content, exists := msg["content"]; exists {
					switch v := content.(type) {
					case string:
						prompt.WriteString(v)
					}
				}
				prompt.WriteString("\n\n")
			}
			continue
		}

		prompt.WriteString(utils.GetRolePrefix(role))
		content, exists := msg["content"]
		if !exists {
			continue
		}
		switch v := content.(type) {
		case string:
			prompt.WriteString(v + "\n\n")
		case []interface{}:
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if itemType, ok := itemMap["type"].(string); ok {
						if itemType == "text" {
							if text, ok := itemMap["text"].(string); ok {
								prompt.WriteString(text + "\n\n")
							}
						} else if itemType == "image_url" {
							if imageUrl, ok := itemMap["image_url"].(map[string]interface{}); ok {
								if url, ok := imageUrl["url"].(string); ok {
									// B5 fix: safe data-URI extraction — avoid panic when no comma
									if strings.HasPrefix(url, "data:image/") {
										if idx := strings.Index(url, ","); idx >= 0 {
											url = url[idx+1:]
										} else {
											logger.Warn("image_url data-URI has no comma, skipping")
											continue
										}
									}
									img_data_list = append(img_data_list, url)
								}
							}
						}
					}
				}
			}
		}
	}

	if !hasToolResults {
		prompt.WriteString("Assistant: ")
	} else {
		prompt.WriteString("\n(Based on the tool results above, provide the final answer to the user's request. Be direct and complete.)\nAssistant: ")
	}

	// B4 fix: capture rootPrompt BEFORE any loop mutation (UploadText resets prompt)
	rootPromptStr := prompt.String()

	var pplxClient *core.Client
	// B1 fix: get starting index once; loop increments before use so index 0 is included
	startIndex := config.Sr.NextIndex()
	for i := 0; i < config.ConfigInstance.RetryCount; i++ {
		// Reset prompt to the original pre-upload version on every attempt
		prompt.Reset()
		prompt.WriteString(rootPromptStr)

		// B1 fix: use (startIndex + i) % len so first attempt uses startIndex, not startIndex+1
		index := (startIndex + i) % len(config.ConfigInstance.Sessions)
		session, err := config.ConfigInstance.GetSessionForModel(index)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get session for model %s: %v", m, err))
			continue
		}
		logger.Info(fmt.Sprintf("Using session index=%d model=%s", index, m))
		pplxClient = core.NewClient(session.SessionKey, config.ConfigInstance.Proxy, m, openSearch)

		if len(img_data_list) > 0 {
			if err := pplxClient.UploadImage(img_data_list); err != nil {
				logger.Error(fmt.Sprintf("Failed to upload images: %v", err))
				continue
			}
		}
		if prompt.Len() > config.ConfigInstance.MaxChatHistoryLength {
			if err := pplxClient.UploadText(prompt.String()); err != nil {
				logger.Error(fmt.Sprintf("Failed to upload text: %v", err))
				continue
			}
			prompt.Reset()
			prompt.WriteString(config.ConfigInstance.PromptForFile)
		}

		if hasTools && !hasToolResults {
			userMsg := core.GetLastUserMessage(req.Messages)
			toolCalls, err := pplxClient.DetermineToolCalls(userMsg, toolDefs)
			if err != nil {
				logger.Error(fmt.Sprintf("Tool determination failed: %v — falling back to direct answer", err))
				if _, err := pplxClient.SendMessage(prompt.String(), req.Stream, config.ConfigInstance.IsIncognito, c); err != nil {
					logger.Error(fmt.Sprintf("Fallback send failed: %v", err))
					continue
				}
				return
			}
			if len(toolCalls) > 0 {
				for _, tc := range toolCalls {
					logger.Info(fmt.Sprintf("Tool call: %s args=%s", tc.Function.Name, tc.Function.Arguments))
				}
				rawCalls := make([][3]string, 0, len(toolCalls))
				for _, tc := range toolCalls {
					rawCalls = append(rawCalls, [3]string{tc.ID, tc.Function.Name, tc.Function.Arguments})
				}
				model.ReturnRawToolCallsResponse(rawCalls, req.Stream, c)
			} else {
				logger.Info("No tool needed — answering directly")
				if _, err := pplxClient.SendMessage(prompt.String(), req.Stream, config.ConfigInstance.IsIncognito, c); err != nil {
					logger.Error(fmt.Sprintf("Failed to send message: %v", err))
					continue
				}
			}
		} else if hasTools && hasToolResults {
			if req.Stream {
				if statusCode, err := pplxClient.SendMessage(prompt.String(), true, config.ConfigInstance.IsIncognito, c); err != nil {
					logger.Error(fmt.Sprintf("Tool result stream failed: status=%d err=%v", statusCode, err))
					continue
				}
			} else {
				text, err := pplxClient.SendMessageCollect(prompt.String(), config.ConfigInstance.IsIncognito)
				if err != nil {
					logger.Error(fmt.Sprintf("Tool result collect failed: %v", err))
					continue
				}
				model.ReturnOpenAIResponse(text, req.Stream, c)
			}
		} else {
			if _, err := pplxClient.SendMessage(prompt.String(), req.Stream, config.ConfigInstance.IsIncognito, c); err != nil {
				logger.Error(fmt.Sprintf("Failed to send message: %v", err))
				continue
			}
		}
		return
	}
	logger.Error("All retries exhausted")
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: "Failed to process request after multiple attempts",
	})
}

func ModelsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": config.ResponseModels})
}
