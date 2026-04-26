package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticBearerAuth 用实例级固定 Bearer Token 保护 /v1 图片网关。
func StaticBearerAuth(expected string) gin.HandlerFunc {
	expected = strings.TrimSpace(expected)
	return func(c *gin.Context) {
		hdr := c.GetHeader("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			openAIGatewayAuthError(c, "missing_api_key", "缺少 Bearer Token")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))
		if expected == "" || token != expected {
			openAIGatewayAuthError(c, "invalid_api_key", "Bearer Token 无效")
			return
		}
		c.Next()
	}
}

func openAIGatewayAuthError(c *gin.Context, code, msg string) {
	c.AbortWithStatusJSON(401, gin.H{
		"error": gin.H{
			"message": msg,
			"type":    "invalid_request_error",
			"code":    code,
		},
	})
}
