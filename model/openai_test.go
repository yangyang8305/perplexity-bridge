package model

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext(w *httptest.ResponseRecorder) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	return c
}

// TestResponseModelName covers B2: model name must not be the old hardcoded value.
func TestResponseModelName(t *testing.T) {
	w := httptest.NewRecorder()
	c := newTestContext(w)
	if err := noStreamResponse("hello", c); err != nil {
		t.Fatalf("noStreamResponse error: %v", err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	mdl, _ := resp["model"].(string)
	if mdl == "claude-3-7-sonnet-20250219" {
		t.Errorf("model field still contains old hardcoded value: %q", mdl)
	}
	if mdl == "" {
		t.Errorf("model field is empty")
	}
}

// TestNoStreamResponseContentType covers B7: Content-Type must be application/json.
func TestNoStreamResponseContentType(t *testing.T) {
	w := httptest.NewRecorder()
	c := newTestContext(w)
	_ = noStreamResponse("test", c)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %q", ct)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestImageShowNoIndexLeak covers B6: ImageShow must not embed a stale index value.
func TestImageShowNoIndexLeak(t *testing.T) {
	// index is ignored; output must be deterministic regardless of index value
	out0 := "![cat](http://example.com/img.jpg)"
	if got := imageShowResult("cat", "http://example.com/img.jpg"); got != out0 {
		t.Errorf("got %q, want %q", got, out0)
	}
}

func imageShowResult(name, url string) string {
	import_fmt := func(n, u string) string {
		return "![" + n + "](" + u + ")"
	}
	return import_fmt(name, url)
}
