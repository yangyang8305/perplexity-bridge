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
		Model:   responseModel,
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
// and consistent with streamRespose.
func noStreamResponse(text string, gc *gin.Context) error {
	openAIResp := &OpenAIResponse{
		ID:      uuid.New().String(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   responseModel,
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
//
// #10 fix: stream mode now emits one chunk per tool call, matching the OpenAI
// streaming spec:
//
//	1. role chunk:  delta={"role":"assistant","content":null}
//	2. per-call chunk: delta={"tool_calls":[{index,id,type,function:{name,arguments}}]}
//	3. finish chunk: delta={}, finish_reason="tool_calls"
func ReturnRawToolCallsResponse(calls [][3]string, stream bool, gc *gin.Context) error {
	id := uuid.New().String()
	created := time.Now().Unix()

	if stream {
		gc.Writer.Header().Set("Content-Type", "text/event-stream")
		gc.Writer.Header().Set("Cache-Control", "no-cache")
		gc.Writer.Header().Set("Connection", "keep-alive")
		gc.Writer.WriteHeader(http.StatusOK)
		gc.Writer.Flush()

		// Chunk 1: role announcement
		fmt.Fprintf(gc.Writer,
			"data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\","+
				"\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":null},"+
				"\"logprobs\":null,\"finish_reason\":null}]}\n\n",
			id, created, responseModel)
		gc.Writer.Flush()

		// Chunk 2..N: one chunk per tool call
		for i, call := range calls {
			tcChunk := map[string]interface{}{
				"index": i,
				"id":    call[0],
				"type":  "function",
				"function": map[string]string{
					"name":      call[1],
					"arguments": call[2],
				},
			}
			tcBytes, _ := json.Marshal([]interface{}{tcChunk})
			fmt.Fprintf(gc.Writer,
				"data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\","+
					"\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":%s},"+
					"\"logprobs\":null,\"finish_reason\":null}]}\n\n",
				id, created, responseModel, string(tcBytes))
			gc.Writer.Flush()
		}

		// Final chunk: finish_reason=tool_calls
		fmt.Fprintf(gc.Writer,
			"data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\","+
				"\"choices\":[{\"index\":0,\"delta\":{},"+
				"\"logprobs\":null,\"finish_reason\":\"tool_calls\"}]}\n\n",
			id, created, responseModel)
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
		fmt.Fprintf(gc.Writer,
			`{"id":"%s","object":"chat.completion","created":%d,"model":"%s",`+
				`"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":%s},`+
				`"logprobs":null,"finish_reason":"tool_calls"}],`+
				`"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
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
