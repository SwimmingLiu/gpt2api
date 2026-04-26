package accountpool

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("accountpool: not found")

type DAO struct{ db *sqlx.DB }

func NewDAO(db *sqlx.DB) *DAO { return &DAO{db: db} }

func (d *DAO) ListPools(ctx context.Context) ([]*Pool, error) {
	out := make([]*Pool, 0, 16)
	err := d.db.SelectContext(ctx, &out,
		`SELECT * FROM account_pools WHERE deleted_at IS NULL ORDER BY id DESC`)
	return out, err
}

func (d *DAO) GetPoolByID(ctx context.Context, id uint64) (*Pool, error) {
	var out Pool
	err := d.db.GetContext(ctx, &out,
		`SELECT * FROM account_pools WHERE id = ? AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &out, err
}

func (d *DAO) CreatePool(ctx context.Context, p *Pool) error {
	res, err := d.db.ExecContext(ctx, `
INSERT INTO account_pools
  (code, name, pool_type, description, enabled, dispatch_strategy, sticky_ttl_sec)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.Code, p.Name, p.PoolType, p.Description, p.Enabled, p.DispatchStrategy, p.StickyTTLSec)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	p.ID = uint64(id)
	return nil
}

func (d *DAO) UpdatePool(ctx context.Context, p *Pool) error {
	res, err := d.db.ExecContext(ctx, `
UPDATE account_pools
   SET name = ?, pool_type = ?, description = ?, enabled = ?, dispatch_strategy = ?, sticky_ttl_sec = ?
 WHERE id = ? AND deleted_at IS NULL`,
		p.Name, p.PoolType, p.Description, p.Enabled, p.DispatchStrategy, p.StickyTTLSec, p.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DAO) SoftDeletePool(ctx context.Context, id uint64) error {
	res, err := d.db.ExecContext(ctx, `
UPDATE account_pools
   SET deleted_at = NOW(),
       enabled = 0,
       code = CONCAT(code, '#del', id)
 WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DAO) ListMembers(ctx context.Context, poolID uint64) ([]*Member, error) {
	out := make([]*Member, 0, 16)
	err := d.db.SelectContext(ctx, &out,
		`SELECT * FROM account_pool_members WHERE pool_id = ? ORDER BY id DESC`, poolID)
	return out, err
}

func (d *DAO) UpsertMember(ctx context.Context, in *Member) error {
	if in.ID > 0 {
		res, err := d.db.ExecContext(ctx, `
UPDATE account_pool_members
   SET enabled = ?, weight = ?, priority = ?, max_parallel = ?, note = ?
 WHERE id = ? AND pool_id = ?`,
			in.Enabled, in.Weight, in.Priority, in.MaxParallel, in.Note, in.ID, in.PoolID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrNotFound
		}
		return nil
	}

	res, err := d.db.ExecContext(ctx, `
INSERT INTO account_pool_members
  (pool_id, account_id, enabled, weight, priority, max_parallel, note)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  enabled = VALUES(enabled),
  weight = VALUES(weight),
  priority = VALUES(priority),
  max_parallel = VALUES(max_parallel),
  note = VALUES(note)`,
		in.PoolID, in.AccountID, in.Enabled, in.Weight, in.Priority, in.MaxParallel, in.Note)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	if id > 0 {
		in.ID = uint64(id)
	}
	return nil
}

func (d *DAO) DeleteMember(ctx context.Context, poolID, memberID uint64) error {
	res, err := d.db.ExecContext(ctx,
		`DELETE FROM account_pool_members WHERE id = ? AND pool_id = ?`, memberID, poolID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DAO) ListRoutes(ctx context.Context) ([]*ModelRoute, error) {
	out := make([]*ModelRoute, 0, 16)
	err := d.db.SelectContext(ctx, &out,
		`SELECT * FROM model_pool_routes ORDER BY model_id ASC`)
	return out, err
}

func (d *DAO) GetRouteByModelID(ctx context.Context, modelID uint64) (*ModelRoute, error) {
	var out ModelRoute
	err := d.db.GetContext(ctx, &out,
		`SELECT * FROM model_pool_routes WHERE model_id = ? LIMIT 1`, modelID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &out, err
}

func (d *DAO) UpsertRoute(ctx context.Context, in *ModelRoute) error {
	res, err := d.db.ExecContext(ctx, `
INSERT INTO model_pool_routes (model_id, pool_id, fallback_pool_id, enabled)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  pool_id = VALUES(pool_id),
  fallback_pool_id = VALUES(fallback_pool_id),
  enabled = VALUES(enabled)`,
		in.ModelID, in.PoolID, in.FallbackPoolID, in.Enabled)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	if id > 0 {
		in.ID = uint64(id)
	}
	return nil
}

func (d *DAO) DeleteRoute(ctx context.Context, modelID uint64) error {
	res, err := d.db.ExecContext(ctx,
		`DELETE FROM model_pool_routes WHERE model_id = ?`, modelID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isValidPoolType(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "chat", "image", "codex", "mixed":
		return true
	default:
		return false
	}
}
