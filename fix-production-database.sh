#!/bin/bash
# 修复生产环境数据库权限问题

echo "🔧 修复BitfinexBot生产环境数据库权限..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 检查MySQL容器是否运行
echo "1. 检查MySQL容器状态..."
if ! docker compose ps mysql | grep -q "running"; then
    echo -e "${RED}❌ MySQL容器未运行，请先启动：docker compose up -d mysql${NC}"
    exit 1
fi
echo -e "${GREEN}✅ MySQL容器正在运行${NC}"

# 2. 等待MySQL完全启动
echo "2. 等待MySQL完全启动..."
sleep 10

# 3. 执行权限修复脚本
echo "3. 执行数据库权限修复..."
docker compose exec mysql mysql -u root -prootpassword <<EOF
-- 创建BitfinexBot数据库
CREATE DATABASE IF NOT EXISTS bitfinex_bot_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 删除可能存在的用户
DROP USER IF EXISTS 'bitfinex_bot'@'%';
DROP USER IF EXISTS 'bitfinex_bot'@'localhost';

-- 创建新用户
CREATE USER 'bitfinex_bot'@'%' IDENTIFIED BY 'BitfinexBot@2025';

-- 授予权限
GRANT ALL PRIVILEGES ON bitfinex_bot_db.* TO 'bitfinex_bot'@'%';

-- 刷新权限
FLUSH PRIVILEGES;

-- 验证
SELECT User, Host FROM mysql.user WHERE User = 'bitfinex_bot';
SHOW DATABASES LIKE 'bitfinex_bot_db';
EOF

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 数据库权限修复成功${NC}"
else
    echo -e "${RED}❌ 数据库权限修复失败${NC}"
    exit 1
fi

# 4. 创建数据库表结构
echo "4. 创建BitfinexBot数据库表..."
docker compose exec mysql mysql -u bitfinex_bot -p'BitfinexBot@2025' bitfinex_bot_db < bitfinex-bot/database/schema.sql

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 数据库表创建成功${NC}"
else
    echo -e "${YELLOW}⚠️  数据库表创建可能失败，请检查schema.sql文件${NC}"
fi

# 5. 测试连接
echo "5. 测试数据库连接..."
docker compose exec mysql mysql -u bitfinex_bot -p'BitfinexBot@2025' -e "USE bitfinex_bot_db; SHOW TABLES;" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 数据库连接测试成功${NC}"
    echo -e "${GREEN}🎉 BitfinexBot数据库权限修复完成！${NC}"
    echo ""
    echo -e "${YELLOW}接下来可以重启BitfinexBot服务：${NC}"
    echo "docker compose restart bitfinex-bot"
    echo "docker compose logs -f bitfinex-bot"
else
    echo -e "${RED}❌ 数据库连接测试失败${NC}"
    exit 1
fi