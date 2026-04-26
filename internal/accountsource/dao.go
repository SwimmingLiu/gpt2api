package accountsource

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type DAO struct {
	db *sqlx.DB
}

func NewDAO(db *sqlx.DB) *DAO { return &DAO{db: db} }

func (d *DAO) ListSources(ctx context.Context) ([]*Source, error) {
	var items []*Source
	const q = `SELECT id, source_type, name, base_url, enabled, auth_mode, email, group_id,
		COALESCE(api_key_enc, '') AS api_key_enc,
		COALESCE(password_enc, '') AS password_enc,
		COALESCE(secret_key_enc, '') AS secret_key_enc,
		default_proxy_id, target_pool_id, created_at, updated_at, deleted_at
	FROM account_import_sources
	WHERE deleted_at IS NULL
	ORDER BY id DESC`
	if err := d.db.SelectContext(ctx, &items, q); err != nil {
		return nil, err
	}
	return items, nil
}

func (d *DAO) GetSourceByID(ctx context.Context, id uint64) (*Source, error) {
	var item Source
	const q = `SELECT id, source_type, name, base_url, enabled, auth_mode, email, group_id,
		COALESCE(api_key_enc, '') AS api_key_enc,
		COALESCE(password_enc, '') AS password_enc,
		COALESCE(secret_key_enc, '') AS secret_key_enc,
		default_proxy_id, target_pool_id, created_at, updated_at, deleted_at
	FROM account_import_sources
	WHERE id = ? AND deleted_at IS NULL
	LIMIT 1`
	if err := d.db.GetContext(ctx, &item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (d *DAO) CreateSource(ctx context.Context, src *Source) error {
	const q = `INSERT INTO account_import_sources
		(source_type, name, base_url, enabled, auth_mode, email, group_id, api_key_enc, password_enc, secret_key_enc, default_proxy_id, target_pool_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := d.db.ExecContext(ctx, q,
		src.SourceType, src.Name, src.BaseURL, src.Enabled, src.AuthMode, src.Email, src.GroupID,
		src.APIKeyEnc, src.PasswordEnc, src.SecretKeyEnc, src.DefaultProxyID, src.TargetPoolID,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	src.ID = uint64(id)
	return nil
}

func (d *DAO) UpdateSource(ctx context.Context, src *Source) error {
	const q = `UPDATE account_import_sources
	SET source_type = ?, name = ?, base_url = ?, enabled = ?, auth_mode = ?, email = ?, group_id = ?,
		api_key_enc = ?, password_enc = ?, secret_key_enc = ?, default_proxy_id = ?, target_pool_id = ?
	WHERE id = ? AND deleted_at IS NULL`
	res, err := d.db.ExecContext(ctx, q,
		src.SourceType, src.Name, src.BaseURL, src.Enabled, src.AuthMode, src.Email, src.GroupID,
		src.APIKeyEnc, src.PasswordEnc, src.SecretKeyEnc, src.DefaultProxyID, src.TargetPoolID, src.ID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DAO) SoftDeleteSource(ctx context.Context, id uint64) error {
	const q = `UPDATE account_import_sources
	SET enabled = 0, deleted_at = CURRENT_TIMESTAMP
	WHERE id = ? AND deleted_at IS NULL`
	res, err := d.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
