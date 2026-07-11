package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pplx2api/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatCompletionRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

type OpenAISrteamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        Delta       `json:"delta"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason interface{} `json:"finish_reason"`
}

type NoStreamChoice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason"`
}

type Delta struct {
	Content string `json:"content"`
}
type Message struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	Refusal    interface{}   `json:"refusal"`
	Annotation []interface{} `json:"annotation"`
}

type OpenAIResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []NoStreamChoice `json:"choices"`
	Usage   Usage            `json:"usage"`
}
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCallInfo mirrors core.ToolCallInfo to avoid import cycle
type ToolCallInfo struct {
	ID       string
	Type     string
	Function ToolCallFunctionInfo
}

type ToolCallFunctionInfo struct {
	Name      string
	Arguments string
}

func ReturnOpenAIResponse(text string, stream bool, gc *gin.Context) error {
	if stream {
		return streamRespose(text, gc)
	} else {
		return noStreamResponse(text, gc)
	}
}

func streamRespose(text string, gc *gin.Context) error {
	openAIResp := &OpenAISrteamResponse{
		ID:      uuid.New().String(),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "claude-3-7-sonnet-20250219",
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: Delta{
					Content: text,
				},
				Logprobs:     nil,
				FinishReason: nil,
			},
		},
	}

	jsonBytes, err := json.Marshal(openAIResp)
	jsonBytes = append([]byte("data: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshalling JSON: %v", err))
		return err
	}

	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()
	return nil
}

func noStreamResponse(text string, gc *gin.Context) error {
	openAIResp := &OpenAIResponse{
		ID:      uuid.New().String(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "claude-3-7-sonnet-20250219",
		Choices: []NoStreamChoice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: text,
				},
				Logprobs:     nil,
				FinishReason: "stop",
			},
		},
	}

	gc.JSON(200, openAIResp)
	return nil
}

// ReturnToolCallResponse returns a single tool call response (kept for backward compat).
func ReturnToolCallResponse(toolCallID, functionName, arguments string, stream bool, gc *gin.Context) error {
	type singleTC struct {
		ID   string
		Name string
		Args string
	}
	return returnToolCallsInternal([]singleTC{{toolCallID, functionName, arguments}}, stream, gc)
}

// ReturnToolCallsResponse returns one or more tool calls in a single response (parallel tool calls).
// The tcs slice elements must have ID, Function.Name, Function.Arguments fields.
func ReturnToolCallsResponse(tcs interface{ Len() int }, stream bool, gc *gin.Context) error {
	// Accept []core.ToolCallInfo via interface — avoid import cycle by using reflection-free approach:
	// tcs is actually passed as a plain slice from handle.go, so we use the concrete type via a type alias.
	// Since we can't import core here, handle.go passes pre-serialized data. See below.
	return nil
}

// ReturnRawToolCallsResponse accepts pre-built tool call data to avoid import cycles.
// Each entry: [id, name, arguments_json_string]
func ReturnRawToolCallsResponse(calls [][3]string, stream bool, gc *gin.Context) error {
	id := uuid.New().String()
	created := time.Now().Unix()
	mdl := "claude-3-7-sonnet-20250219"

	// Build the tool_calls array JSON
	type tcJSON struct {
		Index    int    `json:"index"`
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	tcList := make([]tcJSON, 0, len(calls))
	for i, call := range calls {
		tc := tcJSON{
			Index: i,
			ID:    call[0],
			Type:  "function",
		}
		tc.Function.Name = call[1]
		tc.Function.Arguments = call[2]
		tcList = append(tcList, tc)
	}
	tcBytes, _ := json.Marshal(tcList)

	if stream {
		gc.Writer.Header().Set("Content-Type", "text/event-stream")
		gc.Writer.Header().Set("Cache-Control", "no-cache")
		gc.Writer.Header().Set("Connection", "keep-alive")
		gc.Writer.WriteHeader(http.StatusOK)
		gc.Writer.Flush()

		// Chunk 1: role
		fmt.Fprintf(gc.Writer, "data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":null},\"logprobs\":null,\"finish_reason\":null}]}\n\n", id, created, mdl)
		gc.Writer.Flush()

		// Chunk 2: tool_calls
		fmt.Fprintf(gc.Writer, "data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":%s},\"logprobs\":null,\"finish_reason\":null}]}\n\n", id, created, mdl, string(tcBytes))
		gc.Writer.Flush()

		// Chunk 3: finish
		fmt.Fprintf(gc.Writer, "data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{},\"logprobs\":null,\"finish_reason\":\"tool_calls\"}]}\n\n", id, created, mdl)
		gc.Writer.Flush()

		gc.Writer.Write([]byte("data: [DONE]\n\n"))
		gc.Writer.Flush()
	} else {
		// Build tool_calls for non-stream message
		type tcMsgItem struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		}
		msgTCs := make([]tcMsgItem, 0, len(calls))
		for _, call := range calls {
			item := tcMsgItem{ID: call[0], Type: "function"}
			item.Function.Name = call[1]
			item.Function.Arguments = call[2]
			msgTCs = append(msgTCs, item)
		}
		tcMsgBytes, _ := json.Marshal(msgTCs)

		gc.Writer.Header().Set("Content-Type", "application/json")
		gc.Writer.WriteHeader(http.StatusOK)
		fmt.Fprintf(gc.Writer, `{"id":"%s","object":"chat.completion","created":%d,"model":"%s","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":%s},"logprobs":null,"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
			id, created, mdl, string(tcMsgBytes))
	}
	return nil
}

// returnToolCallsInternal is the internal helper used by ReturnToolCallResponse.
func returnToolCallsInternal(calls []struct {
	ID   string
	Name string
	Args string
}, stream bool, gc *gin.Context) error {
	raw := make([][3]string, len(calls))
	for i, c := range calls {
		raw[i] = [3]string{c.ID, c.Name, c.Args}
	}
	return ReturnRawToolCallsResponse(raw, stream, gc)
}
