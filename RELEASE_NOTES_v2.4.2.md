# BitfinexBot v2.4.2 更新说明

## 发布日期
2026-02-18

## 版本信息
- **版本号**: v2.4.2
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.4.2`

## ✨ 新功能

### 1. 系统日志窗口优化

**改进内容**：
- 固定日志窗口高度为 400px
- 添加 `max-height` 限制，确保不超过左侧面板高度
- 支持窗口内滚动查看历史日志
- 保持智能自动滚动功能（用户在底部时才自动滚动）

**技术实现**：
```css
.log-terminal {
    height: 400px;
    max-height: 400px;
    overflow-y: auto;
}
```

### 2. Tab 显示币种金额统计

**新增显示内容**：
每个币种 Tab 名称下方显示实时金额统计：
- **贷**：已贷出金额（绿色）
- **挂**：已挂单金额（蓝色）
- **余**：剩余可用金额（黄色）
- **总**：总金额（青色）

**显示效果**：
```
USD
贷: 1000  挂: 500  余: 200  总: 1700
```

**技术实现**：
- 修改 `tabs.js` 的 `createTab()` 方法，添加统计信息显示区域
- 新增 `updateTabStats()` 方法，动态更新 Tab 统计数据
- 修改 `dashboard.js` 的 `render()` 方法，在加载数据后更新 Tab 统计
- 添加 CSS 样式支持 Tab 内的多行布局

---

## 📝 修改文件

### 前端文件
1. **web/static/css/app.css**
   - 修改 `.log-terminal` 样式，固定高度为 400px
   - 新增 `.tab-currency-name` 和 `.tab-stats` 样式

2. **web/static/js/tabs.js**
   - 修改 `createTab()` 方法，添加金额统计显示
   - 新增 `updateTabStats()` 方法

3. **web/static/js/dashboard.js**
   - 修改 `render()` 方法，增加 `status` API 调用
   - 新增 `updateTabStats()` 方法，计算并更新 Tab 统计

---

## 🔄 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.4.2

# 2. 拉取新镜像并重新创建容器
docker compose pull
docker compose up -d --force-recreate
```

### 方法 2：手动更新

```bash
# 1. 停止并删除旧容器
docker stop bitfinex-bot-web
docker rm bitfinex-bot-web

# 2. 拉取新镜像
docker pull apexlgf/bitfinexwebbot:v2.4.2

# 3. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.4.2
```

---

## ✅ 验证更新

更新后，请验证以下内容：

### 1. 检查系统日志窗口
- 打开浏览器访问：http://localhost:8089
- **强制刷新浏览器缓存**（Ctrl+Shift+R 或 Cmd+Shift+R）
- 确认系统日志窗口高度固定，不会过高
- 确认可以在日志窗口内滚动查看

### 2. 检查 Tab 金额显示
- 确认每个币种 Tab 下方显示金额统计
- 确认显示格式：`贷: xxx  挂: xxx  余: xxx  总: xxx`
- 切换不同 Tab，确认金额数据正确更新
- 鼠标悬停在金额上，确认有 tooltip 提示

### 3. 功能测试
```bash
# 检查容器状态
docker ps | grep bitfinex-bot
# 应该显示容器正在运行且健康

# 检查 API 响应
curl http://localhost:8089/api/status?currency=USD | jq '.data.available_funds'
# 应该返回可用余额数字
```

---

## 🎨 界面预览

### Tab 显示效果
```
┌─────────────────────────────────────────┐
│  USD                    UST             │
│  贷: 1000  挂: 500     贷: 50  挂: 20   │
│  余: 200   总: 1700    余: 10  总: 80   │
└─────────────────────────────────────────┘
```

### 系统日志窗口
```
┌─────────────────────────────────────────┐
│  系统日志                                │
├─────────────────────────────────────────┤
│  [21:00:00] 开始执行贷出机器人...       │
│  [21:00:01] 取消所有未完成订单...       │
│  [21:00:02] 目前没有未完成的订单         │
│  [21:00:03] 取得可用额度...             │
│  ...                                    │
│  ↕ (固定高度 400px，可滚动)             │
└─────────────────────────────────────────┘
```

---

## 🐛 已知问题

无

---

## 📚 相关文档

- [v2.4.1 发布说明](RELEASE_NOTES_v2.4.1.md) - 修复"没有启用的币种"错误
- [v2.4.0 发布说明](RELEASE_NOTES_v2.4.0.md) - 完全重写多币种 Web 前端界面
- [Docker 部署指南](DOCKER_DEPLOYMENT.md)
- [故障排除指南](REMOTE_TROUBLESHOOTING.md)

---

## 版本历史

### v2.4.2 (2026-02-18)
- ✨ 系统日志窗口固定高度，支持滚动查看
- ✨ Tab 显示币种金额统计（已贷出、已挂单、剩余、总金额）

### v2.4.1 (2026-02-18)
- 🐛 修复 Web 界面"没有启用的币种"错误
- 🔧 为 `CurrencyConfig` 结构体添加 JSON 标签

### v2.4.0 (2026-02-18)
- 🎉 完全重写多币种 Web 前端界面
- ✨ 新增 70-30 分栏布局
- ✨ 新增币种 TAB 导航

---

祝使用愉快！🚀
