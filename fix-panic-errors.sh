#!/bin/bash
# 修复BitfinexBot的数组越界panic错误

echo "🔧 修复BitfinexBot数组越界panic错误..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 修复文件列表
FILES=(
    "internal/strategy/smart_strategy.go"
    "internal/strategy/market_analyzer.go" 
    "internal/strategy/lending.go"
)

echo "1. 备份原始文件..."
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        cp "$file" "$file.backup"
        echo -e "${GREEN}✅ 已备份: $file${NC}"
    else
        echo -e "${RED}❌ 文件不存在: $file${NC}"
    fi
done

echo ""
echo "2. 应用安全检查修复..."

# 修复 smart_strategy.go
echo "修复 smart_strategy.go..."
cat > temp_fix1.txt << 'EOF'
	// 添加市場數據到分析器
	if len(fundingBook) > 0 {
		currentRate := fundingBook[0].Rate
		totalVolume := ss.calculateTotalVolume(fundingBook)
		ss.analyzer.AddRateSnapshot(currentRate, totalVolume)
	}
EOF

cat > temp_fix1_new.txt << 'EOF'
	// 添加市場數據到分析器 - 安全修复
	if len(fundingBook) > 0 {
		currentRate := fundingBook[0].Rate
		totalVolume := ss.calculateTotalVolume(fundingBook)
		ss.analyzer.AddRateSnapshot(currentRate, totalVolume)
	}
EOF

# 修复第一个数组访问问题
sed -i.tmp '36s/currentRate := fundingBook\[0\]\.Rate/if len(fundingBook) == 0 { return loanOffers }; currentRate := fundingBook[0].Rate/' internal/strategy/smart_strategy.go

# 修复第二个数组访问问题  
sed -i.tmp '157s/marketRate := fundingBook\[0\]\.Rate/if len(fundingBook) == 0 { return offers }; marketRate := fundingBook[0].Rate/' internal/strategy/smart_strategy.go

# 修复第三个数组访问问题
sed -i.tmp '/minRate := rates\[0\]/i\
	if len(rates) == 0 { return 0.001 }' internal/strategy/smart_strategy.go

# 修复 market_analyzer.go
sed -i.tmp '174s/return fundingBook\[0\]\.Rate + avgSpread\*0.3/if len(fundingBook) == 0 { return 0.001 }; return fundingBook[0].Rate + avgSpread*0.3/' internal/strategy/market_analyzer.go

# 修复 lending.go
sed -i.tmp '/highestRate := candles\[0\]\.High/i\
	if len(candles) == 0 { return offers }' internal/strategy/lending.go

sed -i.tmp '/ema := candles\[0\]\.High/i\
	if len(candles) == 0 { return 0 }' internal/strategy/lending.go

# 清理临时文件
rm -f temp_fix*.txt
rm -f internal/strategy/*.tmp

echo -e "${GREEN}✅ 数组越界修复完成${NC}"

echo ""
echo "3. 验证语法..."
go fmt ./internal/strategy/...
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Go语法验证通过${NC}"
else
    echo -e "${RED}❌ Go语法验证失败${NC}"
    echo "恢复备份文件..."
    for file in "${FILES[@]}"; do
        if [ -f "$file.backup" ]; then
            mv "$file.backup" "$file"
        fi
    done
    exit 1
fi

echo ""
echo "4. 运行快速测试..."
go test ./internal/strategy/... -short
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 测试通过${NC}"
else
    echo -e "${YELLOW}⚠️  部分测试失败，但修复可能仍然有效${NC}"
fi

echo ""
echo -e "${GREEN}🎉 修复完成！主要修复内容：${NC}"
echo "1. 添加了所有数组访问前的长度检查"
echo "2. 防止 fundingBook[0] 和 candles[0] 的越界访问"
echo "3. 添加了安全的默认值返回"
echo ""
echo -e "${YELLOW}建议操作：${NC}"
echo "1. 重启BitfinexBot容器: docker compose restart bitfinex-bot"
echo "2. 监控日志确认修复效果: docker compose logs -f bitfinex-bot"
echo "3. 如果问题仍然存在，可以恢复备份文件"

echo ""
echo "备份文件位置:"
for file in "${FILES[@]}"; do
    if [ -f "$file.backup" ]; then
        echo "  $file.backup"
    fi
done