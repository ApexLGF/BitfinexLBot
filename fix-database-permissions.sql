-- 修复BitfinexBot数据库权限问题
-- 在生产环境MySQL中执行此脚本

-- 1. 创建BitfinexBot数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS bitfinex_bot_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 2. 创建或更新BitfinexBot用户
-- 删除可能存在的用户（避免冲突）
DROP USER IF EXISTS 'bitfinex_bot'@'%';
DROP USER IF EXISTS 'bitfinex_bot'@'localhost';
DROP USER IF EXISTS 'bitfinex_bot'@'172.19.0.4';

-- 创建新用户，允许从任何IP连接
CREATE USER 'bitfinex_bot'@'%' IDENTIFIED BY 'BitfinexBot@2025';

-- 3. 授予权限
GRANT ALL PRIVILEGES ON bitfinex_bot_db.* TO 'bitfinex_bot'@'%';

-- 4. 刷新权限
FLUSH PRIVILEGES;

-- 5. 验证用户创建
SELECT User, Host FROM mysql.user WHERE User = 'bitfinex_bot';

-- 6. 验证数据库存在
SHOW DATABASES LIKE 'bitfinex_bot_db';