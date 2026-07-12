package core

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"pplx2api/config"
	"pplx2api/logger"
	"pplx2api/model"
	"pplx2api/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/imroc/req/v3"
)

type Client struct {
	sessionToken string
	visitorID    string
	client       *req.Client
	Model        string
	Attachments  []string
	OpenSerch    bool
}

type PerplexityRequest struct {
	Params   PerplexityParams `json:"params"`
	QueryStr string           `json:"query_str"`
}

type PerplexityParams struct {
	Attachments             []string      `json:"attachments"`
	Language                string        `json:"language"`
	Timezone                string        `json:"timezone"`
	SearchFocus             string        `json:"search_focus"`
	Sources                 []string      `json:"sources"`
	SearchRecencyFilter     interface{}   `json:"search_recency_filter"`
	FrontendUUID            string        `json:"frontend_uuid"`
	Mode                    string        `json:"mode"`
	ModelPreference         string        `json:"model_preference"`
	IsRelatedQuery          bool          `json:"is_related_query"`
	IsSponsored             bool          `json:"is_sponsored"`
	VisitorID               string        `json:"visitor_id"`
	UserNextauthID          string        `json:"user_nextauth_id"`
	FrontendContextUUID     string        `json:"frontend_context_uuid"`
	PromptSource            string        `json:"prompt_source"`
	QuerySource             string        `json:"query_source"`
	BrowserHistorySummary   []interface{} `json:"browser_history_summary"`
	IsIncognito             bool          `json:"is_incognito"`
	UseSchematizedAPI       bool          `json:"use_schematized_api"`
	SendBackTextInStreaming bool          `json:"send_back_text_in_streaming_api"`
	SupportedBlockUseCases  []string      `json:"supported_block_use_cases"`
	ClientCoordinates       interface{}   `json:"client_coordinates"`
	IsNavSuggestionsDisabled bool         `json:"is_nav_suggestions_disabled"`
	Version                 string        `json:"version"`
}

type PerplexityResponse struct {
	Blocks       []Block `json:"blocks"`
	Status       string  `json:"status"`
	DisplayModel string  `json:"display_model"`
}

type Block struct {
	MarkdownBlock      *MarkdownBlock      `json:"markdown_block,omitempty"`
	ReasoningPlanBlock *ReasoningPlanBlock `json:"reasoning_plan_block,omitempty"`
	WebResultBlock     *WebResultBlock     `json:"web_result_block,omitempty"`
	ImageModeBlock     *ImageModeBlock     `json:"image_mode_block,omitempty"`
	IntendedUsage      string              `json:"intended_usage"`
}

type MarkdownBlock struct {
	Chunks []string `json:"chunks"`
}

type ReasoningPlanBlock struct {
	Goals []Goal `json:"goals"`
}

type Goal struct {
	Description string `json:"description"`
}

type WebResultBlock struct {
	WebResults []WebResult `json:"web_results"`
}

