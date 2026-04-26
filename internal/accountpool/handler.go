package accountpool

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/pkg/resp"
)

type handlerService interface {
	ListPools(ctx *gin.Context) ([]*Pool, error)
	GetPool(ctx *gin.Context, id uint64) (*Pool, error)
	CreatePool(ctx *gin.Context, in CreatePoolInput) (*Pool, error)
	UpdatePool(ctx *gin.Context, id uint64, in UpdatePoolInput) (*Pool, error)
	DeletePool(ctx *gin.Context, id uint64) error
	ListMembers(ctx *gin.Context, poolID uint64) ([]*Member, error)
	UpsertMember(ctx *gin.Context, poolID uint64, memberID uint64, in UpsertMemberInput) (*Member, error)
	DeleteMember(ctx *gin.Context, poolID, memberID uint64) error
	ListRoutes(ctx *gin.Context) ([]*ModelRoute, error)
	PutRoute(ctx *gin.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error)
	DeleteRoute(ctx *gin.Context, modelID uint64) error
}

type serviceAdapter struct{ svc *Service }

func (s serviceAdapter) ListPools(c *gin.Context) ([]*Pool, error) {
	return s.svc.ListPools(c.Request.Context())
}
func (s serviceAdapter) GetPool(c *gin.Context, id uint64) (*Pool, error) {
	return s.svc.GetPool(c.Request.Context(), id)
}
func (s serviceAdapter) CreatePool(c *gin.Context, in CreatePoolInput) (*Pool, error) {
	return s.svc.CreatePool(c.Request.Context(), in)
}
func (s serviceAdapter) UpdatePool(c *gin.Context, id uint64, in UpdatePoolInput) (*Pool, error) {
	return s.svc.UpdatePool(c.Request.Context(), id, in)
}
func (s serviceAdapter) DeletePool(c *gin.Context, id uint64) error {
	return s.svc.DeletePool(c.Request.Context(), id)
}
func (s serviceAdapter) ListMembers(c *gin.Context, poolID uint64) ([]*Member, error) {
	return s.svc.ListMembers(c.Request.Context(), poolID)
}
func (s serviceAdapter) UpsertMember(c *gin.Context, poolID uint64, memberID uint64, in UpsertMemberInput) (*Member, error) {
	return s.svc.UpsertMember(c.Request.Context(), poolID, memberID, in)
}
func (s serviceAdapter) DeleteMember(c *gin.Context, poolID, memberID uint64) error {
	return s.svc.DeleteMember(c.Request.Context(), poolID, memberID)
}
func (s serviceAdapter) ListRoutes(c *gin.Context) ([]*ModelRoute, error) {
	return s.svc.ListRoutes(c.Request.Context())
}
func (s serviceAdapter) PutRoute(c *gin.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error) {
	return s.svc.PutRoute(c.Request.Context(), modelID, in)
}
func (s serviceAdapter) DeleteRoute(c *gin.Context, modelID uint64) error {
	return s.svc.DeleteRoute(c.Request.Context(), modelID)
}

type Handler struct {
	svc interface {
		ListPools(ctx *gin.Context) ([]*Pool, error)
		GetPool(ctx *gin.Context, id uint64) (*Pool, error)
		CreatePool(ctx *gin.Context, in CreatePoolInput) (*Pool, error)
		UpdatePool(ctx *gin.Context, id uint64, in UpdatePoolInput) (*Pool, error)
		DeletePool(ctx *gin.Context, id uint64) error
		ListMembers(ctx *gin.Context, poolID uint64) ([]*Member, error)
		UpsertMember(ctx *gin.Context, poolID uint64, memberID uint64, in UpsertMemberInput) (*Member, error)
		DeleteMember(ctx *gin.Context, poolID, memberID uint64) error
		ListRoutes(ctx *gin.Context) ([]*ModelRoute, error)
		PutRoute(ctx *gin.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error)
		DeleteRoute(ctx *gin.Context, modelID uint64) error
	}
}

func NewHandler(svc interface {
	ListPools(ctx *gin.Context) ([]*Pool, error)
	GetPool(ctx *gin.Context, id uint64) (*Pool, error)
	CreatePool(ctx *gin.Context, in CreatePoolInput) (*Pool, error)
	UpdatePool(ctx *gin.Context, id uint64, in UpdatePoolInput) (*Pool, error)
	DeletePool(ctx *gin.Context, id uint64) error
	ListMembers(ctx *gin.Context, poolID uint64) ([]*Member, error)
	UpsertMember(ctx *gin.Context, poolID uint64, memberID uint64, in UpsertMemberInput) (*Member, error)
	DeleteMember(ctx *gin.Context, poolID, memberID uint64) error
	ListRoutes(ctx *gin.Context) ([]*ModelRoute, error)
	PutRoute(ctx *gin.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error)
	DeleteRoute(ctx *gin.Context, modelID uint64) error
}) *Handler {
	return &Handler{svc: svc}
}

func NewHTTPHandler(svc *Service) *Handler {
	return NewHandler(serviceAdapter{svc: svc})
}

func (h *Handler) ListPools(c *gin.Context) {
	items, err := h.svc.ListPools(c)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) GetPool(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	item, err := h.svc.GetPool(c, id)
	if err != nil {
		resp.NotFound(c, err.Error())
		return
	}
	resp.OK(c, item)
}

func (h *Handler) CreatePool(c *gin.Context) {
	var req CreatePoolInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.CreatePool(c, req)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, item)
}

func (h *Handler) UpdatePool(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req UpdatePoolInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.UpdatePool(c, id, req)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, item)
}

func (h *Handler) DeletePool(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.DeletePool(c, id); err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"deleted": id})
}

func (h *Handler) ListMembers(c *gin.Context) {
	poolID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	items, err := h.svc.ListMembers(c, poolID)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) UpsertMember(c *gin.Context) {
	poolID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	memberID, _ := strconv.ParseUint(c.Param("memberId"), 10, 64)
	var req UpsertMemberInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.UpsertMember(c, poolID, memberID, req)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, item)
}

func (h *Handler) DeleteMember(c *gin.Context) {
	poolID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	memberID, _ := strconv.ParseUint(c.Param("memberId"), 10, 64)
	if err := h.svc.DeleteMember(c, poolID, memberID); err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"deleted": memberID, "pool_id": poolID})
}

func (h *Handler) ListRoutes(c *gin.Context) {
	items, err := h.svc.ListRoutes(c)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) PutRoute(c *gin.Context) {
	modelID, _ := strconv.ParseUint(c.Param("modelId"), 10, 64)
	var req PutRouteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.PutRoute(c, modelID, req)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, item)
}

func (h *Handler) DeleteRoute(c *gin.Context) {
	modelID, _ := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err := h.svc.DeleteRoute(c, modelID); err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"deleted": modelID})
}
