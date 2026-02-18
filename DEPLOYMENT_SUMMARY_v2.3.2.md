# 🎉 v2.3.2 部署完成

## ✅ 已完成的工作

### 1. 问题诊断
- **问题**: v2.3.1 镜像由于 Docker 构建缓存，未能包含更新后的 Web 文件
- **表现**: 部署环境的 index.html 仍显示 `v=10` 而不是 `v=230`
- **根因**: Docker 构建时使用了缓存层，导致旧文件被保留

### 2. 解决方案
- 使用 `docker build --no-cache` 强制重新构建所有层
- 验证镜像内容确保包含正确的文件版本
- 推送新版本 v2.3.2 到 Docker Hub

### 3. 构建和推送
```bash
# 构建命令
docker build --no-cache -t apexlgf/bitfinexwebbot:v2.3.2 -t apexlgf/bitfinexwebbot:latest .

# 验证命令
docker run --rm --entrypoint cat apexlgf/bitfinexwebbot:v2.3.2 /usr/share/nginx/html/index.html | tail -10

# 验证结果 ✅
<script src="static/js/api.js?v=230"></script>
<script src="static/js/charts.js?v=230"></script>
<script src="static/js/app.js?v=230"></script>

# 推送到 Docker Hub ✅
docker push apexlgf/bitfinexwebbot:v2.3.2
docker push apexlgf/bitfinexwebbot:latest
```

### 4. 文档更新
- ✅ 更新 [docker-compose.yml](docker-compose.yml) 使用 v2.3.2
- ✅ 创建 [RELEASE_NOTES_v2.3.2.md](RELEASE_NOTES_v2.3.2.md) 发布说明
- ✅ 更新 [UPDATE_GUIDE.md](UPDATE_GUIDE.md) 快速更新指南

## 📦 镜像信息

- **仓库**: apexlgf/bitfinexwebbot
- **版本**: v2.3.2
- **标签**: v2.3.2, latest
- **镜像 ID**: sha256:bdc281b992f5fba976c92c5a987f0477d7b988f50e7935307e4c75db3e30be45
- **大小**: ~50-60 MB

## 🚀 部署步骤

### 在部署环境执行以下命令：

```bash
# 方法 1: 使用 Docker Compose（推荐）
cd /path/to/BitfinexLBot
docker compose pull
docker compose up -d --force-recreate

# 方法 2: 使用部署脚本
./deploy.sh v2.3.2

# 方法 3: 手动部署
docker stop bitfinex-bot-web
docker rm bitfinex-bot-web
docker rmi apexlgf/bitfinexwebbot:v2.3.1  # 删除旧版本
docker pull apexlgf/bitfinexwebbot:v2.3.2
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.3.2
```

## ✅ 验证清单

部署后请验证以下内容：

### 1. 容器状态
```bash
docker ps | grep bitfinex-bot
# 应该显示容器正在运行
```

### 2. 访问 Web 界面
- 打开浏览器访问：http://localhost:8089
- **重要**: 强制刷新浏览器缓存
  - Chrome/Edge: `Ctrl+Shift+R` (Mac: `Cmd+Shift+R`)
  - Firefox: `Ctrl+F5` (Mac: `Cmd+Shift+R`)
  - Safari: `Cmd+Option+R`

### 3. 验证币种选择器
- [ ] 在"机器人状态"卡片右上角看到币种选择下拉菜单
- [ ] 下拉菜单显示"所有币种"、"USD"、"UST"等选项
- [ ] 选择不同币种时，数据正确切换

### 4. 检查浏览器控制台
- 按 F12 打开开发者工具
- 在 Network 标签中，确认 JavaScript 文件 URL 包含 `?v=230`
- 例如：`static/js/app.js?v=230`

### 5. 功能测试
- [ ] 查看所有币种汇总数据
- [ ] 切换到单个币种查看详情
- [ ] 查看贷出挂单列表（币种列显示/隐藏正确）
- [ ] 查看已贷出订单列表（币种列显示/隐藏正确）
- [ ] 查看收益统计

## 🐛 故障排除

### 问题 1: 币种选择器仍然不显示

**解决方案**：
1. 确认使用的是 v2.3.2 镜像
   ```bash
   docker inspect bitfinex-bot-web | grep Image
   ```

2. 强制刷新浏览器缓存（Ctrl+Shift+R）

3. 清除浏览器缓存
   - Chrome: 设置 → 隐私和安全 → 清除浏览数据
   - 选择"缓存的图片和文件"
   - 点击"清除数据"

4. 使用无痕模式测试
   - Chrome: Ctrl+Shift+N
   - Firefox: Ctrl+Shift+P

### 问题 2: 镜像版本不对

**解决方案**：
```bash
# 删除所有旧版本镜像
docker rmi apexlgf/bitfinexwebbot:v2.3.0
docker rmi apexlgf/bitfinexwebbot:v2.3.1
docker rmi apexlgf/bitfinexwebbot:latest

# 重新拉取
docker pull apexlgf/bitfinexwebbot:v2.3.2

# 重新创建容器
docker compose up -d --force-recreate
```

### 问题 3: 容器无法启动

**解决方案**：
```bash
# 查看日志
docker logs bitfinex-bot-web

# 检查配置文件
cat config.yaml

# 验证配置
docker run --rm -v $(pwd)/config.yaml:/app/config.yaml:ro \
  apexlgf/bitfinexwebbot:v2.3.2 \
  /app/bitfinex-bot -c /app/config.yaml --validate
```

## 📚 相关文档

- [RELEASE_NOTES_v2.3.2.md](RELEASE_NOTES_v2.3.2.md) - 完整发布说明
- [UPDATE_GUIDE.md](UPDATE_GUIDE.md) - 快速更新指南
- [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md) - Docker 部署指南
- [TROUBLESHOOTING_CURRENCY_SELECTOR.md](TROUBLESHOOTING_CURRENCY_SELECTOR.md) - 币种选择器故障排除

## 🎯 下一步

1. ✅ 在部署环境执行更新命令
2. ✅ 强制刷新浏览器缓存
3. ✅ 验证币种选择器正常显示
4. ✅ 测试多币种功能
5. ✅ 监控日志确保无错误

## 📞 支持

如有问题，请：
- 查看上述故障排除部分
- 查看相关文档
- 提交 Issue: https://github.com/ApexLGF/BitfinexLBot/issues

---

**部署时间**: 2026-02-17
**版本**: v2.3.2
**状态**: ✅ 已完成并验证
