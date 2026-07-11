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

// BuildToolSelectionPrompt builds a meta-prompt asking the model which tools to call.
// Supports parallel tool calls by asking for a JSON array.
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
			requiredSet := map[string]bool{}
			if reqList, ok := tool.Function.Parameters["required"].([]interface{}); ok {
				for _, r := range reqList {
					if s, ok := r.(string); ok {
						requiredSet[s] = true
					}
				}
			}
			for paramName, paramInfo := range params {
				if paramMap, ok := paramInfo.(map[string]interface{}); ok {
					paramType, _ := paramMap["type"].(string)
					paramDesc, _ := paramMap["description"].(string)
					requiredMark := ""
					if requiredSet[paramName] {
						requiredMark = " [required]"
					}
					// Include enum values if present
					enumHint := ""
					if enumVals, ok := paramMap["enum"].([]interface{}); ok && len(enumVals) > 0 {
						enumStrs := make([]string, 0, len(enumVals))
						for _, e := range enumVals {
							enumStrs = append(enumStrs, fmt.Sprintf("%v", e))
						}
						enumHint = fmt.Sprintf(" (one of: %s)", strings.Join(enumStrs, ", "))
					}
					sb.WriteString(fmt.Sprintf("  %s (%s%s)%s: %s\n", paramName, paramType, enumHint, requiredMark, paramDesc))
				}
			}
		}
	}
	sb.WriteString("\nRules:\n")
	sb.WriteString("- You MAY call multiple functions in parallel if the request requires it\n")
	sb.WriteString("- If the request involves reading/listing/searching files or directories, select the appropriate function\n")
	sb.WriteString("- If the request involves writing/creating/modifying files, select the appropriate function\n")
	sb.WriteString("- If the request is a simple conversation, question, or greeting, respond with []\n")
	sb.WriteString("- The request may be in any language; match intent, not exact words\n\n")
	sb.WriteString("Respond with ONLY a JSON array (even for a single call):\n")
	sb.WriteString("[{\"function\":\"tool_name\",\"arguments\":{\"param1\":\"value1\"}}]\n")
	sb.WriteString("If no function is needed: []\n")
	return sb.String()
}

// ParseToolSelectionJSON parses the model response into a list of ToolCallInfo.
// Supports both array format (new) and legacy single-object format for backward compat.
func ParseToolSelectionJSON(text string) []ToolCallInfo {
	// Try array format first
	arrayStart := strings.Index(text, "[")
	arrayEnd := strings.LastIndex(text, "]")
	if arrayStart >= 0 && arrayEnd > arrayStart {
		jsonPart := text[arrayStart : arrayEnd+1]
		var rawList []struct {
			Function  string                 `json:"function"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(jsonPart), &rawList); err == nil {
			result := make([]ToolCallInfo, 0, len(rawList))
			for _, raw := range rawList {
				if raw.Function == "" || raw.Function == "none" {
					continue
				}
				argsBytes, _ := json.Marshal(raw.Arguments)
				result = append(result, ToolCallInfo{
					ID:   "call_" + uuid.New().String()[:12],
					Type: "function",
					Function: FunctionCallInfo{
						Name:      raw.Function,
						Arguments: string(argsBytes),
					},
				})
			}
			return result
		}
	}
	// Fallback: legacy single-object format {"function":"...","arguments":{...}}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		jsonPart := text[start : end+1]
		var raw struct {
			Function  string                 `json:"function"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(jsonPart), &raw); err == nil && raw.Function != "" && raw.Function != "none" {
			argsBytes, _ := json.Marshal(raw.Arguments)
			return []ToolCallInfo{{
				ID:   "call_" + uuid.New().String()[:12],
				Type: "function",
				Function: FunctionCallInfo{
					Name:      raw.Function,
					Arguments: string(argsBytes),
				},
			}}
		}
	}
	return nil
}
