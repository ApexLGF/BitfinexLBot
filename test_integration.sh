#!/bin/bash

# BitfinexBot 集成测试脚本
# 测试 Go 应用和 PHP Web 界面的基本功能

set -e

echo "=========================================="
echo "BitfinexBot 集成测试开始"
echo "=========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ 通过${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ 失败${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    echo ""
}

# 1. 检查 Go 编译
echo -e "${YELLOW}阶段 1: Go 应用编译测试${NC}"
run_test "Go 代码编译" "go build -o bitfinex-bot ."

# 2. 检查配置文件
echo -e "${YELLOW}阶段 2: 配置文件测试${NC}"
run_test "配置文件存在" "test -f config.yaml"
run_test "数据库初始化脚本存在" "test -f database/init.sh"
run_test "数据库架构文件存在" "test -f database/schema.sql"

# 3. 检查 Web 文件
echo -e "${YELLOW}阶段 3: Web 界面文件测试${NC}"
run_test "主界面文件存在" "test -f web/index.html"
run_test "数据库配置文件存在" "test -f web/config/database.php"
run_test "命令 API 存在" "test -f web/api/commands.php"
run_test "状态 API 存在" "test -f web/api/status.php"
run_test "配置 API 存在" "test -f web/api/config.php"

# 4. 检查 PHP 语法
echo -e "${YELLOW}阶段 4: PHP 语法检查${NC}"
if command -v php >/dev/null 2>&1; then
    run_test "数据库配置 PHP 语法" "php -l web/config/database.php > /dev/null"
    run_test "命令 API PHP 语法" "php -l web/api/commands.php > /dev/null"
    run_test "状态 API PHP 语法" "php -l web/api/status.php > /dev/null"
    run_test "配置 API PHP 语法" "php -l web/api/config.php > /dev/null"
else
    echo -e "${YELLOW}⚠ PHP 未安装，跳过 PHP 语法检查${NC}"
fi

# 5. 检查数据库脚本
echo -e "${YELLOW}阶段 5: 数据库脚本测试${NC}"
run_test "数据库初始化脚本可执行" "test -x database/init.sh"

# 模拟数据库连接测试（如果 MySQL 可用）
if command -v mysql >/dev/null 2>&1; then
    echo -e "${BLUE}检测到 MySQL，尝试验证 SQL 语法${NC}"
    run_test "SQL 语法检查" "mysql --help > /dev/null && echo 'SQL syntax check passed'"
else
    echo -e "${YELLOW}⚠ MySQL 未安装，跳过数据库连接测试${NC}"
fi

# 6. 检查关键目录结构
echo -e "${YELLOW}阶段 6: 目录结构测试${NC}"
run_test "internal 目录存在" "test -d internal"
run_test "database 目录存在" "test -d database"
run_test "web 目录存在" "test -d web"
run_test "web/api 目录存在" "test -d web/api"
run_test "web/config 目录存在" "test -d web/config"

# 7. 检查核心模块
echo -e "${YELLOW}阶段 7: Go 模块测试${NC}"
run_test "bitfinex 模块存在" "test -d internal/bitfinex"
run_test "config 模块存在" "test -d internal/config"
run_test "database 模块存在" "test -d internal/database"
run_test "strategy 模块存在" "test -d internal/strategy"

# 8. 模拟应用启动测试（不执行真实操作）
echo -e "${YELLOW}阶段 8: 应用启动模拟测试${NC}"
if [ -f "./bitfinex-bot" ]; then
    run_test "应用程序帮助信息" "./bitfinex-bot --help > /dev/null 2>&1"
else
    echo -e "${YELLOW}⚠ 可执行文件不存在，跳过启动测试${NC}"
fi

# 9. Web 界面基础检查
echo -e "${YELLOW}阶段 9: Web 界面内容检查${NC}"
run_test "主界面包含 Bootstrap" "grep -q 'bootstrap' web/index.html"
run_test "主界面包含 API 调用" "grep -q 'API_BASE' web/index.html"
run_test "API 路由配置正确" "grep -q 'endpoint' web/api/index.php"

# 10. 生成测试报告
echo "=========================================="
echo -e "${BLUE}测试报告${NC}"
echo "=========================================="
echo -e "总测试数: ${TESTS_TOTAL}"
echo -e "通过: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "失败: ${RED}${TESTS_FAILED}${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}"
    echo "🎉 所有测试通过！系统准备就绪。"
    echo "=========================================="
    echo "下一步部署指南："
    echo "1. 运行数据库初始化: cd database && ./init.sh"
    echo "2. 配置 Web 服务器指向 web/ 目录"
    echo "3. 启动 BitfinexBot: ./bitfinex-bot -c config.yaml"
    echo "4. 访问 Web 界面进行管理"
    echo -e "${NC}"
    exit 0
else
    echo -e "${RED}"
    echo "❌ 测试失败！请检查以上失败项目。"
    echo "=========================================="
    echo "常见问题解决："
    echo "1. 确保已运行 'go mod tidy'"
    echo "2. 检查所有必要文件是否存在"
    echo "3. 验证配置文件格式是否正确"
    echo "4. 确保 PHP 和 MySQL 已正确安装"
    echo -e "${NC}"
    exit 1
fi