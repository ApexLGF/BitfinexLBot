# BitfinexBot v2.4.3 更新说明

## 发布日期
2026-02-18

## 版本信息
- **版本号**: v2.4.3
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.4.3`

## ✨ 新功能与改进

### 1. 币种统计卡片重新设计

**改进内容**：
- 将币种统计信息从 Tab 名称下方移至独立卡片
- 新卡片位于 30 日收益趋势图上方
- 采用 4 列网格布局，响应式设计
- 显示内容：
  - **已贷出**（绿色）：当前已贷出的总金额
  - **已挂单**（蓝色）：当前挂单的总金额
  - **剩余可用**（黄色）：可用余额
  - **总金额**（青色）：三者之和

**显示效果**：
```
┌─────────────────────────────────────────────────────┐
│  USD 资金统计                                        │
├─────────────────────────────────────────────────────┤
│  已贷出    已挂单    剩余可用    总金额              │
│  1000.00   500.00    200.00     1700.00            │
└─────────────────────────────────────────────────────┘
```

### 2. 系统日志显示优化

**改进内容**：
- 去掉前端添加的额外时间戳
- 直接显示后端日志原始内容
- 避免时间戳重复显示

**修改前**：
```
[21:19:19] 2026/02/18 21:14:49 [API] UST FRR 利率: 0.000156
```

**修改后**：
```
2026/02/18 21:14:49 [API] UST FRR 利率: 0.000156
```

### 3. 配置管理优化

**改进内容**：
- 移除"刷新配置"按钮（功能冗余）
- 新增显示每个币种的最小贷出利率
- 采用列表形式展示，清晰易读

**显示效果**：
```
┌─────────────────────────────────┐
│  配置管理                        │
├─────────────────────────────────┤
│  最小贷出利率                    │
│  USD        3.8000%             │
│  UST        3.6000%             │
│                                 │
│  [编辑配置]                      │
└─────────────────────────────────┘
```

---

## 🐛 Bug 修复

### 1. 修复 Tab 统计信息一直显示"加载中..."

**问题描述**：
Tab 名称下方的统计信息一直显示"加载中..."，数据未能正确更新。

**解决方案**：
将统计信息移至独立卡片，采用更可靠的数据更新机制。

### 2. 修复币种余额显示为 0 的问题

**问题描述**：
Web 界面显示的各个币种的剩余可用、总金额等数值都是 0，与实际账户中数值不符。

**根本原因**：
`getSingleCurrencyStatus` 方法传递的币种参数是小写（如 "usd"），但 Bitfinex API 返回的钱包币种是大写（如 "USD"）。在 `GetFundingBalance` 方法中比较时，大小写不匹配导致无法找到对应的钱包，返回 0。

**解决方案**：
在 `getSingleCurrencyStatus` 方法中，将币种名称转换为大写后再调用 `GetFundingBalance`，确保与 Bitfinex API 返回的币种名称格式一致。

**修改代码**：
```go
// 确保币种名称是大写
upperCurrency := strings.ToUpper(currency)

// 获取钱包余额
availableFunds, err := h.client.GetFundingBalance(upperCurrency)
```

### 3. 修复 API 返回 null 导致前端错误

**问题描述**：
当没有订单或已贷出记录时，API 返回 `null` 而不是空数组 `[]`，导致前端 JavaScript 调用 `.reduce()` 等数组方法时报错。

**根本原因**：
Go 语言中使用 `var offerData []OfferData` 声明的切片，当为空时会被 JSON 序列化为 `null`。

**解决方案**：
将所有可能返回空数组的地方改为使用 `make([]Type, 0)` 初始化，确保返回空数组而不是 null。

**修改代码**：
```go
// 修改前
var offerData []OfferData

