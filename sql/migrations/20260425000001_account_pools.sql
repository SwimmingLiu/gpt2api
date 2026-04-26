-- +goose Up
-- +goose StatementBegin
CREATE TABLE `account_pools` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `code` VARCHAR(64) NOT NULL,
    `name` VARCHAR(128) NOT NULL,
    `pool_type` VARCHAR(32) NOT NULL DEFAULT 'mixed',
    `description` VARCHAR(255) NOT NULL DEFAULT '',
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `dispatch_strategy` VARCHAR(32) NOT NULL DEFAULT 'least_recently_used',
    `sticky_ttl_sec` INT NOT NULL DEFAULT 0,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_account_pools_code` (`code`)
);

CREATE TABLE `account_pool_members` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `pool_id` BIGINT UNSIGNED NOT NULL,
    `account_id` BIGINT UNSIGNED NOT NULL,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `weight` INT NOT NULL DEFAULT 100,
    `priority` INT NOT NULL DEFAULT 100,
    `max_parallel` INT NOT NULL DEFAULT 1,
    `note` VARCHAR(255) NOT NULL DEFAULT '',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_account_pool_member` (`pool_id`, `account_id`),
    KEY `idx_account_pool_members_account_id` (`account_id`)
);

CREATE TABLE `model_pool_routes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `model_id` BIGINT UNSIGNED NOT NULL,
    `pool_id` BIGINT UNSIGNED NOT NULL,
    `fallback_pool_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_model_pool_routes_model_id` (`model_id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `model_pool_routes`;
DROP TABLE IF EXISTS `account_pool_members`;
DROP TABLE IF EXISTS `account_pools`;
-- +goose StatementEnd
