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

// OpenAISrteamResponse 定义 OpenAI 的流式响应结构
type OpenAISrteamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// Choice 结构表示 OpenAI 返回的单个选项
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

// Delta 结构用于存储返回的文本内容
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

	// 发送数据
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

func ReturnToolCallResponse(toolCallID, functionName, arguments string, stream bool, gc *gin.Context) error {
	id := uuid.New().String()
	created := time.Now().Unix()
	model := "claude-3-7-sonnet-20250219"
	argsEscaped, _ := json.Marshal(arguments)
	if stream {
		gc.Writer.Header().Set("Content-Type", "text/event-stream")
		gc.Writer.Header().Set("Cache-Control", "no-cache")
		gc.Writer.Header().Set("Connection", "keep-alive")
		gc.Writer.WriteHeader(http.StatusOK)
		gc.Writer.Flush()

		fmt.Fprintf(gc.Writer, `data: {"id":"%s","object":"chat.completion.chunk","created":%d,"model":"%s","choices":[{"index":0,"delta":{"role":"assistant","content":null},"logprobs":null,"finish_reason":null}]}`+"\n\n", id, created, model)
		gc.Writer.Flush()

		fmt.Fprintf(gc.Writer, `data: {"id":"%s","object":"chat.completion.chunk","created":%d,"model":"%s","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"%s","type":"function","function":{"name":"%s","arguments":%s}}]},"logprobs":null,"finish_reason":null}]}`+"\n\n", id, created, model, toolCallID, functionName, string(argsEscaped))
		gc.Writer.Flush()

		fmt.Fprintf(gc.Writer, `data: {"id":"%s","object":"chat.completion.chunk","created":%d,"model":"%s","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"tool_calls"}]}`+"\n\n", id, created, model)
		gc.Writer.Flush()

		gc.Writer.Write([]byte("data: [DONE]\n\n"))
		gc.Writer.Flush()
	} else {
		gc.Writer.Header().Set("Content-Type", "application/json")
		gc.Writer.WriteHeader(http.StatusOK)
		fmt.Fprintf(gc.Writer, `{"id":"%s","object":"chat.completion","created":%d,"model":"%s","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"%s","type":"function","function":{"name":"%s","arguments":%s}}]},"logprobs":null,"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`, id, created, model, toolCallID, functionName, string(argsEscaped))
	}
	return nil
}
