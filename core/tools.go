package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCallInfo struct {
	ID       string
	Type     string
	Function FunctionCallInfo
}

type FunctionCallInfo struct {
	Name      string
	Arguments string
}

func HasToolRoleMessages(messages []map[string]interface{}) bool {
	for _, msg := range messages {
		if role, ok := msg["role"].(string); ok && role == "tool" {
			return true
		}
	}
	return false
}

func ExtractToolCallID(msg map[string]interface{}) string {
	if id, ok := msg["tool_call_id"].(string); ok {
		return id
	}
	return ""
}

func ConvertTools(rawTools []map[string]interface{}) []ToolDefinition {
	tools := make([]ToolDefinition, 0, len(rawTools))
	for _, t := range rawTools {
		tool := ToolDefinition{}
		if ttype, ok := t["type"].(string); ok {
			tool.Type = ttype
		}
		if funcRaw, ok := t["function"].(map[string]interface{}); ok {
			if name, ok := funcRaw["name"].(string); ok {
				tool.Function.Name = name
			}
			if desc, ok := funcRaw["description"].(string); ok {
				tool.Function.Description = desc
			}
			if params, ok := funcRaw["parameters"].(map[string]interface{}); ok {
				tool.Function.Parameters = params
			}
		}
		tools = append(tools, tool)
	}
	return tools
}

func GetLastUserMessage(messages []map[string]interface{}) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if role, ok := messages[i]["role"].(string); ok && role == "user" {
			content, exists := messages[i]["content"]
			if !exists {
				continue
			}
			switch v := content.(type) {
			case string:
				return v
			case []interface{}:
				for _, item := range v {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if t, ok := itemMap["type"].(string); ok && t == "text" {
							if text, ok := itemMap["text"].(string); ok {
								return text
							}
						}
					}
				}
			}
		}
	}
	return ""
}

func BuildToolSelectionPrompt(userMessage string, tools []ToolDefinition) string {
	var sb strings.Builder
	sb.WriteString("You are a function selection system. Output ONLY valid JSON.\n\n")
	sb.WriteString(fmt.Sprintf("User request: \"%s\"\n\n", userMessage))
	sb.WriteString("Available functions:\n")
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- %s", tool.Function.Name))
		if tool.Function.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", tool.Function.Description))
		}
		sb.WriteString("\n")
		if params, ok := tool.Function.Parameters["properties"].(map[string]interface{}); ok {
			for paramName, paramInfo := range params {
				if paramMap, ok := paramInfo.(map[string]interface{}); ok {
					paramType, _ := paramMap["type"].(string)
					paramDesc, _ := paramMap["description"].(string)
					sb.WriteString(fmt.Sprintf("  %s (%s): %s\n", paramName, paramType, paramDesc))
				}
			}
		}
	}
	sb.WriteString("\nRules:\n")
	sb.WriteString("- If the request involves reading/listing/searching files or directories, select the appropriate function\n")
	sb.WriteString("- If the request involves writing/creating/modifying files, select the appropriate function\n")
	sb.WriteString("- If the request is a simple conversation, question, or greeting, respond with {\"function\":\"none\"}\n")
	sb.WriteString("- The request may be in any language; match intent, not exact words\n\n")
	sb.WriteString("Respond with ONLY this JSON:\n")
	sb.WriteString("If a function should be called: {\"function\":\"tool_name\",\"arguments\":{\"param1\":\"value1\"}}\n")
	sb.WriteString("If no function is needed: {\"function\":\"none\"}\n")
	return sb.String()
}

func ParseToolSelectionJSON(text string) *ToolCallInfo {
	// Find JSON in the text (handle possible surrounding whitespace or markdown)
	start := strings.Index(text, "{")
	if start < 0 {
		return nil
	}
	end := strings.LastIndex(text, "}")
	if end < 0 || end <= start {
		return nil
	}
	jsonPart := text[start : end+1]
	var raw struct {
		Function  string                 `json:"function"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &raw); err != nil {
		return nil
	}
	if raw.Function == "" || raw.Function == "none" {
		return nil
	}
	argsBytes, _ := json.Marshal(raw.Arguments)
	return &ToolCallInfo{
		ID:   "call_" + uuid.New().String()[:12],
		Type: "function",
		Function: FunctionCallInfo{
			Name:      raw.Function,
			Arguments: string(argsBytes),
		},
	}
}
