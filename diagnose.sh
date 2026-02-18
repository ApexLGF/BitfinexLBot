#!/bin/bash

# BitfinexBot 诊断脚本
# 用于远程环境故障排查

echo "=========================================="
echo "BitfinexBot 诊断脚本"
echo "=========================================="
echo ""

# 1. 检查容器状态
echo "1. 检查容器状态..."
docker ps -a | grep bitfinex-bot

echo ""
echo "2. 检查最近的容器日志..."
docker logs bitfinex-bot-web --tail 100

echo ""
echo "3. 检查 supervisor 错误日志..."
docker exec bitfinex-bot-web cat /var/log/supervisor/bitfinex-bot-stderr.log 2>/dev/null || echo "无法读取 stderr 日志"

echo ""
echo "4. 检查 supervisor 标准输出日志..."
docker exec bitfinex-bot-web cat /var/log/supervisor/bitfinex-bot-stdout.log 2>/dev/null || echo "无法读取 stdout 日志"

echo ""
echo "5. 手动测试 bot 启动..."
docker exec bitfinex-bot-web /app/bitfinex-bot -c /app/config.yaml 2>&1 | head -30

echo ""
echo "6. 检查配置文件..."
docker exec bitfinex-bot-web cat /app/config.yaml | head -20

echo ""
echo "7. 检查文件权限..."
docker exec bitfinex-bot-web ls -la /app/

echo ""
echo "=========================================="
echo "诊断完成"
echo "=========================================="
