-- BitfinexBot 数据库表结构
-- 最后更新: 2025-08-11
-- 基于实际运行数据库导出的完整表结构

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS bitfinex_bot_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE bitfinex_bot_db;

-- 1. 用户管理表
CREATE TABLE IF NOT EXISTS `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `username` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `password_hash` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `role` enum('admin','user') COLLATE utf8mb4_unicode_ci DEFAULT 'user',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `last_login` timestamp NULL DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Web命令队列表
CREATE TABLE IF NOT EXISTS `web_commands` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `command_type` enum('start','stop','restart','status','balance','rates','orders','credits','config','cancel_all','lending_check','rate_check','set_config','get_config','update_all_config','wallets','daily_earnings','funding_sync','earnings_daily','earnings_weekly','earnings_monthly','earnings_report','process_actual_earnings') COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `command_data` json DEFAULT NULL,
  `status` enum('pending','processing','completed','failed') COLLATE utf8mb4_unicode_ci DEFAULT 'pending',
  `result` text COLLATE utf8mb4_unicode_ci,
  `error_message` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `processed_at` timestamp NULL DEFAULT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_user_command` (`user_id`,`command_type`),
  CONSTRAINT `web_commands_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. 机器人实时状态表
CREATE TABLE IF NOT EXISTS `bot_realtime_status` (
  `id` int NOT NULL AUTO_INCREMENT,
  `status` enum('running','stopped','error','maintenance') COLLATE utf8mb4_unicode_ci DEFAULT 'stopped',
  `last_execution` timestamp NULL DEFAULT NULL,
  `next_execution` timestamp NULL DEFAULT NULL,
  `total_balance` decimal(20,8) DEFAULT '0.00000000',
  `available_balance` decimal(20,8) DEFAULT '0.00000000',
  `active_orders` int DEFAULT '0',
  `current_rate` decimal(10,6) DEFAULT '0.000000',
  `avg_rate` decimal(10,6) DEFAULT '0.000000',
  `total_earned` decimal(20,8) DEFAULT '0.00000000',
  `weekly_earned` decimal(20,8) DEFAULT '0.00000000' COMMENT '1周收益',
  `error_count` int DEFAULT '0',
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `performance_metrics` json DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. 通知历史表
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` int NOT NULL AUTO_INCREMENT,
  `type` enum('rate_threshold','error','lending','system','info') COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `message` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `level` enum('info','warning','error','success') COLLATE utf8mb4_unicode_ci DEFAULT 'info',
  `is_read` tinyint(1) DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_type_created` (`type`,`created_at`),
  KEY `idx_is_read` (`is_read`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. 系统配置表（动态配置存储）
CREATE TABLE IF NOT EXISTS `system_config` (
  `id` int NOT NULL AUTO_INCREMENT,
  `config_key` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `config_value` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `config_type` enum('string','number','boolean','json') COLLATE utf8mb4_unicode_ci DEFAULT 'string',
  `description` text COLLATE utf8mb4_unicode_ci,
  `is_sensitive` tinyint(1) DEFAULT '0',
  `updated_by` int DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `config_key` (`config_key`),
  KEY `idx_config_key` (`config_key`),
  KEY `updated_by` (`updated_by`),
  CONSTRAINT `system_config_ibfk_1` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. 操作日志表
CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int DEFAULT NULL,
  `operation_type` enum('command','config_change','login','logout','system') COLLATE utf8mb4_unicode_ci NOT NULL,
  `operation_desc` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ip_address` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` text COLLATE utf8mb4_unicode_ci,
  `result` enum('success','failure') COLLATE utf8mb4_unicode_ci DEFAULT 'success',
  `details` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_operation` (`user_id`,`operation_type`),
  KEY `idx_created_at` (`created_at`),
  CONSTRAINT `operation_logs_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. 订单历史表
CREATE TABLE IF NOT EXISTS `order_history` (
  `id` int NOT NULL AUTO_INCREMENT,
  `order_id` bigint NOT NULL,
  `symbol` varchar(10) COLLATE utf8mb4_unicode_ci DEFAULT 'fUSD',
  `side` enum('lend') COLLATE utf8mb4_unicode_ci DEFAULT 'lend',
  `amount` decimal(20,8) NOT NULL,
  `rate` decimal(10,6) NOT NULL,
  `period` int NOT NULL,
  `status` enum('active','executed','cancelled') COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `executed_at` timestamp NULL DEFAULT NULL,
  `cancelled_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_status_created` (`status`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 8. 每日收益汇总表
CREATE TABLE IF NOT EXISTS `daily_earnings_summary` (
  `id` int NOT NULL AUTO_INCREMENT,
  `currency` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `summary_date` date NOT NULL,
  `total_trades` int DEFAULT '0',
  `total_amount` decimal(20,8) DEFAULT '0.00000000',
  `total_earnings` decimal(20,8) DEFAULT '0.00000000',
  `avg_rate` decimal(12,8) DEFAULT '0.00000000',
  `min_rate` decimal(12,8) DEFAULT '0.00000000',
  `max_rate` decimal(12,8) DEFAULT '0.00000000',
  `avg_period` decimal(8,2) DEFAULT '0.00',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_currency_date` (`currency`,`summary_date`),
  KEY `idx_currency_date` (`currency`,`summary_date`),
  KEY `idx_summary_date` (`summary_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 9. 资金交易历史表 (新增表)
CREATE TABLE IF NOT EXISTS `funding_trades_history` (
  `id` int NOT NULL AUTO_INCREMENT,
  `bitfinex_trade_id` bigint NOT NULL,
  `currency` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `offer_id` bigint NOT NULL,
  `amount` decimal(20,8) NOT NULL,
  `rate` decimal(12,8) NOT NULL,
  `period` int NOT NULL,
  `mts_create` bigint NOT NULL,
  `mts_close` bigint DEFAULT NULL,
  `earnings` decimal(20,8) DEFAULT '0.00000000',
  `actual_earnings` decimal(20,8) DEFAULT '0.00000000',
  `trade_date` date GENERATED ALWAYS AS (cast(from_unixtime((`mts_create` / 1000)) as date)) STORED,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_trade` (`bitfinex_trade_id`,`offer_id`),
  KEY `idx_bitfinex_trade_id` (`bitfinex_trade_id`),
  KEY `idx_offer_id` (`offer_id`),
  KEY `idx_currency_date` (`currency`,`trade_date`),
  KEY `idx_mts_create` (`mts_create`),
  KEY `idx_trade_date` (`trade_date`),
  KEY `idx_mts_close` (`mts_close`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入默认数据

-- 插入默认管理员用户 (密码: admin123)
INSERT IGNORE INTO users (username, password_hash, email, role) VALUES 
('admin', '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@bitfinexbot.local', 'admin');

-- 插入默认机器人状态
INSERT IGNORE INTO bot_realtime_status (status) VALUES ('stopped');

-- 插入默认系统配置
INSERT IGNORE INTO system_config (config_key, config_value, config_type, description) VALUES 
('MINUTES_RUN', '30', 'number', '主要任务执行间隔（分钟）'),
('MIN_DAILY_LEND_RATE', '0.024', 'number', '最低日利率'),
('SPREAD_LEND', '30', 'number', '资金分散笔数'),
('GAP_BOTTOM', '10', 'number', '订单深度下限'),
('GAP_TOP', '5000', 'number', '订单深度上限'),
('HIGH_HOLD_RATE', '0.1', 'number', '高额持有利率'),
('HIGH_HOLD_AMOUNT', '155', 'number', '高额持有金额'),
('TEST_MODE', 'false', 'boolean', '测试模式'),
('ENABLE_SMART_STRATEGY', 'true', 'boolean', '启用智能策略'),
('NOTIFY_RATE_THRESHOLD', '0.1', 'number', '利率通知阈值');