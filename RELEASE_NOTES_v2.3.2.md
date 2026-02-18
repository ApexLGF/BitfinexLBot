# BitfinexBot v2.3.2 发布说明

## 发布日期
2026-02-17

## 版本信息
- **版本号**: v2.3.2
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.3.2`
- **镜像大小**: ~50-60 MB

## 修复内容

### 🐛 关键修复

**修复 Docker 镜像缓存问题**
- **问题描述**: v2.3.1 镜像由于 Docker 构建缓存，未能正确包含更新后的 Web 文件（index.html 仍显示 v=10）
- **解决方案**: 使用 `--no-cache` 标志重新构建镜像，确保所有文件都是最新版本
- **影响**: 部署环境现在可以正确显示币种选择器和多币种功能

### ✅ 验证结果

已验证镜像包含正确的文件版本：
```bash
# 验证命令
docker run --rm --entrypoint cat apexlgf/bitfinexwebbot:v2.3.2 /usr/share/nginx/html/index.html | tail -10

# 输出确认
<script src="static/js/api.js?v=230"></script>
<script src="static/js/charts.js?v=230"></script>
<script src="static/js/app.js?v=230"></script>
```

## 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.3.2

# 2. 拉取新镜像并重新创建容器
docker compose pull
docker compose up -d --force-recreate
```

### 方法 2：手动更新

```bash
# 1. 停止并删除旧容器
docker stop bitfinex-bot-web
docker rm bitfinex-bot-web

# 2. 删除旧镜像（可选，确保使用新镜像）
docker rmi apexlgf/bitfinexwebbot:v2.3.1
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

### 方法 3：使用部署脚本

```bash
./deploy.sh v2.3.2
```

## 验证更新

更新后，请验证以下内容：

1. **检查容器状态**
   ```bash
   docker ps | grep bitfinex-bot
   ```

2. **访问 Web 界面**
   - 打开浏览器访问：http://localhost:8089
   - **重要**: 强制刷新浏览器缓存（Ctrl+Shift+R 或 Cmd+Shift+R）

3. **验证币种选择器**
   - 在"机器人状态"卡片右上角应该看到币种选择下拉菜单
   - 下拉菜单应显示"所有币种"、"USD"、"UST"等选项

4. **检查浏览器控制台**
   - 按 F12 打开开发者工具
   - 在 Network 标签中，确认 JavaScript 文件 URL 包含 `?v=230`
   - 例如：`static/js/app.js?v=230`

## 清除浏览器缓存

如果更新后仍看不到币种选择器，请清除浏览器缓存：

### Chrome/Edge
1. 按 `Ctrl+Shift+Delete`（Mac: `Cmd+Shift+Delete`）
2. 选择"缓存的图片和文件"
3. 点击"清除数据"
4. 或者使用无痕模式访问

### Firefox
1. 按 `Ctrl+Shift+Delete`（Mac: `Cmd+Shift+Delete`）
2. 选择"缓存"
3. 点击"立即清除"

### Safari
1. 按 `Cmd+Option+E` 清空缓存
2. 或在"开发"菜单中选择"清空缓存"

## 版本历史

### v2.3.2 (2026-02-17)
- 🐛 修复 Docker 镜像缓存问题，确保包含最新 Web 文件

### v2.3.1 (2026-02-17)
- 🔧 更新 Web 文件版本号（v=230）以解决浏览器缓存问题
- ⚠️ 由于 Docker 构建缓存，此版本未能正确包含更新

### v2.3.0 (2026-02-17)
- ✨ 实现多币种同时放贷支持
- ✨ 添加币种选择器到 Web 界面
- ✨ 支持 USD、UST 等多个币种独立配置
- ✨ API 支持币种查询参数
- 🔄 配置自动迁移（向后兼容）

## 已知问题

无

## 技术细节

### Docker 构建优化
- 使用 `--no-cache` 标志确保所有文件都是最新版本
- 多阶段构建减小镜像体积
- 基于 Alpine Linux，镜像大小约 50-60 MB

### 缓存策略
- Web 静态文件使用版本号参数（`?v=230`）
- 确保浏览器加载最新版本的 JavaScript 和 CSS

## 支持

如有问题，请：
- 查看 [TROUBLESHOOTING_CURRENCY_SELECTOR.md](TROUBLESHOOTING_CURRENCY_SELECTOR.md)
- 查看 [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md)
- 提交 Issue: https://github.com/ApexLGF/BitfinexLBot/issues

## 下一步

1. 在部署环境执行更新命令
2. 强制刷新浏览器缓存
3. 验证币种选择器正常显示
4. 测试多币种功能
5. 监控日志确保无错误

祝部署顺利！🚀
