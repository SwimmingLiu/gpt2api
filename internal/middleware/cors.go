package middleware

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 简易跨域中间件。
func CORS(origins []string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(origins))
	allowAll := false
	for _, o := range origins {
		if o == "*" {
			allowAll = true
		}
		allow[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAll {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else if _, ok := allow[strings.TrimRight(origin, "/")]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeaders(c.GetHeader("Access-Control-Request-Headers")))
			c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-Id")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func allowHeaders(requested string) string {
	set := map[string]string{
		"authorization":   "Authorization",
		"content-type":    "Content-Type",
		"x-request-id":    "X-Request-Id",
		"x-admin-confirm": "X-Admin-Confirm",
	}
	for _, part := range strings.Split(requested, ",") {
		h := strings.TrimSpace(part)
		if h == "" {
			continue
		}
		key := strings.ToLower(h)
		if _, ok := set[key]; !ok {
			set[key] = h
		}
	}
	keys := make([]string, 0, len(set))
	for _, v := range set {
		keys = append(keys, v)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
