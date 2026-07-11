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

// responseModel is the placeholder model name returned to clients.
// B2 fix: was hardcoded inline in both stream and nostream; centralised here.
const responseModel = "perplexity-bridge"

func ReturnOpenAIResponse(text string, stream bool, gc *gin.Context) error {
	if stream {
		return streamRespose(text, gc)
	}
	return noStreamResponse(text, gc)
}

func streamRespose(text string, gc *gin.Context) error {
	openAIResp := &OpenAISrteamResponse{
		ID:      uuid.New().String(),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   responseModel, // B2 fix
		Choices: []StreamChoice{
			{
				Index:        0,
				Delta:        Delta{Content: text},
				Logprobs:     nil,
				FinishReason: nil,
			},
		},
	}

	jsonBytes, err := json.Marshal(openAIResp)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshalling JSON: %v", err))
		return err
	}
	jsonBytes = append([]byte("data: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)
	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()
	return nil
}

// B7 fix: use gc.Writer instead of gc.JSON so the response is flushable
// and consistent with streamRespose. gc.JSON calls WriteHeader internally
// which would conflict if headers were already sent in a stream scenario.
func noStreamResponse(text string, gc *gin.Context) error {
	openAIResp := &OpenAIResponse{
		ID:      uuid.New().String(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   responseModel, // B2 fix
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
	jsonBytes, err := json.Marshal(openAIResp)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshalling JSON: %v", err))
		return err
	}
	gc.Writer.Header().Set("Content-Type", "application/json")
	gc.Writer.WriteHeader(http.StatusOK)
	gc.Writer.Write(jsonBytes)
	return nil
}

// ReturnRawToolCallsResponse accepts pre-built tool call data to avoid import cycles.
// Each entry: [id, name, arguments_json_string]
func ReturnRawToolCallsResponse(calls [][3]string, stream bool, gc *gin.Context) error {
	id := uuid.New().String()
	created := time.Now().Unix()

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
		tc := tcJSON{Index: i, ID: call[0], Type: "function"}
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

		fmt.Fprintf(gc.Writer, "data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":null},\"logprobs\":null,\"finish_reason\":null}]}\n\n", id, created, responseModel)
		gc.Writer.Flush()
		fmt.Fprintf(gc.Writer, "data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":%s},\"logprobs\":null,\"finish_reason\":null}]}\n\n", id, created, responseModel, string(tcBytes))
		gc.Writer.Flush()
		fmt.Fprintf(gc.Writer, "data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{},\"logprobs\":null,\"finish_reason\":\"tool_calls\"}]}\n\n", id, created, responseModel)
		gc.Writer.Flush()
		gc.Writer.Write([]byte("data: [DONE]\n\n"))
		gc.Writer.Flush()
	} else {
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
			id, created, responseModel, string(tcMsgBytes))
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

// ReturnToolCallResponse returns a single tool call response.
func ReturnToolCallResponse(toolCallID, functionName, arguments string, stream bool, gc *gin.Context) error {
	return returnToolCallsInternal([]struct{ ID, Name, Args string }{{toolCallID, functionName, arguments}}, stream, gc)
}
