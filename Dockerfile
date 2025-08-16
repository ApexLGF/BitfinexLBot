# 多阶段构建 Dockerfile for BitfinexBot Web版本
# 第一阶段：构建 Go 应用程序
FROM golang:1.23-alpine AS go-builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bitfinex-bot .

# 第二阶段：创建运行时镜像
FROM nginx:alpine

# 安装必要的包
RUN apk add --no-cache ca-certificates tzdata supervisor

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建应用目录
RUN mkdir -p /app /app/logs /var/log/supervisor

# 从构建阶段复制应用程序
COPY --from=go-builder /app/bitfinex-bot /app/bitfinex-bot

# 复制 Web 文件到 Nginx 默认目录
COPY web/ /usr/share/nginx/html/

# 创建 Nginx 配置
RUN rm /etc/nginx/conf.d/default.conf
COPY docker/nginx.conf /etc/nginx/conf.d/bitfinex.conf

# 创建 Supervisor 配置
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# 复制启动脚本
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# 设置权限
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chmod -R 755 /usr/share/nginx/html

# 暴露端口
EXPOSE 8089

# 设置健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8089/health || exit 1

# 设置启动命令
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf", "-n"]