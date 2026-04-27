package gateway

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	modelpkg "github.com/432539/gpt2api/internal/model"
)

// Handler 保留 image-only 主线仍需要的最小依赖。
type Handler struct {
	Models    *modelpkg.Registry
	Scheduler any
	Images    *ImagesHandler
	Settings  interface {
		GatewayUpstreamTimeoutSec() int
	}
}

// upstreamTimeout 返回当前应使用的上游非流式超时。未注入时回退 60s。
func (h *Handler) upstreamTimeout() time.Duration {
	if h != nil && h.Settings != nil {
		if n := h.Settings.GatewayUpstreamTimeoutSec(); n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 60 * time.Second
}

// openAIError 按 OpenAI 规范返回错误。
func openAIError(c *gin.Context, httpStatus int, code, msg string) {
	c.AbortWithStatusJSON(httpStatus, gin.H{
		"error": gin.H{
			"message": msg,
			"type":    "invalid_request_error",
			"code":    code,
		},
	})
}

// ListModels GET /v1/models
func (h *Handler) ListModels(c *gin.Context) {
	list, err := h.Models.ListEnabled(c.Request.Context())
	if err != nil {
		openAIError(c, http.StatusInternalServerError, "list_models_error", "获取模型列表失败:"+err.Error())
		return
	}
	data := make([]gin.H, 0, len(list))
	for _, m := range list {
		if m.Slug != "gpt-image-2" {
			continue
		}
		data = append(data, gin.H{
			"id":       m.Slug,
			"object":   "model",
			"created":  m.CreatedAt.Unix(),
			"owned_by": "chatgpt",
		})
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
}