type WebResult struct {
	Name    string `json:"name"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

type ImageModeBlock struct {
	AnswerModeType string `json:"answer_mode_type"`
	Progress       string `json:"progress"`
	MediaItems     []struct {
		Medium    string `json:"medium"`
		Image     string `json:"image"`
		URL       string `json:"url"`
		Name      string `json:"name"`
		Source    string `json:"source"`
		Thumbnail string `json:"thumbnail"`
	} `json:"media_items"`
}

const pplxVersion = "2.18"

// ErrSessionExpired is returned when Perplexity signals the session is invalid.
// A7 fix: callers can detect this and skip the dead session.
var ErrSessionExpired = fmt.Errorf("session expired or invalid")

func NewClient(sessionToken string, proxy string, mdl string, openSerch bool) *Client {
	client := req.C().SetTimeout(time.Minute * 10)
	client.Transport.SetResponseHeaderTimeout(time.Second * 10)
	if proxy != "" {
		client.SetProxyURL(proxy)
	}
	headers := map[string]string{
		"accept":          "text/event-stream, text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"accept-language": "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,zh-TW;q=0.6",
		"accept-encoding": "gzip, deflate",
		"cache-control":   "no-cache",
		"origin":          "https://www.perplexity.ai",
		"pragma":          "no-cache",
		"priority":        "u=1, i",
		"referer":         "https://www.perplexity.ai/",
	}
	for k, v := range headers {
		client.SetCommonHeader(k, v)
	}
	if sessionToken != "" {
		client.SetCommonCookies(&http.Cookie{
			Name:  "__Secure-next-auth.session-token",
			Value: sessionToken,
		})
	}
	return &Client{
		sessionToken: sessionToken,
		visitorID:    uuid.New().String(),
		client:       client,
		Model:        mdl,
		Attachments:  []string{},
		OpenSerch:    openSerch,
	}
}

func redactToken(token string) string {
	if len(token) <= 8 {
		return "[REDACTED]"
	}
	return token[:8] + "...[REDACTED]"
}

// timezone 返回配置的时区，默认 America/New_York。
// #7 fix: 加 RLock 保护并发读取。
func timezone() string {
	config.ConfigInstance.RwMutex.RLock()
	tz := config.ConfigInstance.Timezone
	config.ConfigInstance.RwMutex.RUnlock()
	if tz != "" {
		return tz
	}
	return "America/New_York"
}

func (c *Client) buildRequestBody(message string, isIncognito bool, stream bool) PerplexityRequest {
	searchFocus := "writing"
	sources := []string{}
	if c.OpenSerch {
		searchFocus = "internet"
		sources = append(sources, "web")
	}
	return PerplexityRequest{
		Params: PerplexityParams{
			Attachments:             c.Attachments,
			Language:                "en-US",
			Timezone:                timezone(),
			SearchFocus:             searchFocus,
			Sources:                 sources,
			SearchRecencyFilter:     nil,
			FrontendUUID:            uuid.New().String(),
			Mode:                    "copilot",
			ModelPreference:         c.Model,
			IsRelatedQuery:          false,
			IsSponsored:             false,
			VisitorID:               c.visitorID,
			UserNextauthID:          c.visitorID,
			FrontendContextUUID:     uuid.New().String(),
			PromptSource:            "user",
			QuerySource:             "home",
			BrowserHistorySummary:   []interface{}{},
			IsIncognito:             isIncognito,
			UseSchematizedAPI:       true,
			SendBackTextInStreaming: stream,
			SupportedBlockUseCases: []string{
				"answer_modes", "media_items", "knowledge_cards",
				"inline_entity_cards", "place_widgets", "finance_widgets",
				"sports_widgets", "shopping_widgets", "jobs_widgets",
				"search_result_widgets", "entity_list_answer", "todo_list",
			},
			ClientCoordinates:        nil,
			IsNavSuggestionsDisabled: false,
			Version:                  pplxVersion,
		},
		QueryStr: message,
	}
}

// isSessionExpiredResponse checks whether the HTTP response indicates the
// session cookie is no longer valid (Perplexity returns 401 or redirects to
// the login page when the token has expired).
// A7 fix.
func isSessionExpiredResponse(resp *req.Response) bool {
	if resp.StatusCode == http.StatusUnauthorized {
		return true
	}
	// Perplexity may also 302-redirect to /api/auth/signin
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		loc := resp.Header.Get("Location")
		if strings.Contains(loc, "signin") || strings.Contains(loc, "login") {
			return true
		}
	}
	return false
}

func (c *Client) SendMessage(message string, stream bool, is_incognito bool, gc *gin.Context) (int, error) {
	requestBody := c.buildRequestBody(message, is_incognito, stream)
	logger.Info(fmt.Sprintf("Perplexity request: model=%s search=%v incognito=%v session=%s",
		c.Model, c.OpenSerch, is_incognito, redactToken(c.sessionToken)))

	resp, err := c.client.R().DisableAutoReadResponse().
		SetBody(requestBody).
		Post("https://www.perplexity.ai/rest/sse/perplexity_ask")
	if err != nil {
		logger.Error(fmt.Sprintf("Error sending request: %v", err))
		return 500, fmt.Errorf("request failed: %w", err)
	}
	logger.Info(fmt.Sprintf("Perplexity response status: %d", resp.StatusCode))

	// A7 fix: detect expired/invalid session and signal caller to skip this session
	if isSessionExpiredResponse(resp) {
		resp.Body.Close()
		logger.Warn(fmt.Sprintf("Session expired for token %s", redactToken(c.sessionToken)))
		return http.StatusUnauthorized, ErrSessionExpired
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		return http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Unexpected status: %d", resp.StatusCode))
		resp.Body.Close()
		return resp.StatusCode, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return 200, c.HandleResponse(resp.Body, stream, gc)
}

func (c *Client) SendMessageCollect(message string, is_incognito bool) (string, error) {
	requestBody := c.buildRequestBody(message, is_incognito, false)
	resp, err := c.client.R().DisableAutoReadResponse().
		SetBody(requestBody).
		Post("https://www.perplexity.ai/rest/sse/perplexity_ask")
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if isSessionExpiredResponse(resp) {
		logger.Warn(fmt.Sprintf("Session expired for token %s", redactToken(c.sessionToken)))
		return "", ErrSessionExpired
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return c.collectResponse(resp.Body)
}

// DetermineToolCalls runs a two-phase tool selection when the tool list exceeds
// MaxToolsPerRound, otherwise uses a single-phase approach.
func (c *Client) DetermineToolCalls(userMessage string, tools []ToolDefinition) ([]ToolCallInfo, error) {
	if len(tools) <= MaxToolsPerRound {
		prompt := BuildToolSelectionPrompt(userMessage, tools)
		text, err := c.SendMessageCollect(prompt, true)
		if err != nil {
			return nil, err
		}
		return ParseToolSelectionJSONMulti(text), nil
	}

	phase1Prompt := BuildToolNameSelectionPrompt(userMessage, tools)
	phase1Text, err := c.SendMessageCollect(phase1Prompt, true)
	if err != nil {
		return nil, fmt.Errorf("tool selection phase 1 failed: %w", err)
	}
	selectedNames, err := ParseToolNames(phase1Text)
	if err != nil {
		return nil, fmt.Errorf("tool selection phase 1 parse error: %w", err)
	}
	if len(selectedNames) == 0 {
		return nil, nil
	}

	selectedTools := FilterToolsByNames(tools, selectedNames)
	if len(selectedTools) == 0 {
		return nil, fmt.Errorf("tool selection phase 1 returned unknown tool names: %v", selectedNames)
	}
	phase2Prompt := BuildToolSelectionPrompt(userMessage, selectedTools)
	phase2Text, err := c.SendMessageCollect(phase2Prompt, true)
	if err != nil {
		return nil, fmt.Errorf("tool selection phase 2 failed: %w", err)
	}
	return ParseToolSelectionJSONMulti(phase2Text), nil
}

func (c *Client) collectResponse(body io.ReadCloser) (string, error) {
	defer body.Close()
	reader := bufio.NewReaderSize(body, 1024*1024)
	full_text := ""
	inThinking := false
	thinkShown := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		var response PerplexityResponse
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			continue
		}
		if response.Status == "COMPLETED" {
			if inThinking { full_text += "</think>\n\n" }
			inThinking = false
			thinkShown = true
		}
		for _, block := range response.Blocks {
			if block.ReasoningPlanBlock != nil && len(block.ReasoningPlanBlock.Goals) > 0 {
				res_text := ""
				if !inThinking && !thinkShown {
					res_text += "<think>"
					inThinking = true
				}
				for _, goal := range block.ReasoningPlanBlock.Goals {
					if goal.Description != "" && goal.Description != "Beginning analysis" && goal.Description != "Wrapping up analysis" {
						res_text += goal.Description
					}
				}
				full_text += res_text
			}
		}
		for _, block := range response.Blocks {
			if block.MarkdownBlock != nil && len(block.MarkdownBlock.Chunks) > 0 && block.IntendedUsage == "ask_text_0_markdown" {
				res_text := ""
				if inThinking {
					res_text += "</think>\n\n"
					inThinking = false
					thinkShown = true
				}
				for _, chunk := range block.MarkdownBlock.Chunks {
					if chunk != "" {
						res_text += chunk
					}
				}
				full_text += res_text
			}
		}
	}
	return full_text, nil
}

func (c *Client) HandleResponse(body io.ReadCloser, stream bool, gc *gin.Context) error {
	defer body.Close()
	if stream {
		gc.Writer.Header().Set("Content-Type", "text/event-stream")
		gc.Writer.Header().Set("Cache-Control", "no-cache")
		gc.Writer.Header().Set("Connection", "keep-alive")
		gc.Writer.WriteHeader(http.StatusOK)
		gc.Writer.Flush()
	}
	reader := bufio.NewReaderSize(body, 1024*1024)
	clientDone := gc.Request.Context().Done()
	full_text := ""
	inThinking := false
	thinkShown := false
	final := false

	// #7 fix: 快照需要的 Config 字段，避免在流式循环中反复加锁
	config.ConfigInstance.RwMutex.RLock()
	ignoreSearch := config.ConfigInstance.IgnoreSerchResult
	ignoreMonitor := config.ConfigInstance.IgnoreModelMonitoring
	config.ConfigInstance.RwMutex.RUnlock()

	for {
		select {
		case <-clientDone:
			logger.Info("Client connection closed")
			return nil
		default:
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error(fmt.Sprintf("Error reading line: %v", err))
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		var response PerplexityResponse
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			logger.Error(fmt.Sprintf("Error parsing JSON: %v", err))
			continue
		}
		if response.Status == "COMPLETED" {
			final = true
			if inThinking {
				closeTag := "</think>\n\n"
				full_text += closeTag
				if stream {
					model.ReturnOpenAIResponse(closeTag, stream, gc)
				}
				inThinking = false
			}
			for _, block := range response.Blocks {
				if block.ImageModeBlock != nil && block.ImageModeBlock.Progress == "DONE" && len(block.ImageModeBlock.MediaItems) > 0 {
					imageResultsText := ""
					imageModelList := []string{}
					for i, result := range block.ImageModeBlock.MediaItems {
						imageResultsText += utils.ImageShow(i, result.Name, result.Image)
						imageModelList = append(imageModelList, result.Name)
					}
					if len(imageModelList) > 0 {
						imageResultsText = imageResultsText + "\n\n---\n" + strings.Join(imageModelList, ", ")
					}
					full_text += imageResultsText
					if stream {
						model.ReturnOpenAIResponse(imageResultsText, stream, gc)
					}
				}
			}
			for _, block := range response.Blocks {
				if !ignoreSearch && block.WebResultBlock != nil && len(block.WebResultBlock.WebResults) > 0 {
					webResultsText := "\n\n---\n"
					for i, result := range block.WebResultBlock.WebResults {
						webResultsText += "\n\n" + utils.SearchShow(i, result.Name, result.URL, result.Snippet)
					}
					full_text += webResultsText
					if stream {
						model.ReturnOpenAIResponse(webResultsText, stream, gc)
					}
				}
			}
			if !ignoreMonitor && response.DisplayModel != c.Model {
				logger.Warn(fmt.Sprintf("Model drift: requested=%s actual=%s",
					c.Model, config.ModelReverseMapGet(response.DisplayModel, response.DisplayModel)))
			}
		}
		if final {
			// Process remaining blocks from COMPLETED event before breaking
			for _, block := range response.Blocks {
				if block.MarkdownBlock != nil && len(block.MarkdownBlock.Chunks) > 0 && (block.IntendedUsage == "ask_text" || block.IntendedUsage == "ask_text_0_markdown") {
					res_text := ""
					if inThinking {
						res_text += "``` \n\n"
						inThinking = false
					}
					for _, chunk := range block.MarkdownBlock.Chunks {
						if chunk != "" {
							res_text += chunk
							full_text += chunk
						}
					}
					if stream {
						model.ReturnOpenAIResponse(res_text, stream, gc)
					}
				}
			}
			break
		}
		for _, block := range response.Blocks {
			if block.ReasoningPlanBlock != nil && len(block.ReasoningPlanBlock.Goals) > 0 {
				res_text := ""
				if !inThinking && !thinkShown {
					res_text += "<think>"
					inThinking = true
				}
				for _, goal := range block.ReasoningPlanBlock.Goals {
					if goal.Description != "" && goal.Description != "Beginning analysis" && goal.Description != "Wrapping up analysis" {
						res_text += goal.Description
					}
				}
				full_text += res_text
				if stream {
					model.ReturnOpenAIResponse(res_text, stream, gc)
				}
			}
		}
		for _, block := range response.Blocks {
			if block.MarkdownBlock != nil && len(block.MarkdownBlock.Chunks) > 0 && block.IntendedUsage == "ask_text_0_markdown" {
				res_text := ""
				if inThinking {
					res_text += "</think>\n\n"
					inThinking = false
					thinkShown = true
				}
				for _, chunk := range block.MarkdownBlock.Chunks {
					if chunk != "" {
						res_text += chunk
					}
				}
				full_text += res_text
				if stream {
					model.ReturnOpenAIResponse(res_text, stream, gc)
				}
			}
		}
	}
	if !stream {
		model.ReturnOpenAIResponse(full_text, stream, gc)
	} else {
		gc.Writer.Write([]byte("data: [DONE]\n\n"))
		gc.Writer.Flush()
	}
	return nil
}

