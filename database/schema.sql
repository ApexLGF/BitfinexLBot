-- BitfinexBot 数据库表结构
-- 创建时间: 2025-08-01

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS bitfinex_bot_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE bitfinex_bot_db;

-- 1. 用户管理表
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    role ENUM('admin', 'user') DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Web命令队列表
CREATE TABLE IF NOT EXISTS web_commands (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    command_type ENUM(
        'start', 'stop', 'restart', 'status', 
        'balance', 'rates', 'orders', 'config',
        'cancel_all', 'lending_check', 'rate_check',
        'set_config', 'get_config', 'update_all_config',
        'credits', 'wallets', 'daily_earnings'
    ) NOT NULL,
    command_data JSON,                    -- 命令参数 (JSON格式)
    status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending',
    result TEXT,                          -- 命令执行结果
    error_message TEXT,                   -- 错误信息
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL,          -- 处理时间
    completed_at TIMESTAMP NULL,          -- 完成时间
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_user_command (user_id, command_type),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. 机器人实时状态表
CREATE TABLE IF NOT EXISTS bot_realtime_status (
    id INT AUTO_INCREMENT PRIMARY KEY,
    status ENUM('running', 'stopped', 'error', 'maintenance') DEFAULT 'stopped',
    last_execution TIMESTAMP NULL,       -- 上次执行时间
    next_execution TIMESTAMP NULL,       -- 下次执行时间
    total_balance DECIMAL(20, 8) DEFAULT 0.00000000,    -- 总余额
    available_balance DECIMAL(20, 8) DEFAULT 0.00000000, -- 可用余额
    active_orders INT DEFAULT 0,         -- 活跃订单数
    current_rate DECIMAL(10, 6) DEFAULT 0.000000,       -- 当前利率
    avg_rate DECIMAL(10, 6) DEFAULT 0.000000,           -- 平均利率
    total_earned DECIMAL(20, 8) DEFAULT 0.00000000,     -- 1日收益
    weekly_earned DECIMAL(20, 8) DEFAULT 0.00000000,    -- 1周收益
    error_count INT DEFAULT 0,           -- 错误计数
    last_error TEXT,                     -- 最后错误信息
    performance_metrics JSON,            -- 性能指标 (JSON)
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. 通知历史表
CREATE TABLE IF NOT EXISTS notifications (
    id INT AUTO_INCREMENT PRIMARY KEY,
    type ENUM('rate_threshold', 'error', 'lending', 'system', 'info') NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    level ENUM('info', 'warning', 'error', 'success') DEFAULT 'info',
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_type_created (type, created_at),
    INDEX idx_is_read (is_read)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. 系统配置表（动态配置存储）
CREATE TABLE IF NOT EXISTS system_config (
    id INT AUTO_INCREMENT PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    config_type ENUM('string', 'number', 'boolean', 'json') DEFAULT 'string',
    description TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,   -- 是否敏感信息
    updated_by INT,                       -- 更新用户ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_config_key (config_key),
    FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    operation_type ENUM('command', 'config_change', 'login', 'logout', 'system') NOT NULL,
    operation_desc VARCHAR(500) NOT NULL,
    ip_address VARCHAR(45),               -- 支持IPv6
    user_agent TEXT,
    result ENUM('success', 'failure') DEFAULT 'success',
    details JSON,                         -- 详细信息
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_operation (user_id, operation_type),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. 订单历史表（可选，用于统计分析）
CREATE TABLE IF NOT EXISTS order_history (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,            -- Bitfinex 订单ID
    symbol VARCHAR(10) DEFAULT 'fUSD',   -- 交易对
    side ENUM('lend') DEFAULT 'lend',    -- 订单类型
    amount DECIMAL(20, 8) NOT NULL,      -- 金额
    rate DECIMAL(10, 6) NOT NULL,        -- 利率
    period INT NOT NULL,                 -- 期间（天数）
    status ENUM('active', 'executed', 'cancelled') NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    executed_at TIMESTAMP NULL,          -- 成交时间
    cancelled_at TIMESTAMP NULL,         -- 取消时间
    INDEX idx_order_id (order_id),
    INDEX idx_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入默认数据

-- 插入默认管理员用户 (密码: admin123)
INSERT INTO users (username, password_hash, email, role) VALUES 
('admin', '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@bitfinexbot.local', 'admin')
ON DUPLICATE KEY UPDATE password_hash=VALUES(password_hash);

-- 插入默认机器人状态
INSERT INTO bot_realtime_status (status) VALUES ('stopped')
ON DUPLICATE KEY UPDATE status=VALUES(status);

-- 插入默认系统配置
INSERT INTO system_config (config_key, config_value, config_type, description) VALUES 
('MINUTES_RUN', '30', 'number', '主要任务执行间隔（分钟）'),
('MIN_DAILY_LEND_RATE', '0.024', 'number', '最低日利率'),
('SPREAD_LEND', '30', 'number', '资金分散笔数'),
('GAP_BOTTOM', '10', 'number', '订单深度下限'),
('GAP_TOP', '5000', 'number', '订单深度上限'),
('HIGH_HOLD_RATE', '0.1', 'number', '高额持有利率'),
('HIGH_HOLD_AMOUNT', '155', 'number', '高额持有金额'),
('TEST_MODE', 'false', 'boolean', '测试模式'),
('ENABLE_SMART_STRATEGY', 'true', 'boolean', '启用智能策略'),
('NOTIFY_RATE_THRESHOLD', '0.1', 'number', '利率通知阈值')
ON DUPLICATE KEY UPDATE config_value=VALUES(config_value);