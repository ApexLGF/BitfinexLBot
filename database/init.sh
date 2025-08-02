#!/bin/bash

# BitfinexBot 数据库初始化脚本
# 使用方法: ./init.sh [mysql_host] [mysql_user] [mysql_password]

set -e

# 默认参数
MYSQL_HOST=${1:-"localhost"}
MYSQL_USER=${2:-"root"}
MYSQL_PASSWORD=${3:-""}
DB_NAME="bitfinex_bot_db"
DB_USER="bitfinex_bot"
DB_PASSWORD="BitfinexBot@2025"

echo "=== BitfinexBot 数据库初始化 ==="
echo "MySQL 主机: $MYSQL_HOST"
echo "MySQL 用户: $MYSQL_USER"
echo "数据库名称: $DB_NAME"
echo "应用用户: $DB_USER"
echo ""

# 检查 MySQL 连接
echo "检查 MySQL 连接..."
if ! mysql -h"$MYSQL_HOST" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1;" > /dev/null 2>&1; then
    echo "错误: 无法连接到 MySQL 服务器"
    echo "请检查连接参数是否正确"
    exit 1
fi

echo "MySQL 连接正常"

# 创建数据库用户和权限
echo "创建数据库用户和权限..."
mysql -h"$MYSQL_HOST" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" <<EOF
-- 创建数据库用户
CREATE USER IF NOT EXISTS '$DB_USER'@'localhost' IDENTIFIED BY '$DB_PASSWORD';
CREATE USER IF NOT EXISTS '$DB_USER'@'%' IDENTIFIED BY '$DB_PASSWORD';

-- 授予权限
GRANT ALL PRIVILEGES ON $DB_NAME.* TO '$DB_USER'@'localhost';
GRANT ALL PRIVILEGES ON $DB_NAME.* TO '$DB_USER'@'%';

-- 刷新权限
FLUSH PRIVILEGES;
EOF

echo "数据库用户创建完成"

# 执行数据库架构脚本
echo "创建数据库表结构..."
mysql -h"$MYSQL_HOST" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" < "$(dirname "$0")/schema.sql"

echo "数据库表结构创建完成"

# 验证安装
echo "验证数据库安装..."
TABLE_COUNT=$(mysql -h"$MYSQL_HOST" -u"$DB_USER" -p"$DB_PASSWORD" -D"$DB_NAME" -se "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$DB_NAME';")

if [ "$TABLE_COUNT" -gt 0 ]; then
    echo "✅ 数据库初始化成功!"
    echo "   - 创建了 $TABLE_COUNT 个数据表"
    echo "   - 数据库用户: $DB_USER"
    echo "   - 数据库名称: $DB_NAME"
    echo ""
    echo "请更新 config.yaml 中的数据库配置:"
    echo "database:"
    echo "  host: \"$MYSQL_HOST\""
    echo "  port: 3306"
    echo "  username: \"$DB_USER\""
    echo "  password: \"$DB_PASSWORD\""
    echo "  database: \"$DB_NAME\""
    echo "  charset: \"utf8mb4\""
else
    echo "❌ 数据库初始化失败!"
    exit 1
fi