type UploadURLResponse struct {
	S3BucketURL string               `json:"s3_bucket_url"`
	S3ObjectURL string               `json:"s3_object_url"`
	Fields      CloudinaryUploadInfo `json:"fields"`
	RateLimited bool                 `json:"rate_limited"`
}

type CloudinaryUploadInfo struct {
	Timestamp         int    `json:"timestamp"`
	UniqueFilename    string `json:"unique_filename"`
	Folder            string `json:"folder"`
	UseFilename       string `json:"use_filename"`
	PublicID          string `json:"public_id"`
	Transformation    string `json:"transformation"`
	Moderation        string `json:"moderation"`
	ResourceType      string `json:"resource_type"`
	APIKey            string `json:"api_key"`
	CloudName         string `json:"cloud_name"`
	Signature         string `json:"signature"`
	AWSAccessKeyId    string `json:"AWSAccessKeyId"`
	Key               string `json:"key"`
	Tagging           string `json:"tagging"`
	Policy            string `json:"policy"`
	Xamzsecuritytoken string `json:"x-amz-security-token"`
	ACL               string `json:"acl"`
}

func (c *Client) createUploadURL(filename string, contentType string) (*UploadURLResponse, error) {
	requestBody := map[string]interface{}{
		"filename":     filename,
		"content_type": contentType,
		"source":       "default",
		"file_size":    12000,
		"force_image":  false,
	}
	resp, err := c.client.R().
		SetBody(requestBody).
		Post("https://www.perplexity.ai/rest/uploads/create_upload_url?version=" + pplxVersion + "&source=default")
	if err != nil {
		logger.Error(fmt.Sprintf("Error creating upload URL: %v", err))
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Upload URL failed with status %d", resp.StatusCode))
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var uploadURLResponse UploadURLResponse
	if err := json.Unmarshal(resp.Bytes(), &uploadURLResponse); err != nil {
		logger.Error(fmt.Sprintf("Error unmarshalling upload URL response: %v", err))
		return nil, err
	}
	if uploadURLResponse.RateLimited {
		logger.Error("Rate limit exceeded for upload URL")
		return nil, fmt.Errorf("rate limit exceeded")
	}
	return &uploadURLResponse, nil
}

func (c *Client) UploadImage(img_list []string) error {
	logger.Info(fmt.Sprintf("Uploading %d images", len(img_list)))
	for _, img := range img_list {
		filename := utils.RandomString(5) + ".jpg"
		uploadURLResponse, err := c.createUploadURL(filename, "image/jpeg")
		if err != nil {
			return err
		}
		if err := c.UloadFileToCloudinary(uploadURLResponse.Fields, "img", img, filename); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) UloadFileToCloudinary(uploadInfo CloudinaryUploadInfo, contentType string, filedata string, filename string) error {
	logger.Info(fmt.Sprintf("Uploading file %s to storage", filename))
	var formFields map[string]string
	if contentType == "img" {
		formFields = map[string]string{
			"signature":            uploadInfo.Signature,
			"key":                  uploadInfo.Key,
			"tagging":              uploadInfo.Tagging,
			"AWSAccessKeyId":       uploadInfo.AWSAccessKeyId,
			"policy":               uploadInfo.Policy,
			"x-amz-security-token": uploadInfo.Xamzsecuritytoken,
			"acl":                  uploadInfo.ACL,
			"Content-Type":         "image/jpeg",
		}
	} else {
		formFields = map[string]string{
			"acl":                  uploadInfo.ACL,
			"Content-Type":         "text/plain",
			"tagging":              uploadInfo.Tagging,
			"key":                  uploadInfo.Key,
			"AWSAccessKeyId":       uploadInfo.AWSAccessKeyId,
			"x-amz-security-token": uploadInfo.Xamzsecuritytoken,
			"policy":               uploadInfo.Policy,
			"signature":            uploadInfo.Signature,
		}
	}
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	for key, value := range formFields {
		if err := writer.WriteField(key, value); err != nil {
			return err
		}
	}
	decodedData, err := base64.StdEncoding.DecodeString(filedata)
	if err != nil {
		return err
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(decodedData); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	resp, err := c.client.R().
		SetHeader("Content-Type", writer.FormDataContentType()).
		SetBodyBytes(requestBody.Bytes()).
		Post("https://ppl-ai-file-upload.s3.amazonaws.com/")
	if err != nil {
		return fmt.Errorf("S3 upload request failed: %w", err)
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("S3 upload failed with status %d", resp.StatusCode)
	}
	logger.Info(fmt.Sprintf("File uploaded successfully: %s (status %d)", filename, resp.StatusCode))
	c.Attachments = append(c.Attachments, "https://ppl-ai-file-upload.s3.amazonaws.com/"+uploadInfo.Key)
	return nil
}

func (c *Client) UploadText(context string) error {
	logger.Info("Uploading text context to storage")
	filedata := base64.StdEncoding.EncodeToString([]byte(context))
	filename := utils.RandomString(5) + ".txt"
	uploadURLResponse, err := c.createUploadURL(filename, "text/plain")
	if err != nil {
		return err
	}
	return c.UloadFileToCloudinary(uploadURLResponse.Fields, "txt", filedata, filename)
}

func (c *Client) GetNewCookie() (string, error) {
	resp, err := c.client.R().Get("https://www.perplexity.ai/api/auth/session")
	if err != nil {
		logger.Error(fmt.Sprintf("Error getting session cookie: %v", err))
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "__Secure-next-auth.session-token" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("session cookie not found")
}
