#!/bin/bash

# BitfinexBot 自动化部署脚本
# 使用方法: ./deploy.sh [version]
# 示例: ./deploy.sh v2.3.0 或 ./deploy.sh latest

set -e  # 遇到错误立即退出

VERSION=${1:-v2.3.0}
IMAGE="apexlgf/bitfinexwebbot:$VERSION"
CONTAINER_NAME="bitfinex-bot"
PORT="8089"

echo "=========================================="
echo "BitfinexBot 自动化部署脚本"
echo "=========================================="
echo "镜像: $IMAGE"
echo "容器名称: $CONTAINER_NAME"
echo "端口: $PORT"
echo "=========================================="
echo ""

# 检查 config.yaml 是否存在
if [ ! -f "config.yaml" ]; then
    echo "❌ 错误: config.yaml 文件不存在！"
    echo "请先创建配置文件，或从 config.yaml.example 复制"
    exit 1
fi

# 拉取最新镜像
echo "📥 [1/5] 拉取镜像..."
if docker pull $IMAGE; then
    echo "✅ 镜像拉取成功"
else
    echo "❌ 镜像拉取失败"
    exit 1
fi
echo ""

# 停止旧容器
echo "🛑 [2/5] 停止旧容器..."
if docker stop $CONTAINER_NAME 2>/dev/null; then
    echo "✅ 旧容器已停止"
else
    echo "ℹ️  没有运行中的容器"
fi
echo ""

# 删除旧容器
echo "🗑️  [3/5] 删除旧容器..."
if docker rm $CONTAINER_NAME 2>/dev/null; then
    echo "✅ 旧容器已删除"
else
    echo "ℹ️  没有需要删除的容器"
fi
echo ""

# 启动新容器
echo "🚀 [4/5] 启动新容器..."
docker run -d \
  --name $CONTAINER_NAME \
  -p $PORT:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  $IMAGE

if [ $? -eq 0 ]; then
    echo "✅ 容器启动成功"
else
    echo "❌ 容器启动失败"
    exit 1
fi
echo ""

# 等待容器启动
echo "⏳ [5/5] 等待容器启动..."
sleep 5

# 检查容器状态
if docker ps | grep -q $CONTAINER_NAME; then
    echo "✅ 容器运行正常"
    echo ""
    echo "=========================================="
    echo "🎉 部署成功！"
    echo "=========================================="
    echo "📊 容器信息:"
    docker ps | grep $CONTAINER_NAME
    echo ""
    echo "🌐 Web 界面: http://localhost:$PORT"
    echo "📝 查看日志: docker logs -f $CONTAINER_NAME"
    echo "🛑 停止容器: docker stop $CONTAINER_NAME"
    echo "=========================================="
    echo ""
    echo "📋 最近日志:"
    docker logs --tail 20 $CONTAINER_NAME
else
    echo "❌ 容器启动失败！"
    echo ""
    echo "错误日志:"
    docker logs $CONTAINER_NAME
    exit 1
fi
