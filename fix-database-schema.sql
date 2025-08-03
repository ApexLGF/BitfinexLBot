-- 修复数据库表结构 - command_type 字段
-- 解决: SQLSTATE[01000]: Warning: 1265 Data truncated for column 'command_type' at row 1

USE bitfinex_bot_db;

-- 1. 修改 web_commands 表的 command_type 字段，添加缺失的枚举值
ALTER TABLE web_commands MODIFY COLUMN command_type ENUM(
    'start', 'stop', 'restart', 'status', 
    'balance', 'rates', 'orders', 'config',
    'cancel_all', 'lending_check', 'rate_check',
    'set_config', 'get_config',
    -- 新增的命令类型
    'credits', 'wallets', 'daily_earnings'
) NOT NULL;

-- 2. 验证修改结果
DESCRIBE web_commands;

-- 3. 显示command_type字段的详细信息
SHOW COLUMNS FROM web_commands LIKE 'command_type';

-- 4. 清理可能存在的问题数据（可选）
-- DELETE FROM web_commands WHERE command_type NOT IN (
--     'start', 'stop', 'restart', 'status', 
--     'balance', 'rates', 'orders', 'config',
--     'cancel_all', 'lending_check', 'rate_check',
--     'set_config', 'get_config', 'credits', 'wallets', 'daily_earnings'
-- );

SELECT 'Database schema fix completed successfully' as message;