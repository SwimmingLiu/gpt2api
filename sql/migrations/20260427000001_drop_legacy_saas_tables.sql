-- +goose Up
-- +goose StatementBegin

DROP TABLE IF EXISTS `recharge_packages`;
DROP TABLE IF EXISTS `recharge_orders`;
DROP TABLE IF EXISTS `backup_files`;
DROP TABLE IF EXISTS `admin_audit_logs`;
DROP TABLE IF EXISTS `usage_logs`;
DROP TABLE IF EXISTS `billing_ratios`;
DROP TABLE IF EXISTS `credit_transactions`;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `credit_transactions` (
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`        BIGINT UNSIGNED NOT NULL,
    `key_id`         BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `type`           VARCHAR(32)     NOT NULL,
    `amount`         BIGINT          NOT NULL,
    `balance_after`  BIGINT          NOT NULL,
    `ref_id`         VARCHAR(64)     NOT NULL DEFAULT '',
    `remark`         VARCHAR(255)    NOT NULL DEFAULT '',
    `created_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `billing_ratios` (
    `id`        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `group_id`  BIGINT UNSIGNED NOT NULL,
    `model_id`  BIGINT UNSIGNED NOT NULL,
    `ratio`     DECIMAL(6,4)    NOT NULL DEFAULT 1.0000,
    `created_at` DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `usage_logs` (
    `id`                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`             BIGINT UNSIGNED NOT NULL,
    `key_id`              BIGINT UNSIGNED NOT NULL,
    `account_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `request_id`          VARCHAR(64)     NOT NULL DEFAULT '',
    `type`                VARCHAR(16)     NOT NULL,
    `model_id`            BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `input_tokens`        BIGINT          NOT NULL DEFAULT 0,
    `output_tokens`       BIGINT          NOT NULL DEFAULT 0,
    `image_count`         INT             NOT NULL DEFAULT 0,
    `credit_cost`         BIGINT          NOT NULL DEFAULT 0,
    `duration_ms`         INT             NOT NULL DEFAULT 0,
    `status`              VARCHAR(16)     NOT NULL DEFAULT 'success',
    `error_code`          VARCHAR(64)     NOT NULL DEFAULT '',
    `ip`                  VARCHAR(64)     NOT NULL DEFAULT '',
    `ua`                  VARCHAR(255)    NOT NULL DEFAULT '',
    `created_at`          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `admin_audit_logs` (
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `actor_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `actor_email`  VARCHAR(128)    NOT NULL DEFAULT '',
    `action`       VARCHAR(128)    NOT NULL,
    `method`       VARCHAR(8)      NOT NULL,
    `path`         VARCHAR(255)    NOT NULL,
    `status_code`  INT             NOT NULL DEFAULT 0,
    `ip`           VARCHAR(64)     NOT NULL DEFAULT '',
    `ua`           VARCHAR(255)    NOT NULL DEFAULT '',
    `target`       VARCHAR(128)    NOT NULL DEFAULT '',
    `meta`         JSON            NULL,
    `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `backup_files` (
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `backup_id`    VARCHAR(64)     NOT NULL,
    `file_name`    VARCHAR(255)    NOT NULL,
    `size_bytes`   BIGINT          NOT NULL DEFAULT 0,
    `sha256`       CHAR(64)        NOT NULL DEFAULT '',
    `trigger`      VARCHAR(16)     NOT NULL DEFAULT 'manual',
    `status`       VARCHAR(16)     NOT NULL DEFAULT 'running',
    `error`        VARCHAR(500)    NOT NULL DEFAULT '',
    `include_data` TINYINT(1)      NOT NULL DEFAULT 1,
    `created_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `finished_at`  DATETIME        NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `recharge_packages` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(64)     NOT NULL,
    `price_cny`   INT             NOT NULL,
    `credits`     BIGINT          NOT NULL,
    `bonus`       BIGINT          NOT NULL DEFAULT 0,
    `description` VARCHAR(255)    NOT NULL DEFAULT '',
    `sort`        INT             NOT NULL DEFAULT 0,
    `enabled`     TINYINT(1)      NOT NULL DEFAULT 1,
    `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `recharge_orders` (
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `out_trade_no` CHAR(32)        NOT NULL,
    `user_id`      BIGINT UNSIGNED NOT NULL,
    `package_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `price_cny`    INT             NOT NULL,
    `credits`      BIGINT          NOT NULL,
    `bonus`        BIGINT          NOT NULL DEFAULT 0,
    `channel`      VARCHAR(16)     NOT NULL DEFAULT 'epay',
    `pay_method`   VARCHAR(16)     NOT NULL DEFAULT '',
    `status`       VARCHAR(16)     NOT NULL DEFAULT 'pending',
    `trade_no`     VARCHAR(64)     NOT NULL DEFAULT '',
    `paid_at`      DATETIME        NULL,
    `pay_url`      VARCHAR(512)    NOT NULL DEFAULT '',
    `client_ip`    VARCHAR(64)     NOT NULL DEFAULT '',
    `notify_raw`   TEXT            NULL,
    `remark`       VARCHAR(255)    NOT NULL DEFAULT '',
    `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose StatementEnd
