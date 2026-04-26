package accountsource

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/pkg/resp"
)

type handlerService interface {
	List(ctx *gin.Context) ([]*SourceView, error)
	Create(ctx *gin.Context, in CreateInput) (*SourceView, error)
	Get(ctx *gin.Context, id uint64) (*SourceView, error)
	Update(ctx *gin.Context, id uint64, in UpdateInput) (*SourceView, error)
	Delete(ctx *gin.Context, id uint64) error
	ListSub2APIGroups(ctx *gin.Context, id uint64) ([]*Sub2APIGroup, error)
	ListSub2APIAccounts(ctx *gin.Context, id uint64) ([]*Sub2APIAccount, error)
	ListCPAFiles(ctx *gin.Context, id uint64) ([]*CPAFile, error)
	ImportSelected(ctx *gin.Context, id uint64, in ImportSelectedInput) (*ImportSummary, error)
}

type serviceAdapter struct{ svc *Service }

func (s serviceAdapter) List(c *gin.Context) ([]*SourceView, error) {
	return s.svc.List(c.Request.Context())
}
func (s serviceAdapter) Create(c *gin.Context, in CreateInput) (*SourceView, error) {
	return s.svc.Create(c.Request.Context(), in)
}
func (s serviceAdapter) Get(c *gin.Context, id uint64) (*SourceView, error) {
	return s.svc.Get(c.Request.Context(), id)
}
func (s serviceAdapter) Update(c *gin.Context, id uint64, in UpdateInput) (*SourceView, error) {
	return s.svc.Update(c.Request.Context(), id, in)
}
func (s serviceAdapter) Delete(c *gin.Context, id uint64) error {
	return s.svc.Delete(c.Request.Context(), id)
}
func (s serviceAdapter) ListSub2APIGroups(c *gin.Context, id uint64) ([]*Sub2APIGroup, error) {
	return s.svc.ListSub2APIGroups(c.Request.Context(), id)
}
func (s serviceAdapter) ListSub2APIAccounts(c *gin.Context, id uint64) ([]*Sub2APIAccount, error) {
	return s.svc.ListSub2APIAccounts(c.Request.Context(), id)
}
func (s serviceAdapter) ListCPAFiles(c *gin.Context, id uint64) ([]*CPAFile, error) {
	return s.svc.ListCPAFiles(c.Request.Context(), id)
}
func (s serviceAdapter) ImportSelected(c *gin.Context, id uint64, in ImportSelectedInput) (*ImportSummary, error) {
	return s.svc.ImportSelected(c.Request.Context(), id, in)
}

type Handler struct {
	svc handlerService
}

func NewHandler(svc handlerService) *Handler {
	return &Handler{svc: svc}
}

func NewHTTPHandler(svc *Service) *Handler {
	return NewHandler(serviceAdapter{svc: svc})
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.svc.List(c)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.Create(c, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, item)
}

func (h *Handler) Get(c *gin.Context) {
	id := parseUintParam(c, "id")
	item, err := h.svc.Get(c, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	id := parseUintParam(c, "id")
	var req UpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.Update(c, id, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	id := parseUintParam(c, "id")
	if err := h.svc.Delete(c, id); err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, gin.H{"deleted": id})
}

func (h *Handler) ListSub2APIGroups(c *gin.Context) {
	id := parseUintParam(c, "id")
	items, err := h.svc.ListSub2APIGroups(c, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListSub2APIAccounts(c *gin.Context) {
	id := parseUintParam(c, "id")
	items, err := h.svc.ListSub2APIAccounts(c, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListCPAFiles(c *gin.Context) {
	id := parseUintParam(c, "id")
	items, err := h.svc.ListCPAFiles(c, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ImportSelected(c *gin.Context) {
	id := parseUintParam(c, "id")
	var req ImportSelectedInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.ImportSelected(c, id, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	resp.OK(c, item)
}

func parseUintParam(c *gin.Context, key string) uint64 {
	id, _ := strconv.ParseUint(c.Param(key), 10, 64)
	return id
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		resp.NotFound(c, err.Error())
	case errors.Is(err, ErrBadRequest):
		resp.BadRequest(c, err.Error())
	default:
		resp.Internal(c, err.Error())
	}
}
