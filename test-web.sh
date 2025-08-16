#!/bin/bash

# BitfinexBot Web版本测试脚本

echo "🚀 BitfinexBot Web版本集成测试"
echo "================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试结果
TESTS_PASSED=0
TESTS_TOTAL=0

# 测试函数
test_step() {
    local name="$1"
    local command="$2"
    local expected_exit_code="${3:-0}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -e "\n${BLUE}[测试 $TESTS_TOTAL]${NC} $name"
    echo "命令: $command"
    
    if eval "$command"; then
        if [ $? -eq $expected_exit_code ]; then
            echo -e "${GREEN}✅ 通过${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            echo -e "${RED}❌ 失败 (退出码不匹配)${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ 失败${NC}"
        return 1
    fi
}

# 检查环境
echo -e "\n${YELLOW}🔍 检查环境...${NC}"
test_step "Go环境检查" "go version"
test_step "Docker环境检查" "docker --version"

# 代码测试
echo -e "\n${YELLOW}📝 代码测试...${NC}"
test_step "Go模块整理" "go mod tidy"
test_step "代码构建测试" "go build -o bitfinex-bot-test ."
test_step "代码语法检查" "go vet ./..."

# 配置文件测试
echo -e "\n${YELLOW}⚙️ 配置文件测试...${NC}"
test_step "创建测试配置" "cp config.yaml.example config.test.yaml"
test_step "配置文件格式验证" "./bitfinex-bot-test -c config.test.yaml --help > /dev/null 2>&1"

# Web文件测试
echo -e "\n${YELLOW}🌐 Web文件测试...${NC}"
test_step "Web文件结构检查" "test -f web/index.html && test -d web/static"
test_step "CSS文件检查" "test -f web/static/css/app.css"
test_step "JS文件检查" "test -f web/static/js/api.js && test -f web/static/js/app.js && test -f web/static/js/charts.js"

# Docker测试
echo -e "\n${YELLOW}🐳 Docker测试...${NC}"
test_step "Docker镜像构建" "docker build -t bitfinex-bot-web-test . > /dev/null 2>&1"
test_step "Docker配置文件检查" "test -f docker/nginx.conf && test -f docker/supervisord.conf && test -f docker/entrypoint.sh"

# API端点测试（模拟）
echo -e "\n${YELLOW}🔌 API接口设计验证...${NC}"
API_HANDLERS=(
    "GetStatus"
    "GetEarnings"
    "GetOffers" 
    "GetConfig"
    "GetLogs"
    "GetSystemInfo"
    "Control"
)

for handler in "${API_HANDLERS[@]}"; do
    test_step "API处理器验证: $handler" "grep -r \"func.*$handler\" internal/api/ > /dev/null"
done

test_step "API路由组验证" "grep -r 'api := ' internal/api/ > /dev/null"
test_step "健康检查端点验证" "grep -r '/health' internal/api/ > /dev/null"

# 清理测试文件
echo -e "\n${YELLOW}🧹 清理测试文件...${NC}"
test_step "清理测试文件" "rm -f bitfinex-bot-test config.test.yaml"

# 测试结果汇总
echo -e "\n${YELLOW}📊 测试结果汇总${NC}"
echo "================================"
echo -e "总测试数: ${BLUE}$TESTS_TOTAL${NC}"
echo -e "通过测试: ${GREEN}$TESTS_PASSED${NC}"
echo -e "失败测试: ${RED}$((TESTS_TOTAL - TESTS_PASSED))${NC}"

if [ $TESTS_PASSED -eq $TESTS_TOTAL ]; then
    echo -e "\n${GREEN}🎉 所有测试通过！BitfinexBot Web版本准备就绪！${NC}"
    echo -e "\n${BLUE}📋 下一步操作:${NC}"
    echo "1. 编辑 config.yaml 设置您的 Bitfinex API 密钥"
    echo "2. 运行: docker compose up -d"
    echo "3. 访问: http://localhost:8089"
    exit 0
else
    echo -e "\n${RED}❌ 测试失败，请检查上述错误信息${NC}"
    exit 1
fi