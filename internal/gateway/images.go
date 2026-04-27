package gateway

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/internal/accountpool"
	"github.com/432539/gpt2api/internal/image"
	modelpkg "github.com/432539/gpt2api/internal/model"
)

const maxReferenceImageBytes = 20 * 1024 * 1024
const maxReferenceImages = 4

type ImagesHandler struct {
	*Handler
	Runner        *image.Runner
	DAO           *image.DAO
	RouteResolver interface {
		ResolveDispatchRoute(
			ctx context.Context,
			modelID uint64,
			defaultPoolID uint64,
			defaultFallbackPoolID uint64,
		) (accountpool.DispatchRoute, error)
	}
	DefaultPoolID         uint64
	DefaultFallbackPoolID uint64
	ImageAccResolver      ImageAccountResolver
}

type ImageGenRequest struct {
	Model           string   `json:"model"`
	Prompt          string   `json:"prompt"`
	N               int      `json:"n"`
	Size            string   `json:"size"`
	Quality         string   `json:"quality,omitempty"`
	Style           string   `json:"style,omitempty"`
	ResponseFormat  string   `json:"response_format,omitempty"`
	User            string   `json:"user,omitempty"`
	ReferenceImages []string `json:"reference_images,omitempty"`
}

type ImageGenData struct {
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	FileID        string `json:"file_id,omitempty"`
}

type ImageGenResponse struct {
	Created   int64          `json:"created"`
	Data      []ImageGenData `json:"data"`
	TaskID    string         `json:"task_id,omitempty"`
	IsPreview bool           `json:"is_preview,omitempty"`
}

// ImageGenerations POST /v1/images/generations
func (h *ImagesHandler) ImageGenerations(c *gin.Context) {
	var req ImageGenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		openAIError(c, http.StatusBadRequest, "invalid_request_error", "请求参数错误:"+err.Error())
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		openAIError(c, http.StatusBadRequest, "invalid_request_error", "prompt 不能为空")
		return
	}
	if req.Model == "" {
		req.Model = "gpt-image-2"
	}
	if req.N <= 0 {
		req.N = 1
	}
	if req.N > 4 {
		req.N = 4
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}

	if req.Model != "gpt-image-2" {
		openAIError(c, http.StatusBadRequest, "model_not_found",
			fmt.Sprintf("当前实例仅支持模型 %q", "gpt-image-2"))
		return
	}
	m, err := h.Models.BySlug(c.Request.Context(), req.Model)
	if err != nil || m == nil || !m.Enabled {
		openAIError(c, http.StatusBadRequest, "model_not_found",
			fmt.Sprintf("模型 %q 不存在或已下架", req.Model))
		return
	}
	if m.Type != modelpkg.TypeImage {
		openAIError(c, http.StatusBadRequest, "model_type_mismatch",
			fmt.Sprintf("模型 %q 不是图像模型,不能用于 /v1/images/generations", req.Model))
		return
	}

	if h.RouteResolver == nil {
		openAIError(c, http.StatusServiceUnavailable, "pool_route_not_configured", "图片池路由未初始化")
		return
	}
	route, err := h.RouteResolver.ResolveDispatchRoute(
		c.Request.Context(),
		m.ID,
		h.DefaultPoolID,
		h.DefaultFallbackPoolID,
	)
	if err != nil {
		openAIError(c, http.StatusServiceUnavailable, "pool_route_not_configured", "未配置可用的图片账号池路由")
		return
	}

	taskID := image.GenerateTaskID()
	task := &image.Task{
		TaskID:          taskID,
		UserID:          0,
		KeyID:           0,
		ModelID:         m.ID,
		Prompt:          req.Prompt,
		N:               req.N,
		Size:            req.Size,
		Status:          image.StatusDispatched,
		EstimatedCredit: 0,
	}
	if h.DAO != nil {
		if err := h.DAO.Create(c.Request.Context(), task); err != nil {
			openAIError(c, http.StatusInternalServerError, "internal_error", "创建任务失败:"+err.Error())
			return
		}
	}

	refs, err := decodeReferenceInputs(c.Request.Context(), req.ReferenceImages)
	if err != nil {
		openAIError(c, http.StatusBadRequest, "invalid_reference_image", "参考图解析失败:"+err.Error())
		return
	}

	runCtx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Minute)
	defer cancel()

	maxAttempts := 2
	if len(refs) > 0 {
		maxAttempts = 1
	}

	res := h.Runner.Run(runCtx, image.RunOptions{
		TaskID:        taskID,
		UserID:        0,
		KeyID:         0,
		ModelID:       m.ID,
		DispatchRoute: route,
		UpstreamModel: m.UpstreamModelSlug,
		Prompt:        maybeAppendClaritySuffix(req.Prompt),
		N:             req.N,
		MaxAttempts:   maxAttempts,
		References:    refs,
	})

	if res.Status != image.StatusSuccess {
		httpStatus := http.StatusBadGateway
		if res.ErrorCode == image.ErrNoAccount || res.ErrorCode == image.ErrRateLimited {
			httpStatus = http.StatusServiceUnavailable
		}
		openAIError(c, httpStatus, ifEmpty(res.ErrorCode, "upstream_error"),
			localizeImageErr(res.ErrorCode, res.ErrorMessage))
		return
	}

	if h.DAO != nil {
		_ = h.DAO.UpdateCost(c.Request.Context(), taskID, 0)
	}

	out := ImageGenResponse{
		Created:   time.Now().Unix(),
		TaskID:    taskID,
		IsPreview: res.IsPreview,
		Data:      make([]ImageGenData, 0, len(res.SignedURLs)),
	}
	for i := range res.SignedURLs {
		d := ImageGenData{URL: BuildImageProxyURL(taskID, i, ImageProxyTTL)}
		if i < len(res.FileIDs) {
			d.FileID = strings.TrimPrefix(res.FileIDs[i], "sed:")
		}
		out.Data = append(out.Data, d)
	}
	c.JSON(http.StatusOK, out)
}

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func localizeImageErr(code, raw string) string {
	var zh string
	switch code {
	case image.ErrNoAccount:
		zh = "账号池暂无可用账号,请稍后重试"
	case image.ErrRateLimited:
		zh = "上游风控,请稍后再试"
	case image.ErrPreviewOnly:
		zh = "上游仅返回预览,请稍后重试(已尝试切换账号)"
	case image.ErrUnknown, "":
		zh = "图片生成失败"
	case "upstream_error":
		zh = "上游返回错误"
	default:
		zh = "图片生成失败(" + code + ")"
	}
	if raw != "" && raw != code {
		return zh + ":" + raw
	}
	return zh
}

