package accountpool

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

var poolCodeRe = regexp.MustCompile(`^[a-z][a-z0-9\-]{1,63}$`)

var (
	ErrRouteNotConfigured = errors.New("accountpool: dispatch route not configured")
	ErrPoolDisabled       = errors.New("accountpool: pool disabled")
)

type routeReader interface {
	GetRouteByModelID(ctx context.Context, modelID uint64) (*ModelRoute, error)
}

type store interface {
	routeReader
	ListPools(ctx context.Context) ([]*Pool, error)
	GetPoolByID(ctx context.Context, id uint64) (*Pool, error)
	CreatePool(ctx context.Context, p *Pool) error
	UpdatePool(ctx context.Context, p *Pool) error
	SoftDeletePool(ctx context.Context, id uint64) error
	ListMembers(ctx context.Context, poolID uint64) ([]*Member, error)
	UpsertMember(ctx context.Context, in *Member) error
	DeleteMember(ctx context.Context, poolID, memberID uint64) error
	ListRoutes(ctx context.Context) ([]*ModelRoute, error)
	UpsertRoute(ctx context.Context, in *ModelRoute) error
	DeleteRoute(ctx context.Context, modelID uint64) error
}

// Service 提供账号池相关基础能力。
type Service struct {
	store store
}

func NewService(store store) *Service {
	return &Service{store: store}
}

func validatePoolCode(code string) error {
	if !poolCodeRe.MatchString(code) {
		return errors.New("invalid pool code")
	}
	return nil
}

// ResolveModelRoute 解析模型的账号池路由。
// 未配置时返回 legacy global 兼容模式。
func (s *Service) ResolveModelRoute(ctx context.Context, modelID uint64) (ResolvedRoute, error) {
	if s == nil || s.store == nil {
		return ResolvedRoute{LegacyGlobal: true}, nil
	}
	route, err := s.store.GetRouteByModelID(ctx, modelID)
	if err != nil {
		return ResolvedRoute{}, err
	}
	if route == nil || !route.Enabled {
		return ResolvedRoute{LegacyGlobal: true}, nil
	}
	return ResolvedRoute{
		PoolID:         route.PoolID,
		FallbackPoolID: route.FallbackPoolID,
		LegacyGlobal:   false,
	}, nil
}

// ResolveDispatchRoute 把模型路由与默认池配置折叠成运行时稳定契约。
func (s *Service) ResolveDispatchRoute(
	ctx context.Context,
	modelID uint64,
	defaultPoolID uint64,
	defaultFallbackPoolID uint64,
) (DispatchRoute, error) {
	route, err := s.ResolveModelRoute(ctx, modelID)
	if err != nil {
		return DispatchRoute{}, err
	}
	if !route.LegacyGlobal && route.PoolID > 0 {
		return DispatchRoute{
			ModelID:        modelID,
			PrimaryPoolID:  route.PoolID,
			FallbackPoolID: route.FallbackPoolID,
			AllowFallback:  route.FallbackPoolID > 0,
			Source:         "model_route",
		}, nil
	}
	if defaultPoolID > 0 {
		return DispatchRoute{
			ModelID:        modelID,
			PrimaryPoolID:  defaultPoolID,
			FallbackPoolID: defaultFallbackPoolID,
			AllowFallback:  defaultFallbackPoolID > 0,
			Source:         "default_pool",
		}, nil
	}
	return DispatchRoute{}, ErrRouteNotConfigured
}

// ResolvePool 返回池开关 + 成员列表组成的运行时快照。
func (s *Service) ResolvePool(ctx context.Context, poolID uint64) (ResolvedPool, error) {
	if poolID == 0 {
		return ResolvedPool{}, ErrNotFound
	}
	pool, err := s.store.GetPoolByID(ctx, poolID)
	if err != nil {
		return ResolvedPool{}, err
	}
	if pool == nil {
		return ResolvedPool{}, ErrNotFound
	}
	if !pool.Enabled {
		return ResolvedPool{}, ErrPoolDisabled
	}
	members, err := s.store.ListMembers(ctx, poolID)
	if err != nil {
		return ResolvedPool{}, err
	}
	return ResolvedPool{
		Pool:    pool,
		Members: members,
	}, nil
}

type CreatePoolInput struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	PoolType         string `json:"pool_type"`
	Description      string `json:"description"`
	Enabled          *bool  `json:"enabled,omitempty"`
	DispatchStrategy string `json:"dispatch_strategy"`
	StickyTTLSec     int    `json:"sticky_ttl_sec"`
}

type UpdatePoolInput struct {
	Name             string `json:"name"`
	PoolType         string `json:"pool_type"`
	Description      string `json:"description"`
	Enabled          *bool  `json:"enabled,omitempty"`
	DispatchStrategy string `json:"dispatch_strategy"`
	StickyTTLSec     int    `json:"sticky_ttl_sec"`
}

