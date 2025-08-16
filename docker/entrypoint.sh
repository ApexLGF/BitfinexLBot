#!/bin/sh
set -e

echo "🚀 Starting BitfinexBot Web Application..."

# 检查配置文件
if [ ! -f /app/config.yaml ]; then
    echo "⚠️  配置文件不存在: /app/config.yaml"
    echo "📋 请通过 Docker volume 挂载配置文件到 /app/config.yaml"
    echo "示例命令："
    echo "docker run -v $(pwd)/config.yaml:/app/config.yaml -p 8089:8089 bitfinex-bot"
    exit 1
fi

echo "✅ 配置文件找到: /app/config.yaml"

# 验证配置文件格式
if ! /app/bitfinex-bot -c /app/config.yaml --help > /dev/null 2>&1; then
    echo "❌ 配置文件格式验证失败"
    echo "请检查配置文件语法"
    exit 1
fi

echo "✅ 配置文件格式验证通过"

# 确保日志目录存在
mkdir -p /app/logs /var/log/supervisor /var/log/nginx

# 设置正确的权限
chown -R nginx:nginx /usr/share/nginx/html
chmod -R 755 /usr/share/nginx/html

echo "📊 Web界面访问地址: http://localhost:8089"
echo "🔧 API接口地址: http://localhost:8089/api/"
echo "❤️  健康检查地址: http://localhost:8089/health"

# 显示配置信息
echo "📋 当前配置："
echo "  - 配置文件: /app/config.yaml"
echo "  - 日志目录: /app/logs"
echo "  - Web端口: 8089"

echo "🎉 启动完成，开始运行服务..."

# 执行传入的命令
exec "$@"