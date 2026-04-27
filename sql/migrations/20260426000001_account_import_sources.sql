-- +goose Up
-- +goose StatementBegin
CREATE TABLE `account_import_sources` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `source_type` VARCHAR(32) NOT NULL,
    `name` VARCHAR(128) NOT NULL,
    `base_url` VARCHAR(255) NOT NULL,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `auth_mode` VARCHAR(32) NOT NULL DEFAULT '',
    `email` VARCHAR(255) NOT NULL DEFAULT '',
    `group_id` VARCHAR(128) NOT NULL DEFAULT '',
    `api_key_enc` TEXT NULL,
    `password_enc` TEXT NULL,
    `secret_key_enc` TEXT NULL,
    `default_proxy_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `target_pool_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME NULL,
    PRIMARY KEY (`id`),
    KEY `idx_account_import_sources_type_enabled` (`source_type`, `enabled`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `account_import_sources`;
-- +goose StatementEnd