// 修改后
offerData := make([]OfferData, 0)
```

**影响的方法**：
- `getSingleCurrencyOffers()`
- `getAllCurrenciesOffers()`
- `getSingleCurrencyCredits()`
- `getAllCurrenciesCredits()`

---

## 📝 修改文件

### 前端文件

1. **web/index.html**
   - 修改配置管理卡片，移除刷新配置按钮
   - 添加 `min-rates-container` 容器用于显示最小利率

2. **web/static/js/tabs.js**
   - 简化 `createTab()` 方法，移除统计信息显示
   - 修改 `createTabContent()` 方法，在顶部添加币种统计卡片
   - 移除 `updateTabStats()` 方法（功能转移到 dashboard.js）

3. **web/static/js/dashboard.js**
   - 修改 `updateTabStats()` 方法，更新统计卡片而非 Tab
   - 使用独立的 DOM 元素 ID 更新统计数据

4. **web/static/js/config-manager.js**
   - 移除 `refreshBtn` 相关代码
   - 新增 `displayMinRates()` 方法显示最小利率
   - 修改 `loadConfig()` 方法，调用 `displayMinRates()`

5. **web/static/js/app.js**
   - 修改 `updateLogDisplay()` 方法，移除额外时间戳

6. **web/static/css/app.css**
   - 新增 `.stat-item` 样式（统计卡片项）
   - 新增 `.min-rates-list` 和 `.min-rate-item` 样式
   - 移除 `.tab-currency-name` 和 `.tab-stats` 样式（不再使用）

### 后端文件

1. **internal/api/handlers.go**
   - 修复 `getSingleCurrencyStatus()` 方法中的币种大小写问题
   - 确保传递给 `GetFundingBalance()` 的币种名称为大写
   - 修复所有可能返回空数组的方法，使用 `make([]Type, 0)` 初始化

---

## 🔄 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 确认 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.4.3

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
docker pull apexlgf/bitfinexwebbot:v2.4.3

# 3. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.4.3
```

---

## ✅ 验证更新

更新后，请验证以下内容：

### 1. 检查币种统计卡片
- 打开浏览器访问：http://localhost:8089
- **强制刷新浏览器缓存**（Ctrl+Shift+R 或 Cmd+Shift+R）
- 确认每个币种 Tab 内容顶部显示统计卡片
- 确认统计卡片显示：已贷出、已挂单、剩余可用、总金额
- 切换不同 Tab，确认统计数据正确更新

### 2. 检查系统日志
- 确认日志不再有重复的时间戳
- 确认日志格式为：`2026/02/18 21:14:49 [API] ...`

### 3. 检查配置管理
- 确认右侧配置管理卡片不再有"刷新配置"按钮
- 确认显示每个币种的最小贷出利率
- 确认利率格式为百分比（如 3.8000%）

### 4. 功能测试
```bash
# 检查容器状态
docker ps | grep bitfinex-bot
# 应该显示容器正在运行且健康

# 检查 API 响应
curl http://localhost:8089/api/config | jq '.data.CURRENCIES.USD.MIN_DAILY_LEND_RATE'
# 应该返回最小利率数值
```

---

## 🎨 界面预览

### 币种统计卡片
```
┌──────────────────────────────────────────────────────┐
│  USD 资金统计                                         │
├──────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────┐│
│  │ 已贷出   │  │ 已挂单   │  │ 剩余可用 │  │ 总金额││
│  │ 1000.00  │  │ 500.00   │  │ 200.00   │  │1700.00││
│  └──────────┘  └──────────┘  └──────────┘  └──────┘│
└──────────────────────────────────────────────────────┘
```

### 配置管理卡片
```
┌─────────────────────────────────┐
│  配置管理                        │
├─────────────────────────────────┤
│  最小贷出利率                    │
│  ─────────────────────────────  │
│  USD                   3.8000%  │
│  UST                   3.6000%  │
│                                 │
│  ┌───────────────────────────┐ │
│  │     编辑配置              │ │
│  └───────────────────────────┘ │
└─────────────────────────────────┘
```

---

## 🐛 已知问题

无

---

## 📚 相关文档

- [v2.4.2 发布说明](RELEASE_NOTES_v2.4.2.md) - 系统日志窗口优化和 Tab 金额统计
- [v2.4.1 发布说明](RELEASE_NOTES_v2.4.1.md) - 修复"没有启用的币种"错误
- [v2.4.0 发布说明](RELEASE_NOTES_v2.4.0.md) - 完全重写多币种 Web 前端界面
- [Docker 部署指南](DOCKER_DEPLOYMENT.md)
- [故障排除指南](REMOTE_TROUBLESHOOTING.md)

---

## 版本历史

### v2.4.3 (2026-02-18)
- ✨ 币种统计卡片重新设计，移至独立卡片显示
- ✨ 配置管理新增显示每个币种的最小贷出利率
- 🐛 修复 Tab 统计信息一直显示"加载中..."的问题
- 🐛 修复币种余额显示为 0 的问题（币种大小写不匹配）
- 🐛 修复 API 返回 null 导致前端错误（改为返回空数组）
- 🔧 移除配置管理的"刷新配置"按钮
- 🔧 优化系统日志显示，去掉重复时间戳

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
