package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsRequestedHeadersOnPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS([]string{"http://localhost:5173"}))
	r.OPTIONS("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-admin-confirm")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	got := w.Header().Get("Access-Control-Allow-Headers")
	for _, want := range []string{"Authorization", "Content-Type", "X-Admin-Confirm"} {
		if !strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
			t.Fatalf("expected Access-Control-Allow-Headers to contain %q, got %q", want, got)
		}
	}
}
