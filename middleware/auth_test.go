package middleware

import (
	"net/http"
	"net/http/httptest"
	"pplx2api/config"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(apiKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	config.ConfigInstance = &config.Config{
		APIKey:  apiKey,
		RwMutex: sync.RWMutex{},
	}
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) { c.Status(200) })
	return r
}

// A1: empty APIKEY must allow all requests through
func TestAuthNoKeyRequired(t *testing.T) {
	r := setupRouter("")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 with no APIKEY set, got %d", w.Code)
	}
}

// A1: empty APIKEY, even with a Bearer token, must still allow through
func TestAuthNoKeyRequiredWithToken(t *testing.T) {
	r := setupRouter("")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 with no APIKEY set, got %d", w.Code)
	}
}

// A4: correct key must pass
func TestAuthCorrectKey(t *testing.T) {
	r := setupRouter("secret")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer secret")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 with correct key, got %d", w.Code)
	}
}

// Wrong key must be rejected
func TestAuthWrongKey(t *testing.T) {
	r := setupRouter("secret")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401 with wrong key, got %d", w.Code)
	}
}

// Missing header must be rejected when APIKEY is set
func TestAuthMissingHeader(t *testing.T) {
	r := setupRouter("secret")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401 with missing header, got %d", w.Code)
	}
}
