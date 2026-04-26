package accountpool

import "time"

// DispatchRoute 是图片请求进入调度器前的稳定路由契约。
// Phase 1 只表达主池 / fallback 池,后续可在此扩展 sticky/source 等维度。
type DispatchRoute struct {
	ModelID        uint64 `json:"model_id"`
	PrimaryPoolID  uint64 `json:"primary_pool_id"`
	FallbackPoolID uint64 `json:"fallback_pool_id"`
	AllowFallback  bool   `json:"allow_fallback"`
	LegacyGlobal   bool   `json:"legacy_global,omitempty"`
	Source         string `json:"source,omitempty"`
}

// Pool 对应 account_pools 表。
type Pool struct {
	ID               uint64     `db:"id" json:"id"`
	Code             string     `db:"code" json:"code"`
	Name             string     `db:"name" json:"name"`
	PoolType         string     `db:"pool_type" json:"pool_type"`
	Description      string     `db:"description" json:"description"`
	Enabled          bool       `db:"enabled" json:"enabled"`
	DispatchStrategy string     `db:"dispatch_strategy" json:"dispatch_strategy"`
	StickyTTLSec     int        `db:"sticky_ttl_sec" json:"sticky_ttl_sec"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt        *time.Time `db:"deleted_at" json:"-"`
}

// Member 对应 account_pool_members 表。
type Member struct {
	ID          uint64    `db:"id" json:"id"`
	PoolID      uint64    `db:"pool_id" json:"pool_id"`
	AccountID   uint64    `db:"account_id" json:"account_id"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	Weight      int       `db:"weight" json:"weight"`
	Priority    int       `db:"priority" json:"priority"`
	MaxParallel int       `db:"max_parallel" json:"max_parallel"`
	Note        string    `db:"note" json:"note"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ModelRoute 对应 model_pool_routes 表。
type ModelRoute struct {
	ID             uint64    `db:"id" json:"id"`
	ModelID        uint64    `db:"model_id" json:"model_id"`
	PoolID         uint64    `db:"pool_id" json:"pool_id"`
	FallbackPoolID uint64    `db:"fallback_pool_id" json:"fallback_pool_id"`
	Enabled        bool      `db:"enabled" json:"enabled"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// ResolvedRoute 是运行时解析后的模型路由结果。
type ResolvedRoute struct {
	PoolID         uint64 `json:"pool_id"`
	FallbackPoolID uint64 `json:"fallback_pool_id"`
	LegacyGlobal   bool   `json:"legacy_global"`
}

// ResolvedPool 是调度器侧看到的池快照。
// Phase 1 语义:
//   - priority: 数值越小优先级越高
//   - weight: 同优先级内按数值从大到小做稳定排序
//   - max_parallel: 当前全局仍是一号一锁,有效语义会被收敛到 >=1 的占位值
type ResolvedPool struct {
	Pool    *Pool     `json:"pool"`
	Members []*Member `json:"members"`
}
