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

// BuildToolSelectionPrompt constructs a meta-prompt that includes the full JSON
// schema (required fields, types, descriptions) for each tool so the model can
// infer correct argument values. It asks for a JSON array to support parallel
// tool calls; single-function legacy format is still accepted by the parser.
func BuildToolSelectionPrompt(userMessage string, tools []ToolDefinition) string {
	var sb strings.Builder
	sb.WriteString("You are a function selection system. Output ONLY valid JSON, no markdown, no explanation.\n\n")
	sb.WriteString(fmt.Sprintf("User request: \"%s\"\n\n", userMessage))
	sb.WriteString("Available functions (with full JSON Schema):\n")
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("\n### %s\n", tool.Function.Name))
		if tool.Function.Description != "" {
			sb.WriteString(fmt.Sprintf("Description: %s\n", tool.Function.Description))
		}
		if len(tool.Function.Parameters) > 0 {
			schemaBytes, err := json.MarshalIndent(tool.Function.Parameters, "", "  ")
			if err == nil {
				sb.WriteString(fmt.Sprintf("Parameters schema:\n%s\n", string(schemaBytes)))
			}
		}
	}
	sb.WriteString("\nRules:\n")
	sb.WriteString("- You MAY call multiple functions in parallel when needed.\n")
	sb.WriteString("- Fill ALL required parameters; use null only when the schema explicitly allows it.\n")
	sb.WriteString("- The request may be in any language; match intent, not exact words.\n")
	sb.WriteString("- If NO function is needed, respond with: {\"functions\": []}\n\n")
	sb.WriteString("Respond with ONLY this JSON format:\n")
	sb.WriteString("{\"functions\": [{\"name\": \"tool_name\", \"arguments\": {\"param1\": \"value1\"}}]}\n")
	sb.WriteString("For a single call: {\"functions\": [{\"name\": \"read_file\", \"arguments\": {\"path\": \"/foo/bar.go\"}}]}\n")
	sb.WriteString("For parallel calls: {\"functions\": [{\"name\": \"read_file\", \"arguments\": {\"path\": \"a.go\"}}, {\"name\": \"read_file\", \"arguments\": {\"path\": \"b.go\"}}]}\n")
	return sb.String()
}

// ParseToolSelectionJSONMulti parses the model response and returns zero or more
// ToolCallInfo values, supporting both the new array format and the legacy
// single-function format for backward compatibility.
func ParseToolSelectionJSONMulti(text string) []ToolCallInfo {
	start := strings.Index(text, "{")
	if start < 0 {
		return nil
	}
	// Find the matching closing brace for the outermost object
	depth := 0
	end := -1
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return nil
	}
	jsonPart := text[start : end+1]

	// Try new array format first: {"functions": [...]}
	var newFmt struct {
		Functions []struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		} `json:"functions"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &newFmt); err == nil && newFmt.Functions != nil {
		var calls []ToolCallInfo
		for _, f := range newFmt.Functions {
			if f.Name == "" {
				continue
			}
			argsBytes, _ := json.Marshal(f.Arguments)
			calls = append(calls, ToolCallInfo{
				ID:   "call_" + uuid.New().String()[:12],
				Type: "function",
				Function: FunctionCallInfo{
					Name:      f.Name,
					Arguments: string(argsBytes),
				},
			})
		}
		return calls
	}

	// Fallback: legacy single-function format {"function": "name", "arguments": {...}}
	var legacyFmt struct {
		Function  string                 `json:"function"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &legacyFmt); err == nil {
		if legacyFmt.Function == "" || legacyFmt.Function == "none" {
			return nil
		}
		argsBytes, _ := json.Marshal(legacyFmt.Arguments)
		return []ToolCallInfo{{
			ID:   "call_" + uuid.New().String()[:12],
			Type: "function",
			Function: FunctionCallInfo{
				Name:      legacyFmt.Function,
				Arguments: string(argsBytes),
			},
		}}
	}

	return nil
}
