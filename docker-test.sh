#!/bin/bash

# BitfinexBot Docker 环境测试脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "=========================================="
echo "   BitfinexBot Docker 环境测试"
echo "=========================================="
echo -e "${NC}"

# 测试结果统计
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# 测试函数
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -e "${BLUE}测试 $TESTS_TOTAL: $test_name${NC}"
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 通过${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ 失败${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        # 显示错误详情
        echo -e "${YELLOW}错误详情:${NC}"
        eval "$test_command" 2>&1 | head -3
    fi
    echo ""
}

# 1. 检查 Docker 服务
echo -e "${YELLOW}阶段 1: Docker 服务检查${NC}"
run_test "Docker 服务运行" "docker info"
# 检查 Docker Compose（支持新旧版本）
DOCKER_COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
else
    DOCKER_COMPOSE_CMD="docker-compose"  # 回退到旧版本
fi

run_test "Docker Compose 可用" "$DOCKER_COMPOSE_CMD --version"

# 2. 检查容器状态
echo -e "${YELLOW}阶段 2: 容器状态检查${NC}"
run_test "MySQL 容器运行" "$DOCKER_COMPOSE_CMD ps mysql | grep -q Up"
run_test "PHP 容器运行" "$DOCKER_COMPOSE_CMD ps php | grep -q Up"
run_test "Web 容器运行" "$DOCKER_COMPOSE_CMD ps web | grep -q Up"
run_test "phpMyAdmin 容器运行" "$DOCKER_COMPOSE_CMD ps phpmyadmin | grep -q Up"

# 3. 数据库连接测试
echo -e "${YELLOW}阶段 3: 数据库连接测试${NC}"
run_test "MySQL 服务响应" "$DOCKER_COMPOSE_CMD exec mysql mysqladmin ping -h localhost -u bitfinex_bot -pBitfinexBot@2025 --silent"
run_test "数据库存在" "$DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -e 'USE bitfinex_bot_db;'"

# 4. 数据库表结构测试
echo -e "${YELLOW}阶段 4: 数据库表结构测试${NC}"
run_test "用户表存在" "$DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -e 'DESCRIBE users;'"
run_test "命令表存在" "$DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -e 'DESCRIBE web_commands;'"
run_test "状态表存在" "$DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -e 'DESCRIBE bot_realtime_status;'"
run_test "通知表存在" "$DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -e 'DESCRIBE notifications;'"

# 5. Web 服务测试
echo -e "${YELLOW}阶段 5: Web 服务测试${NC}"
run_test "Web 首页访问" "curl -f http://localhost:8080"
run_test "API 路由响应" "curl -f http://localhost:8080/api/"
run_test "状态 API 响应" "curl -f http://localhost:8080/api/index.php?endpoint=status&action=bot"
run_test "phpMyAdmin 访问" "curl -f http://localhost:8081"

# 6. PHP 功能测试
echo -e "${YELLOW}阶段 6: PHP 功能测试${NC}"
run_test "PHP-FPM 进程运行" "$DOCKER_COMPOSE_CMD exec php pgrep php-fpm"
run_test "PDO MySQL 扩展" "$DOCKER_COMPOSE_CMD exec php php -m | grep -q pdo_mysql"

# 7. API 功能测试
echo -e "${YELLOW}阶段 7: API 功能测试${NC}"

# 测试命令提交
SUBMIT_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"action":"submit","command_type":"status"}' \
    http://localhost:8080/api/index.php?endpoint=commands)

run_test "命令提交 API" "echo '$SUBMIT_RESPONSE' | grep -q '\"success\":true'"

# 测试命令列表
run_test "命令列表 API" "curl -s 'http://localhost:8080/api/index.php?endpoint=commands&action=list' | grep -q '\"success\":true'"

# 测试配置 API
run_test "配置列表 API" "curl -s 'http://localhost:8080/api/index.php?endpoint=config&action=list' | grep -q '\"success\":true'"

