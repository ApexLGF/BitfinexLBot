#!/bin/bash

# BitfinexBot 集成到现有Docker环境的部署脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "=========================================="
echo "   BitfinexBot 集成部署到现有环境"
echo "=========================================="
echo -e "${NC}"

# 检查当前目录
if [ ! -f "config.yaml" ] && [ ! -f "config-existing.yaml" ]; then
    echo -e "${RED}错误: 请在 BitfinexBot 项目目录中运行此脚本${NC}"
    exit 1
fi

# 检查 Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}错误: Docker Compose 未安装${NC}"
    exit 1
fi

# 检测 Docker Compose 命令
DOCKER_COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
else
    echo -e "${RED}错误: Docker Compose 未安装${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 使用 Docker Compose 命令: $DOCKER_COMPOSE_CMD${NC}"

# 检查现有容器是否运行
if ! docker ps | grep -q "mysql\|nginx\|php-fpm"; then
    echo -e "${RED}错误: 现有 Docker 环境未运行${NC}"
    echo -e "${YELLOW}请确保现有环境已启动${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 检测到现有 Docker 环境${NC}"

# 创建必要目录
echo -e "${YELLOW}创建必要目录...${NC}"
mkdir -p logs docker web

# 检查配置文件
echo -e "${YELLOW}检查配置文件...${NC}"
if [ ! -f "config.yaml" ]; then
    if [ -f "config-existing.yaml" ]; then
        cp config-existing.yaml config.yaml
        echo -e "${GREEN}✓ 使用 config-existing.yaml 作为配置文件${NC}"
    else
        echo -e "${RED}错误: 找不到配置文件${NC}"
        exit 1
    fi
fi

# 验证配置文件
if ! grep -q "your_api_key_here" config.yaml; then
    echo -e "${GREEN}✓ 配置文件已设置 API 密钥${NC}"
else
    echo -e "${YELLOW}⚠ 请编辑 config.yaml 设置您的 API 密钥${NC}"
fi

# 创建数据库初始化脚本（如果数据库未初始化）
echo -e "${YELLOW}创建数据库初始化文件...${NC}"
cat > docker/init-bitfinex-tables.sql << 'EOF'
-- BitfinexBot 集成到现有数据库的表结构
USE appdb;

-- 创建 BitfinexBot 所需的表
CREATE TABLE IF NOT EXISTS bot_realtime_status (
    id INT AUTO_INCREMENT PRIMARY KEY,
    status ENUM('running', 'stopped', 'error', 'maintenance') DEFAULT 'stopped',
    last_execution TIMESTAMP NULL,
    next_execution TIMESTAMP NULL,
    total_balance DECIMAL(20, 8) DEFAULT 0.00000000,
    available_balance DECIMAL(20, 8) DEFAULT 0.00000000,
    active_orders INT DEFAULT 0,
    current_rate DECIMAL(10, 6) DEFAULT 0.000000,
    total_earned DECIMAL(20, 8) DEFAULT 0.00000000,
    weekly_earned DECIMAL(20, 8) DEFAULT 0.00000000,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS notifications (
    id INT AUTO_INCREMENT PRIMARY KEY,
    type ENUM('rate_threshold', 'error', 'lending', 'system', 'info') NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    level ENUM('info', 'warning', 'error', 'success') DEFAULT 'info',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入默认状态
INSERT INTO bot_realtime_status (status) VALUES ('stopped')
ON DUPLICATE KEY UPDATE status=VALUES(status);
EOF

# 复制Web界面文件
echo -e "${YELLOW}复制Web界面文件...${NC}"
cp -r web/* nginx/html/bitfinex/ 2>/dev/null || true

# 创建Nginx配置
echo -e "${YELLOW}创建Nginx配置...${NC}"
cat > nginx/conf.d/bitfinex.conf << 'EOF'
location /bitfinex {
    alias /var/www/html/bitfinex;
    try_files $uri $uri/ /index.html;
}

location /bitfinex/api {
    alias /var/www/html/bitfinex/api;
    try_files $uri $uri/ =404;
    location ~ \.php$ {
        include fastcgi_params;
        fastcgi_pass php-fpm:9000;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME /var/www/html/bitfinex/api/$fastcgi_script_name;
    }
}
EOF

# 构建BitfinexBot镜像
echo -e "${YELLOW}构建BitfinexBot镜像...${NC}"
docker build -t bitfinex-bot:latest .

# 运行数据库初始化
echo -e "${YELLOW}初始化数据库表...${NC}"
docker exec mysql mysql -u appuser -papppass -D appdb < docker/init-bitfinex-tables.sql || echo "数据库已初始化或连接失败"

# 启动BitfinexBot容器
echo -e "${YELLOW}启动BitfinexBot容器...${NC}"
docker run -d \
    --name bitfinex_bot \
    --restart unless-stopped \
    --network $(docker network ls | grep -v NETWORK | head -1 | awk '{print $2}') \
    -v "$(pwd)/config.yaml:/app/config.yaml" \
    -v "$(pwd)/logs:/app/logs" \
    bitfinex-bot:latest

# 等待容器启动
echo -e "${YELLOW}等待容器启动...${NC}"
sleep 5

# 检查容器状态
if docker ps | grep -q "bitfinex_bot"; then
    echo -e "${GREEN}✓ BitfinexBot 容器启动成功${NC}"
else
    echo -e "${RED}错误: BitfinexBot 容器启动失败${NC}"
    docker logs bitfinex_bot
    exit 1
fi

# 显示部署信息
echo -e "${BLUE}"
echo "=========================================="
echo "         部署完成"
echo "=========================================="
echo -e "${NC}"

echo -e "${GREEN}🌐 Web 管理界面:${NC} http://localhost/bitfinex"
echo -e "${GREEN}📊 实时日志:${NC} docker logs -f bitfinex_bot"
echo -e "${GREEN}📁 日志目录:${NC} $(pwd)/logs"
echo ""
echo -e "${YELLOW}管理命令:${NC}"
echo -e "  查看状态: docker exec bitfinex_bot ./bitfinex-bot --status"
echo -e "  重启:    docker restart bitfinex_bot"
echo -e "  停止:    docker stop bitfinex_bot"
echo -e "  启动:    docker start bitfinex_bot"
echo ""
echo -e "${YELLOW}下一步:${NC}"
echo -e "1. 编辑 config.yaml 设置您的 API 密钥"
echo -e "2. 访问 http://localhost/bitfinex 查看Web界面"
echo -e "3. 使用 docker logs -f bitfinex_bot 查看实时日志"

echo -e "${GREEN}"
echo "🎉 BitfinexBot 集成部署完成！"
echo -e "${NC}"