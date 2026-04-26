package image

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/internal/middleware"
	"github.com/432539/gpt2api/pkg/resp"
)

// MeHandler 面向当前用户的图片任务只读接口(JWT 鉴权)。
// 与 /v1/images/tasks/:id(API Key 鉴权)共享同一张 image_tasks 表,
// 只是入口改到 /api/me/images/* 便于前端面板调用。
type MeHandler struct {
	dao        *DAO
	urlBuilder URLBuilder
}

// NewMeHandler 构造。
func NewMeHandler(dao *DAO) *MeHandler { return &MeHandler{dao: dao} }

// URLBuilder 用于把 task_id + idx 转成可访问的图片 URL。
// 由 main 注入,避免 image 包直接依赖 gateway 包造成循环依赖。
type URLBuilder func(taskID string, idx int) string

// SetURLBuilder 注入图片 URL 构造器(可选)。
func (h *MeHandler) SetURLBuilder(b URLBuilder) { h.urlBuilder = b }

// taskView 是对外返回的视图结构,解码 JSON 列 + 隐藏内部字段。
type taskView struct {
	ID             uint64     `json:"id"`
	TaskID         string     `json:"task_id"`
	UserID         uint64     `json:"user_id"`
	ModelID        uint64     `json:"model_id"`
	AccountID      uint64     `json:"account_id"`
	Prompt         string     `json:"prompt"`
	N              int        `json:"n"`
	Size           string     `json:"size"`
	Status         string     `json:"status"`
	ConversationID string     `json:"conversation_id,omitempty"`
	Error          string     `json:"error,omitempty"`
	CreditCost     int64      `json:"credit_cost"`
	ImageURLs      []string   `json:"image_urls"`
	FileIDs        []string   `json:"file_ids,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

func toView(t *Task, builder URLBuilder) taskView {
	urls := t.DecodeResultURLs()
	fids := t.DecodeFileIDs()
	for i, id := range fids {
		fids[i] = strings.TrimPrefix(id, "sed:")
	}
	if builder != nil {
		urlCount := len(fids)
		if urlCount == 0 {
			urlCount = len(urls)
		}
		if urlCount > 0 {
			rebuilt := make([]string, 0, urlCount)
			for i := 0; i < urlCount; i++ {
				rebuilt = append(rebuilt, builder(t.TaskID, i))
			}
			urls = rebuilt
		}
	}
	return taskView{
		ID: t.ID, TaskID: t.TaskID, UserID: t.UserID, ModelID: t.ModelID,
		AccountID: t.AccountID, Prompt: t.Prompt, N: t.N, Size: t.Size,
		Status: t.Status, ConversationID: t.ConversationID, Error: t.Error,
		CreditCost: t.CreditCost, ImageURLs: urls, FileIDs: fids,
		CreatedAt: t.CreatedAt, StartedAt: t.StartedAt, FinishedAt: t.FinishedAt,
	}
}

// GET /api/me/images/tasks
// 查询参数:limit(默认 20,上限 100), offset
func (h *MeHandler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		resp.Unauthorized(c, "not logged in")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	tasks, err := h.dao.ListByUser(c.Request.Context(), uid, limit, offset)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	items := make([]taskView, 0, len(tasks))
	for i := range tasks {
		items = append(items, toView(&tasks[i], h.urlBuilder))
	}
	resp.OK(c, gin.H{"items": items, "limit": limit, "offset": offset})
}

// GET /api/me/images/tasks/:id
func (h *MeHandler) Get(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		resp.Unauthorized(c, "not logged in")
		return
	}
	id := c.Param("id")
	if id == "" {
		resp.Fail(c, 40000, "task id required")
		return
	}
	t, err := h.dao.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.Fail(c, 40400, "task not found")
			return
		}
		resp.Internal(c, err.Error())
		return
	}
	if t.UserID != uid {
		resp.Fail(c, 40400, "task not found")
		return
	}
	resp.OK(c, toView(t, h.urlBuilder))
}
