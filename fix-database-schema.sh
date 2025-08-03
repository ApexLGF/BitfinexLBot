#!/bin/bash
# 修复BitfinexBot数据库schema问题

echo "🔧 修复BitfinexBot数据库表结构..."

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

# 2. 执行数据库schema修复
echo "2. 修复command_type字段..."
docker compose exec mysql mysql -u bitfinex_bot -p'BitfinexBot@2025' bitfinex_bot_db <<EOF
-- 修改 command_type 字段添加缺失的枚举值
ALTER TABLE web_commands MODIFY COLUMN command_type ENUM(
    'start', 'stop', 'restart', 'status', 
    'balance', 'rates', 'orders', 'config',
    'cancel_all', 'lending_check', 'rate_check',
    'set_config', 'get_config',
    'credits', 'wallets', 'daily_earnings'
) NOT NULL;

-- 验证修改结果
SHOW COLUMNS FROM web_commands LIKE 'command_type';
EOF

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 数据库schema修复成功${NC}"
else
    echo -e "${RED}❌ 数据库schema修复失败${NC}"
    exit 1
fi

# 3. 测试插入新的command_type值
echo "3. 测试新的command_type值..."
docker compose exec mysql mysql -u bitfinex_bot -p'BitfinexBot@2025' bitfinex_bot_db -e "
INSERT INTO web_commands (user_id, command_type, command_data) 
VALUES (1, 'credits', '{\"test\": true}');
SELECT 'Test insert successful' as result;
DELETE FROM web_commands WHERE command_type = 'credits' AND command_data = '{\"test\": true}';
" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 新command_type值测试成功${NC}"
else
    echo -e "${YELLOW}⚠️  新command_type值测试失败，但基本修复可能已完成${NC}"
fi

echo -e "${GREEN}🎉 数据库schema修复完成！${NC}"
echo ""
echo -e "${YELLOW}现在可以重新测试Web界面的借贷订单查询功能${NC}"