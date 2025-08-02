#!/bin/bash

# BitfinexBot Docker 环境设置脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "=========================================="
echo "   BitfinexBot Docker 环境初始化"
echo "=========================================="
echo -e "${NC}"

# 检查 Docker 和 Docker Compose
echo -e "${YELLOW}检查 Docker 环境...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: Docker 未安装或未在 PATH 中${NC}"
    exit 1
fi

# 检查 Docker Compose（支持新旧版本）
DOCKER_COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
else
    echo -e "${RED}错误: Docker Compose 未安装或未在 PATH 中${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker Compose 命令: $DOCKER_COMPOSE_CMD${NC}"

echo -e "${GREEN}✓ Docker 环境检查通过${NC}"

# 检查 Docker 服务是否运行
if ! docker info &> /dev/null; then
    echo -e "${RED}错误: Docker 服务未运行，请启动 Docker${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker 服务运行正常${NC}"

# 创建必要的目录
echo -e "${YELLOW}创建必要的目录...${NC}"
mkdir -p docker logs

# 停止并清理现有容器（如果存在）
echo -e "${YELLOW}清理现有容器...${NC}"
$DOCKER_COMPOSE_CMD down --remove-orphans 2>/dev/null || true

# 构建和启动服务
echo -e "${YELLOW}构建 Docker 镜像...${NC}"
$DOCKER_COMPOSE_CMD build --no-cache

echo -e "${YELLOW}启动服务...${NC}"
$DOCKER_COMPOSE_CMD up -d mysql php web phpmyadmin

# 等待 MySQL 健康检查通过
echo -e "${YELLOW}等待 MySQL 数据库启动...${NC}"
timeout=120
counter=0
while [ $counter -lt $timeout ]; do
    if $DOCKER_COMPOSE_CMD exec mysql mysqladmin ping -h localhost -u bitfinex_bot -pBitfinexBot@2025 --silent; then
        echo -e "${GREEN}✓ MySQL 数据库启动成功${NC}"
        break
    fi
    
    echo -n "."
    sleep 2
    counter=$((counter + 2))
done

if [ $counter -ge $timeout ]; then
    echo -e "${RED}错误: MySQL 数据库启动超时${NC}"
    echo -e "${YELLOW}查看日志:${NC}"
    $DOCKER_COMPOSE_CMD logs mysql
    exit 1
fi

# 等待额外时间确保数据库完全就绪
echo -e "${YELLOW}等待数据库完全初始化...${NC}"
sleep 10

# 验证数据库初始化
echo -e "${YELLOW}验证数据库初始化...${NC}"
DB_TABLES=$($DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -e "SHOW TABLES;" 2>/dev/null | wc -l)

if [ "$DB_TABLES" -gt 1 ]; then
    echo -e "${GREEN}✓ 数据库表创建成功 ($((DB_TABLES - 1)) 个表)${NC}"
else
    echo -e "${RED}错误: 数据库表创建失败${NC}"
    echo -e "${YELLOW}手动初始化数据库...${NC}"
    $DOCKER_COMPOSE_CMD exec -T mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db < database/schema.sql
fi

# 测试 Web 服务
echo -e "${YELLOW}测试 Web 服务...${NC}"
sleep 5
if curl -f http://localhost:8080 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Web 服务启动成功${NC}"
else
    echo -e "${YELLOW}⚠ Web 服务可能未完全启动，请稍后检查${NC}"
fi

# 显示服务状态
echo -e "${BLUE}"
echo "=========================================="
echo "           服务状态"
echo "=========================================="
echo -e "${NC}"

$DOCKER_COMPOSE_CMD ps

echo -e "${BLUE}"
echo "=========================================="
echo "         访问信息"
echo "=========================================="
echo -e "${NC}"

echo -e "${GREEN}🌐 Web 管理界面:${NC} http://localhost:8080"
echo -e "${GREEN}🗄️ phpMyAdmin:${NC}   http://localhost:8081"
echo -e "${GREEN}📊 数据库连接:${NC}   localhost:3306"
echo ""
echo -e "${PURPLE}数据库信息:${NC}"
echo -e "  用户名: bitfinex_bot"
echo -e "  密码:   BitfinexBot@2025"
echo -e "  数据库: bitfinex_bot_db"

echo -e "${BLUE}"
echo "=========================================="
echo "         后续操作"
echo "=========================================="
echo -e "${NC}"

echo -e "${YELLOW}1. 配置 Bitfinex API 密钥:${NC}"
echo "   编辑 config.yaml 文件，设置您的 API 密钥"
echo ""
echo -e "${YELLOW}2. 启动 BitfinexBot:${NC}"
echo "   $DOCKER_COMPOSE_CMD up -d bitfinex-bot"
echo ""
echo -e "${YELLOW}3. 查看日志:${NC}"
echo "   $DOCKER_COMPOSE_CMD logs -f bitfinex-bot"
echo ""
echo -e "${YELLOW}4. 停止服务:${NC}"
echo "   $DOCKER_COMPOSE_CMD down"

echo -e "${GREEN}"
echo "🎉 Docker 环境初始化完成！"
echo -e "${NC}"