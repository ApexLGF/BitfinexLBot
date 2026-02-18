# BitfinexBot v2.4.0 发布说明

## 发布日期
2026-02-18

## 版本信息
- **版本号**: v2.4.0
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.4.0`
- **镜像大小**: ~50-60 MB

## 🎉 重大更新

### 完全重写多币种 Web 前端界面

本版本对 Web 界面进行了彻底重写，实现了更直观、更高效的多币种管理体验。

---

## ✨ 新功能

### 1. 全新 70-30 分栏布局

**左侧面板（70%）**：
- 币种 TAB 导航，支持快速切换不同币种
- 每个币种独立显示详细信息

**右侧面板（30%）**：
- 整体放贷信息（FRR 利率、年度收益）
- 配置管理
- 系统日志（黑底白字，智能滚动）

### 2. 币种 TAB 导航

- 多 TAB 结构，每个币种一个独立 TAB
- TAB 名称直接使用币种名（USD、UST 等）
- 默认显示第一个启用的币种
- 点击 TAB 即可切换查看不同币种数据

### 3. 每个币种 TAB 包含

**30日收益趋势图**：
- 柱状图显示每日收益
- 折线图显示累计收益
- 双 Y 轴设计，数据清晰易读

**贷出挂单列表**：
- 显示数量、总额、平均利率
- 表格展示所有活跃订单详情

**已贷出订单列表**：
- 显示数量、总额、平均利率
- 表格展示所有已贷出订单详情

### 4. 右侧整体信息面板

**当前 FRR 利率**：
- 显示所有币种的实时 FRR（Flash Return Rate）利率
- 百分比格式显示，精确到小数点后4位

**年度累计收益**：
- 显示每个币种最近365天的累计收益
- 显示总计收益
- 纯数字格式，无货币符号

### 5. 配置管理增强

- 支持编辑多币种配置
- 每个币种独立配置区域
- 支持启用/禁用币种
- 支持编辑所有币种参数

### 6. 系统日志优化

- 黑底白字终端风格
- 智能自动滚动（仅当用户在底部��才滚动）
- 实时更新（3秒轮询）
- 自定义滚动条样式

### 7. 纯数字格式化

- 所有数值显示为纯数字（如 1000.50）
- 不带货币符号（$）或单位
- 利率显示为百��比（如 3.5000%）

---

## 🔧 后端新增 API

### 1. FRR 利率端点

```
GET /api/frr-rates
```

**响应示例**：
```json
{
  "success": true,
  "data": {
    "USD": 0.035,
    "UST": 0.038
  }
}
```

### 2. 年度收益端点

```
GET /api/earnings/yearly
```

**响应示例**：
```json
{
  "success": true,
  "data": {
    "currencies": {
      "USD": 1250.50,
      "UST": 850.30
    },
    "total": 2100.80
  }
}
```

---

## 📁 新增文件

### JavaScript 模块

- `web/static/js/tabs.js` - 币种 TAB 管理器
- `web/static/js/dashboard.js` - 币种详情面板管理器
- `web/static/js/overall-info.js` - 整体信息面板管理器
- `web/static/js/config-manager.js` - 配置管理器

### 修改文件

- `web/index.html` - 完全重写，实现 70-30 布局
- `web/static/js/app.js` - 重构主应用逻辑
- `web/static/css/app.css` - 新增样式支持新布局
- `internal/api/handlers.go` - 新增 FRR 和年度收益端点
- `internal/api/server.go` - 注册新 API 路由

---

## 🎨 界面特性

### 响应式设计

- 桌面（≥992px）：70-30 分栏
- 平板（768-991px）：70-30 分栏
- 手机（<768px）：垂直堆叠，左侧面板优先

### 视觉优化

- Bootstrap 5.3 现代化设计
- 卡片阴影和悬停效果
- 平滑过渡动画
- 自定义滚动条
- 深色日志终端

### 用户体验

- 智能日志滚动（不打扰用户查看历史）
- 实时数据更新
- 快速 TAB 切换
- 清晰的数据展示

---

## 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.4.0

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
docker pull apexlgf/bitfinexwebbot:v2.4.0

# 3. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.4.0
```

### 方法 3：使用部署脚本

```bash
./deploy.sh v2.4.0
```

---

## 验证更新

更新后，请验证以下内容：

### 1. 检查容器状态
```bash
docker ps | grep bitfinex-bot
# 应该显示容器正在运行
```

### 2. 检查进程状态
```bash
docker exec bitfinex-bot-web supervisorctl status
# 应该显示：
# bitfinex-bot    RUNNING   pid xxx, uptime x:xx:xx
# nginx           RUNNING   pid xxx, uptime x:xx:xx
```

### 3. 访问 Web 界面
- 打开浏览器访问：http://localhost:8089
- **强制刷新浏览器缓存**（Ctrl+Shift+R 或 Cmd+Shift+R）
- 确认看到新的 70-30 分栏布局
- 确认看到币种 TAB 导航