type UpsertMemberInput struct {
	AccountID   uint64 `json:"account_id"`
	Enabled     *bool  `json:"enabled,omitempty"`
	Weight      int    `json:"weight"`
	Priority    int    `json:"priority"`
	MaxParallel int    `json:"max_parallel"`
	Note        string `json:"note"`
}

type PutRouteInput struct {
	PoolID         uint64 `json:"pool_id"`
	FallbackPoolID uint64 `json:"fallback_pool_id"`
	Enabled        *bool  `json:"enabled,omitempty"`
}

func (s *Service) ListPools(ctx context.Context) ([]*Pool, error) {
	return s.store.ListPools(ctx)
}

func (s *Service) GetPool(ctx context.Context, id uint64) (*Pool, error) {
	return s.store.GetPoolByID(ctx, id)
}

func (s *Service) CreatePool(ctx context.Context, in CreatePoolInput) (*Pool, error) {
	in.Code = strings.TrimSpace(strings.ToLower(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.PoolType = strings.TrimSpace(strings.ToLower(in.PoolType))
	in.Description = strings.TrimSpace(in.Description)
	in.DispatchStrategy = strings.TrimSpace(strings.ToLower(in.DispatchStrategy))

	if err := validatePoolCode(in.Code); err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, errors.New("name 不能为空")
	}
	if in.PoolType == "" {
		in.PoolType = "mixed"
	}
	if !isValidPoolType(in.PoolType) {
		return nil, errors.New("pool_type 仅支持 chat / image / codex / mixed")
	}
	if in.DispatchStrategy == "" {
		in.DispatchStrategy = "least_recently_used"
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	p := &Pool{
		Code:             in.Code,
		Name:             in.Name,
		PoolType:         in.PoolType,
		Description:      in.Description,
		Enabled:          enabled,
		DispatchStrategy: in.DispatchStrategy,
		StickyTTLSec:     in.StickyTTLSec,
	}
	if err := s.store.CreatePool(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) UpdatePool(ctx context.Context, id uint64, in UpdatePoolInput) (*Pool, error) {
	cur, err := s.store.GetPoolByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		cur.Name = name
	}
	if poolType := strings.TrimSpace(strings.ToLower(in.PoolType)); poolType != "" {
		if !isValidPoolType(poolType) {
			return nil, errors.New("pool_type 仅支持 chat / image / codex / mixed")
		}
		cur.PoolType = poolType
	}
	if ds := strings.TrimSpace(strings.ToLower(in.DispatchStrategy)); ds != "" {
		cur.DispatchStrategy = ds
	}
	cur.Description = strings.TrimSpace(in.Description)
	cur.StickyTTLSec = in.StickyTTLSec
	if in.Enabled != nil {
		cur.Enabled = *in.Enabled
	}
	if err := s.store.UpdatePool(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *Service) DeletePool(ctx context.Context, id uint64) error {
	return s.store.SoftDeletePool(ctx, id)
}

func (s *Service) ListMembers(ctx context.Context, poolID uint64) ([]*Member, error) {
	return s.store.ListMembers(ctx, poolID)
}

func (s *Service) UpsertMember(ctx context.Context, poolID uint64, memberID uint64, in UpsertMemberInput) (*Member, error) {
	if poolID == 0 {
		return nil, errors.New("pool_id 不能为空")
	}
	if memberID == 0 && in.AccountID == 0 {
		return nil, errors.New("account_id 不能为空")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	if in.Weight <= 0 {
		in.Weight = 100
	}
	if in.Priority <= 0 {
		in.Priority = 100
	}
	if in.MaxParallel <= 0 {
		in.MaxParallel = 1
	}
	m := &Member{
		ID:          memberID,
		PoolID:      poolID,
		AccountID:   in.AccountID,
		Enabled:     enabled,
		Weight:      in.Weight,
		Priority:    in.Priority,
		MaxParallel: in.MaxParallel,
		Note:        strings.TrimSpace(in.Note),
	}
	if err := s.store.UpsertMember(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) DeleteMember(ctx context.Context, poolID, memberID uint64) error {
	return s.store.DeleteMember(ctx, poolID, memberID)
}

func (s *Service) ListRoutes(ctx context.Context) ([]*ModelRoute, error) {
	return s.store.ListRoutes(ctx)
}

func (s *Service) PutRoute(ctx context.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error) {
	if modelID == 0 {
		return nil, errors.New("model_id 非法")
	}
	if in.PoolID == 0 {
		return nil, errors.New("pool_id 不能为空")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	route := &ModelRoute{
		ModelID:        modelID,
		PoolID:         in.PoolID,
		FallbackPoolID: in.FallbackPoolID,
		Enabled:        enabled,
	}
	if err := s.store.UpsertRoute(ctx, route); err != nil {
		return nil, err
	}
	return route, nil
}

func (s *Service) DeleteRoute(ctx context.Context, modelID uint64) error {
	return s.store.DeleteRoute(ctx, modelID)
}
