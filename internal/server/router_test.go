package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/432539/gpt2api/internal/config"
	pkgjwt "github.com/432539/gpt2api/pkg/jwt"
)

func TestRouterDoesNotExposeLegacyGatewayRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := New(&Deps{
		Config: &config.Config{
			App:      config.AppConfig{Env: "test"},
			Security: config.SecurityConfig{},
		},
		JWT:    testJWTManager(),
	})

	for _, path := range []string{
		"/v1/chat/completions",
		"/v1/images/edits",
		"/v1/images/tasks/test",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if path == "/v1/images/tasks/test" {
			req = httptest.NewRequest(http.MethodGet, path, nil)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected %s to be 404, got %d", path, w.Code)
		}
	}
}

func testJWTManager() *pkgjwt.Manager {
	return pkgjwt.NewManager(pkgjwt.Config{
		Secret:        "test-secret",
		Issuer:        "test",
		AccessTTLSec:  3600,
		RefreshTTLSec: 86400,
	})
}