var textHintKeywords = []string{
	"文字", "对话", "台词", "旁白", "标语", "字幕", "标题", "文案",
	"招牌", "横幅", "海报文字", "弹幕", "气泡", "字体",
	"text:", "caption", "subtitle", "title:", "label", "banner", "poster text",
}

const claritySuffix = "\n\nclean readable Chinese text, prioritize text clarity over image details"

func decodeReferenceInputs(ctx context.Context, inputs []string) ([]image.ReferenceImage, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if len(inputs) > maxReferenceImages {
		return nil, fmt.Errorf("最多支持 %d 张参考图", maxReferenceImages)
	}
	out := make([]image.ReferenceImage, 0, len(inputs))
	for i, s := range inputs {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("第 %d 张参考图为空", i+1)
		}
		data, name, err := fetchReferenceBytes(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("第 %d 张参考图:%w", i+1, err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("第 %d 张参考图解码后为空", i+1)
		}
		if len(data) > maxReferenceImageBytes {
			return nil, fmt.Errorf("第 %d 张参考图超过 %dMB 上限", i+1, maxReferenceImageBytes/1024/1024)
		}
		out = append(out, image.ReferenceImage{Data: data, FileName: name})
	}
	return out, nil
}

func fetchReferenceBytes(ctx context.Context, s string) ([]byte, string, error) {
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(low, "data:"):
		comma := strings.IndexByte(s, ',')
		if comma < 0 {
			return nil, "", errors.New("无效 data URL")
		}
		meta := s[5:comma]
		payload := s[comma+1:]
		if strings.Contains(strings.ToLower(meta), ";base64") {
			b, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				if b2, err2 := base64.URLEncoding.DecodeString(payload); err2 == nil {
					b = b2
				} else {
					return nil, "", fmt.Errorf("base64 解码失败:%w", err)
				}
			}
			return b, "", nil
		}
		return []byte(payload), "", nil
	case strings.HasPrefix(low, "http://"), strings.HasPrefix(low, "https://"):
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s, nil)
		if err != nil {
			return nil, "", err
		}
		hc := &http.Client{Timeout: 15 * time.Second}
		res, err := hc.Do(req)
		if err != nil {
			return nil, "", err
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			return nil, "", fmt.Errorf("下载失败 HTTP %d", res.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, int64(maxReferenceImageBytes)+1))
		if err != nil {
			return nil, "", err
		}
		name := filepath.Base(req.URL.Path)
		return body, name, nil
	default:
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			if b2, err2 := base64.URLEncoding.DecodeString(s); err2 == nil {
				return b2, "", nil
			}
			return nil, "", fmt.Errorf("既非 URL 也非可解析的 base64:%w", err)
		}
		return b, "", nil
	}
}

func maybeAppendClaritySuffix(prompt string) string {
	lower := strings.ToLower(prompt)
	need := false
	for _, kw := range textHintKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			need = true
			break
		}
	}
	if !need {
		for _, pair := range [][2]string{
			{"\"", "\""}, {"'", "'"},
			{"“", "”"}, {"‘", "’"},
			{"「", "」"}, {"『", "』"},
		} {
			if idx := strings.Index(prompt, pair[0]); idx >= 0 {
				rest := prompt[idx+len(pair[0]):]
				if end := strings.Index(rest, pair[1]); end >= 2 {
					need = true
					break
				}
			}
		}
	}
	if need && !strings.Contains(prompt, strings.TrimSpace(claritySuffix)) {
		return prompt + claritySuffix
	}
	return prompt
}
