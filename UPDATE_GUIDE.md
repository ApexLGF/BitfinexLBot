# 快速更新指南

## ✅ 已完成

1. ✅ Docker 镜像已构建并推送到 Docker Hub
   - 镜像名称: `apexlgf/bitfinexwebbot`
   - 最新版本: `v2.3.2`
   - 版本标签: `v2.3.0`, `v2.3.1`, `v2.3.2`, `latest`
   - 镜像大小: ~50-60 MB

2. ✅ 创建了部署文档和脚本
   - `DOCKER_DEPLOYMENT.md` - 完整部署指南
   - `deploy.sh` - 自动化部署脚本
   - `docker-compose.yml` - 已更新使用 Docker Hub 镜像

## 🚀 如何在部署环境更新到最新版本

### 方法1：使用自动化脚本（推荐）

```bash
# 下载部署脚本
curl -O https://raw.githubusercontent.com/ApexLGF/BitfinexLBot/main/deploy.sh
chmod +x deploy.sh

# 部署最新版本
./deploy.sh v2.3.2
```

### 方法2：使用 Docker Compose

```bash
# 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.3.2

# 拉取最新镜像并重新创建容器
docker compose pull
docker compose up -d --force-recreate
```

### 方法3：手动更新

```bash
# 1. 停止并删除旧容器
docker stop bitfinex-bot-web
docker rm bitfinex-bot-web

# 2. 删除旧镜像（强制更新）
docker rmi apexlgf/bitfinexwebbot:latest

# 3. 拉取新镜像
docker pull apexlgf/bitfinexwebbot:v2.3.2

# 4. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.3.2
```

## 🔍 验证更新

```bash
# 检查容器状态
docker ps | grep bitfinex-bot

# 查看日志
docker logs -f bitfinex-bot-web

# 访问 Web 界面
open http://localhost:8089
```

## 📝 版本说明

### v2.3.2 新特性 (2026-02-17)

**关键修复**：
- 🐛 修复 Docker 镜像缓存问题，确保包含最新 Web 文件
- ✅ 使用 `--no-cache` 重新构建，解决 v2.3.1 镜像未更新问题
- ✅ 验证镜像包含正确的版本号（v=230）

**更新建议**：
- 如果从 v2.3.1 更新，请删除旧镜像后重新拉取
- 更新后强制刷新浏览器缓存（Ctrl+Shift+R）

### v2.3.0 新特性

1. **多币种同时放贷支持**
   - 支持同时管理 USD、UST 等多个币种
   - 每个币种独立配置和策略
   - Web 界面支持币种选择器

2. **配置自动迁移**
   - 旧的单币种配置自动迁移到新格式
   - 完全向后兼容

3. **Web 界面改进**
   - 添加币种选择下拉菜单
   - 支持查看所有币种汇总或单个币种详情
   - 表格动态显示/隐藏币种列

4. **API 增强**
   - 支持 `?currency=USD` 查询参数
   - 返回多币种汇总数据

## ⚠️ 重要提示

### 关于 latest 标签

**不推荐在生产环境使用 `latest` 标签**，原因：
1. Docker 会缓存 `latest` 标签，导致无法更新
2. 无法明确知道运行的是哪个版本
3. 回滚困难

**推荐做法**：
- ✅ 使用具体版本标签：`apexlgf/bitfinexwebbot:v2.3.2`
- ❌ 避免使用：`apexlgf/bitfinexwebbot:latest`

### 配置文件

确保你的 `config.yaml` 使用新的多币种格式：

```yaml
# 多币种配置
CURRENCIES:
  USD:
    ENABLED: true
    MIN_LOAN: 10
    MIN_DAILY_LEND_RATE: 0.038
    # ... 其他配置

  UST:
    ENABLED: true
    MIN_LOAN: 1
    MIN_DAILY_LEND_RATE: 0.036
    # ... 其他配置
```

旧格式会自动迁移，但建议手动更新为新格式。

## 🆘 故障排除

### 问题1: 更新后还是旧版本

```bash
# 删除本地镜像缓存
docker rmi apexlgf/bitfinexwebbot:latest
docker rmi apexlgf/bitfinexwebbot:v2.3.2

# 重新拉取
docker pull apexlgf/bitfinexwebbot:v2.3.2
```

### 问题2: 容器无法启动

```bash
# 查看详细日志
docker logs bitfinex-bot-web

# 检查配置文件
cat config.yaml

# 验证配置文件格式
docker run --rm -v $(pwd)/config.yaml:/app/config.yaml:ro \
  apexlgf/bitfinexwebbot:v2.3.2 \
  /app/bitfinex-bot -c /app/config.yaml --validate
```

### 问题3: 端口冲突

```bash
# 检查端口占用
lsof -i :8089

# 使用不同端口
docker run -d \
  --name bitfinex-bot-web \
  -p 8090:8089 \  # 改为 8090
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  apexlgf/bitfinexwebbot:v2.3.2
```

## 📚 更多信息

- 完整部署指南: [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md)
- 多币种实现文档: [MULTI_CURRENCY_IMPLEMENTATION.md](MULTI_CURRENCY_IMPLEMENTATION.md)
- Docker Hub: https://hub.docker.com/r/apexlgf/bitfinexwebbot
- GitHub: https://github.com/ApexLGF/BitfinexLBot

## 🎯 下一步

1. 在部署环境执行更新命令
2. 验证 Web 界面正常访问
3. 检查多币种功能是否正常
4. 监控日志确保无错误

祝部署顺利！🚀