### 4. 测试币种 TAB 切换
- 点击不同的币种 TAB（USD、UST 等）
- 确认每个 TAB 显示对应币种的数据
- 确认收益图表正确显示
- 确认订单列表正确显示

### 5. 测试右侧面板
- 确认 FRR 利率正确显示
- 确认年度收益正确显示
- 确认系统日志正常滚动
- 测试配置管理功能

### 6. 测试配置编辑
- 点击"编辑配置"按钮
- 确认看到多币种配置表单
- 修改某个参数并保存
- 验证配置已更新

---

## 配置要求

确保你的 `config.yaml` 使用多币种格式：

```yaml
# 多币种配置
CURRENCIES:
  USD:
    ENABLED: true
    MIN_LOAN: 1000
    MIN_DAILY_LEND_RATE: 0.032
    SPREAD_LEND: 5
    GAP_BOTTOM: 5000
    GAP_TOP: 10000
    THIRTY_DAY_LEND_RATE_THRESHOLD: 0.05
    ONE_TWENTY_DAY_LEND_RATE_THRESHOLD: 0.08
    RATE_BONUS: 0.0001
    HIGH_HOLD_RATE: 0.15
    HIGH_HOLD_AMOUNT: 5000
    HIGH_HOLD_ORDERS: 2
    NOTIFY_RATE_THRESHOLD: 0.1
    RESERVE_AMOUNT: 100

  UST:
    ENABLED: true
    MIN_LOAN: 1000
    MIN_DAILY_LEND_RATE: 0.03
    # ... 其他参数
```

---

## 故障排除

### 问题 1: 界面显示旧版本

**解决方案**：
1. 强制刷新浏览器缓存（Ctrl+Shift+R）
2. 清除浏览器缓存
3. 使用无痕模式测试
4. 检查容器是否使用新版本：`docker inspect bitfinex-bot-web | grep Image`

### 问题 2: TAB 不显示或为空

**可能原因**：
1. 配置文件中没有启用的币种
2. 配置文件格式错误

**解决方案**：
```bash
# 检查日志
docker logs bitfinex-bot-web --tail 100

# 验证配置
docker exec bitfinex-bot-web cat /app/config.yaml
```

### 问题 3: FRR 利率或年度收益不显示

**解决方案**：
1. 检查浏览器控制台是否有 JavaScript 错误
2. 检查 API 响应：访问 http://localhost:8089/api/frr-rates
3. 检查 API 响应：访问 http://localhost:8089/api/earnings/yearly
4. 检查后端日志：`docker logs bitfinex-bot-web`

### 问题 4: 日志不滚动

**解决方案**：
- 日志使用智能滚动，只有当你在底部时才会自动滚动
- 如果你在查看历史日志，它不会自动滚动
- 滚动到底部即可恢复自动滚动

---

## 版本历史

### v2.4.0 (2026-02-18)
- 🎉 完全重写多币种 Web 前端界面
- ✨ 新增 70-30 分栏布局
- ✨ 新增币种 TAB 导航
- ✨ 新增 FRR 利率显示
- ✨ 新增年度收益显示
- ✨ 优化配置管理支持多币种
- ✨ 优化系统日志显示（黑底白字，智能滚动）
- 🔧 新增 `/api/frr-rates` 端点
- 🔧 新增 `/api/earnings/yearly` 端点

### v2.3.5 (2026-02-18)
- 🐛 修复日志中币种字段显示为空的问题
- 🐛 修复 Web 界面配置管理显示错误数据的问题

### v2.3.4 (2026-02-18)
- 🐛 修复多币种 API 调用问题
- 🔧 重构 API handlers 以支持多币种查询

### v2.3.3 (2026-02-18)
- 🐛 修复配置验证错误，支持多币种配置
- 🔧 重构 LendingBot 结构

---

## 技术细节

### 前端架构

**模块化设计**：
- `tabs.js` - TAB 导航管理
- `dashboard.js` - 币种面板渲染
- `overall-info.js` - 整体信息管理
- `config-manager.js` - 配置编辑
- `app.js` - 主应用协调器

**数据流**：
```
用户操作 → TAB 切换 → 触发事件 → Dashboard 渲染 → API 调用 → 更新 UI
```

**轮询机制**：
- 日志：3秒轮询
- 整体信息：60秒轮询
- 币种数据：按需加载（TAB 切换时）

### 后端架构

**新增端点**：
- `GetFRRRates()` - 获取所有币种 FRR 利率
- `GetYearlyEarnings()` - 获取365天累计收益

**数据处理**：
- 并行获取多币种数据
- 过滤放贷收益记录
- 计算累计收益

---

## 支持

如有问题，请：
- 查看 [REMOTE_TROUBLESHOOTING.md](REMOTE_TROUBLESHOOTING.md)
- 查看 [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md)
- 提交 Issue: https://github.com/ApexLGF/BitfinexLBot/issues

---

## 下一步

1. 在部署环境执行更新命令
2. 验证容器正常启动
3. 访问 Web 界面测试新功能
4. 测试币种 TAB 切换
5. 测试配置管理功能
6. 监控运行状态

祝使用愉快！🚀