# 8. 日志和文件权限测试
echo -e "${YELLOW}阶段 8: 文件和权限测试${NC}"
run_test "日志目录存在" "test -d logs"
run_test "配置文件可读" "test -r config.yaml"
run_test "Web 文件可读" "test -r web/index.html"

# 9. 网络连接测试
echo -e "${YELLOW}阶段 9: 网络连接测试${NC}"
run_test "容器间 MySQL 连接" "$DOCKER_COMPOSE_CMD exec php nc -z mysql 3306"
run_test "容器间 PHP 连接" "$DOCKER_COMPOSE_CMD exec web nc -z php 9000"

# 10. 数据完整性测试
echo -e "${YELLOW}阶段 10: 数据完整性测试${NC}"

# 检查默认数据
DEFAULT_USER_COUNT=$($DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -se "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
run_test "默认用户数据存在" "[ '$DEFAULT_USER_COUNT' -gt 0 ]"

DEFAULT_STATUS_COUNT=$($DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -se "SELECT COUNT(*) FROM bot_realtime_status;" 2>/dev/null || echo "0")
run_test "默认状态数据存在" "[ '$DEFAULT_STATUS_COUNT' -gt 0 ]"

CONFIG_COUNT=$($DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -se "SELECT COUNT(*) FROM system_config;" 2>/dev/null || echo "0")
run_test "默认配置数据存在" "[ '$CONFIG_COUNT' -gt 0 ]"

# 生成测试报告
echo -e "${BLUE}"
echo "=========================================="
echo "           测试报告"
echo "=========================================="
echo -e "${NC}"

echo -e "总测试数: ${TESTS_TOTAL}"
echo -e "通过: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "失败: ${RED}${TESTS_FAILED}${NC}"
echo -e "成功率: $(( TESTS_PASSED * 100 / TESTS_TOTAL ))%"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}"
    echo "🎉 所有测试通过！Docker 环境运行正常。"
    echo ""
    echo "=========================================="
    echo "           环境信息"
    echo "=========================================="
    
    # 显示容器状态
    echo -e "${BLUE}容器状态:${NC}"
    $DOCKER_COMPOSE_CMD ps
    
    echo ""
    echo -e "${BLUE}数据库表信息:${NC}"
    $DOCKER_COMPOSE_CMD exec mysql mysql -u bitfinex_bot -pBitfinexBot@2025 -D bitfinex_bot_db -e "
        SELECT 
            TABLE_NAME as '表名',
            TABLE_ROWS as '记录数',
            ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024), 2) as '大小(KB)'
        FROM information_schema.TABLES 
        WHERE TABLE_SCHEMA = 'bitfinex_bot_db'
        ORDER BY TABLE_NAME;
    " 2>/dev/null || echo "无法获取表信息"
    
    echo ""
    echo -e "${BLUE}资源使用情况:${NC}"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null || echo "无法获取资源信息"
    
    echo -e "${GREEN}"
    echo "=========================================="
    echo "         下一步操作"
    echo "=========================================="
    echo ""
    echo "1. 编辑 config.yaml 设置 Bitfinex API 密钥"
    echo "2. 启动 BitfinexBot: $DOCKER_COMPOSE_CMD up -d bitfinex-bot"
    echo "3. 访问管理界面: http://localhost:8080"
    echo "4. 访问数据库管理: http://localhost:8081"
    echo -e "${NC}"
    
    exit 0
else
    echo -e "${RED}"
    echo "❌ 部分测试失败！请检查以上失败项目。"
    echo ""
    echo "=========================================="
    echo "         故障排查建议"
    echo "=========================================="
    echo ""
    echo "1. 检查容器日志: $DOCKER_COMPOSE_CMD logs [service_name]"
    echo "2. 重新构建: $DOCKER_COMPOSE_CMD build --no-cache"
    echo "3. 完全重置: $DOCKER_COMPOSE_CMD down -v && $DOCKER_COMPOSE_CMD up -d"
    echo "4. 检查端口占用: netstat -tulpn | grep :8080"
    echo "5. 查看系统资源: docker system df"
    echo -e "${NC}"
    exit 1
fi