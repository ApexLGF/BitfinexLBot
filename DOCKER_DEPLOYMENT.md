# Docker 镜像部署指南

## 镜像信息

- **仓库**: apexlgf/bitfinexwebbot
- **最新版本**: v2.3.0
- **标签**:
  - `latest` - 最新版本（可能有缓存问题）
  - `v2.3.0` - 稳定版本（推荐使用）

## 解决 latest 标签缓存问题

### 问题说明

使用 `latest` 标签时，Docker 可能会使用本地缓存的旧镜像，导致无法更新到最新版本。

### 解决方案

#### 方案1：使用版本标签（推荐）

始终使用具体的版本标签，而不是 `latest`：

```bash
# 拉取指定版本
docker pull apexlgf/bitfinexwebbot:v2.3.0

# 运行容器
docker run -d \
  --name bitfinex-bot \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  apexlgf/bitfinexwebbot:v2.3.0
```

#### 方案2：强制拉取最新镜像

如果必须使用 `latest` 标签，使用 `--pull always` 强制拉取：

```bash
# 停止并删除旧容器
docker stop bitfinex-bot
docker rm bitfinex-bot

# 删除本地镜像
docker rmi apexlgf/bitfinexwebbot:latest

# 强制拉取最新镜像
docker pull apexlgf/bitfinexwebbot:latest

# 运行新容器
docker run -d \
  --name bitfinex-bot \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  apexlgf/bitfinexwebbot:latest
```

#### 方案3：使用 Docker Compose（推荐生产环境）

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  bitfinex-bot:
    image: apexlgf/bitfinexwebbot:v2.3.0  # 使用具体版本
    container_name: bitfinex-bot
    ports:
      - "8089:8089"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - bitfinex_logs:/app/logs
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8089/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

volumes:
  bitfinex_logs:
```

更新部署：

```bash
# 拉取最新镜像
docker compose pull

# 重新创建容器
docker compose up -d --force-recreate
```

## 版本更新流程

### 1. 构建新版本

```bash
# 构建新版本镜像
docker build -t apexlgf/bitfinexwebbot:v2.3.1 -t apexlgf/bitfinexwebbot:latest .

# 推送到 Docker Hub
docker push apexlgf/bitfinexwebbot:v2.3.1
docker push apexlgf/bitfinexwebbot:latest
```

### 2. 部署新版本

```bash
# 方法1：使用版本标签
docker pull apexlgf/bitfinexwebbot:v2.3.1
docker stop bitfinex-bot
docker rm bitfinex-bot
docker run -d \
  --name bitfinex-bot \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  apexlgf/bitfinexwebbot:v2.3.1

# 方法2：使用 Docker Compose
# 更新 docker-compose.yml 中的版本号
docker compose pull
docker compose up -d --force-recreate
```

## 验证部署

### 1. 检查容器状态

```bash
# 查看容器运行状态
docker ps | grep bitfinex-bot

# 查看容器日志
docker logs -f bitfinex-bot

# 检查健康状态
docker inspect bitfinex-bot | grep -A 10 Health
```

### 2. 访问 Web 界面

打开浏览器访问：http://localhost:8089

### 3. 验证镜像版本

```bash
# 查看本地镜像
docker images | grep bitfinexwebbot

# 查看容器使用的镜像
docker inspect bitfinex-bot | grep Image
```

## 常见问题

### Q1: 为什么更新后还是旧版本？

**A**: Docker 使用了本地缓存的镜像。解决方法：
1. 删除本地镜像：`docker rmi apexlgf/bitfinexwebbot:latest`
2. 重新拉取：`docker pull apexlgf/bitfinexwebbot:latest`
3. 或者使用具体版本标签

### Q2: 如何回滚到旧版本？

**A**: 使用具体的版本标签：
```bash
docker stop bitfinex-bot
docker rm bitfinex-bot
docker run -d \
  --name bitfinex-bot \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  apexlgf/bitfinexwebbot:v2.2.0  # 使用旧版本
```

### Q3: 如何查看所有可用版本？

**A**: 访问 Docker Hub 页面：
https://hub.docker.com/r/apexlgf/bitfinexwebbot/tags

或使用命令：
```bash
# 需要安装 jq
curl -s https://registry.hub.docker.com/v2/repositories/apexlgf/bitfinexwebbot/tags/ | jq -r '.results[].name'
```

## 最佳实践

1. **生产环境使用版本标签**
   - ✅ `apexlgf/bitfinexwebbot:v2.3.0`
   - ❌ `apexlgf/bitfinexwebbot:latest`

2. **使用 Docker Compose 管理**
   - 便于版本控制
   - 支持一键更新
   - 配置文件化

3. **定期备份配置**
   ```bash
   cp config.yaml config.yaml.backup.$(date +%Y%m%d)
   ```

4. **监控容器健康状态**
   ```bash
   docker inspect bitfinex-bot | grep -A 10 Health
   ```

5. **保留日志**
   ```bash
   docker logs bitfinex-bot > logs/bitfinex-bot-$(date +%Y%m%d).log
   ```

## 自动化部署脚本

创建 `deploy.sh`：

```bash
#!/bin/bash

VERSION=${1:-latest}
IMAGE="apexlgf/bitfinexwebbot:$VERSION"

echo "部署 BitfinexBot $VERSION..."

# 拉取最新镜像
echo "拉取镜像..."
docker pull $IMAGE

# 停止旧容器
echo "停止旧容器..."
docker stop bitfinex-bot 2>/dev/null || true
docker rm bitfinex-bot 2>/dev/null || true

# 启动新容器
echo "启动新容器..."
docker run -d \
  --name bitfinex-bot \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  $IMAGE

# 等待容器启动
echo "等待容器启动..."
sleep 5

# 检查状态
if docker ps | grep -q bitfinex-bot; then
  echo "✅ 部署成功！"
  echo "访问 Web 界面: http://localhost:8089"
  docker logs --tail 20 bitfinex-bot
else
  echo "❌ 部署失败！"
  docker logs bitfinex-bot
  exit 1
fi
```

使用方法：

```bash
# 部署最新版本
./deploy.sh latest

# 部署指定版本
./deploy.sh v2.3.0
```

## 镜像清理

定期清理未使用的镜像：

```bash
# 清理悬空镜像
docker image prune -f

# 清理所有未使用的镜像
docker image prune -a -f

# 清理特定镜像的旧版本
docker images | grep bitfinexwebbot | grep -v v2.3.0 | awk '{print $3}' | xargs docker rmi
```

## 支持

如有问题，请查看：
- GitHub Issues: https://github.com/ApexLGF/BitfinexLBot/issues
- Docker Hub: https://hub.docker.com/r/apexlgf/bitfinexwebbot
