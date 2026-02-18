# BitfinexBot v2.3.4 发布说明

## 发布日期
2026-02-18

## 版本信息
- **版本号**: v2.3.4
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.3.4`
- **镜像大小**: ~50-60 MB

## 🐛 关键修复

### 修复多币种 API 调用问题

**问题描述**：
v2.3.3 在生产环境部署后，Web 界面无法显示任何数据（可用资金、总资金、总收益、订单等全部为 0），并且日志中频繁出现 API 错误：
```
[API] GetOffers 开始调用, symbol: f
获取账本记录失败: failed to get funding ledgers: API request failed with status 500: ["error",10100,"apikey: invalid"]
获取 usd 账本记录失败: failed to get funding ledgers: API request failed with status 500: ["error",10020,"currency: invalid"]
```

**根本原因**：
1. API handlers 中的 `GetOffers()`, `GetFundingCredits()`, `GetEarnings()` 方法仍然使用旧的 `h.config.Currency` 字段（在多币种模式下为空字符串）
2. 币种大小写不一致：配置文件使用小写（`usd`, `ust`），但 Bitfinex API 要求大写（`USD`, `UST`）
3. `GetFundingSymbol()` 依赖空的 `h.config.Currency`，导致 symbol 变成 `f` 而不是 `fUSD` 或 `fUST`

**解决方案**：
1. 重构 `GetOffers()` 方法，支持查询参数 `?currency=USD` 或返回所有币种
2. 重构 `GetFundingCredits()` 方法，支持查询参数 `?currency=USD` 或返回所有币种
3. 重构 `GetEarnings()` 方法，支持查询参数 `?currency=USD` 或返回所有币种
4. 修改 `processEarningsData()` 方法，添加 currency 参数
5. 修改 `getCurrencyEarningsInternal()` 方法，确保传递大写币种给 API
6. 修改 `CurrencyManager.GetEnabledCurrencies()` 方法，统一返回大写币种列表

**修改的文件**：
- `internal/api/handlers.go` - 重构所有 API 端点以支持多币种查询
- `internal/currency/manager.go` - 统一返回大写币种

## ✅ 验证结果

修复后，应该看到：
- Web 界面正确显示所有数据（可用资金、总资金、总收益、订单等）
- 日志中不再出现 `symbol: f` 或 `currency: invalid` 错误
- 所有 API 调用成功返回数据
- 币种选择器正常工作，可以切换查看不同币种的数据

## 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.3.4

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
docker pull apexlgf/bitfinexwebbot:v2.3.4

# 3. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.3.4
```

### 方法 3：使用部署脚本

```bash
./deploy.sh v2.3.4
```

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

### 3. 查看日志
```bash
docker logs bitfinex-bot-web --tail 50
# 应该看到：
# [CurrencyManager] 初始化幣種: usd
# [CurrencyManager] 初始化幣種: ust
# [CurrencyManager] 已初始化 2 個幣種
# 🚀 === 正式模式啟動 ===
# [API] GetOffers 开始调用, symbol: fUSD  (不再是 "f")
```

### 4. 访问 Web 界面
- 打开浏览器访问：http://localhost:8089
- **强制刷新浏览器缓存**（Ctrl+Shift+R 或 Cmd+Shift+R）
- 确认可用资金、总资金、总收益正确显示
- 确认订单列表正确显示
- 确认币种选择器正常显示

### 5. 测试多币种功能
- 在币种选择器中选择"所有币种"，查看汇总数据
- 选择"USD"，查看 USD 的详细数据
- 选择"UST"，查看 UST 的详细数据

## 配置要求

确保你的 `config.yaml` 使用多币种格式：

```yaml
# 多币种配置
CURRENCIES:
  USD:
    ENABLED: true
    MIN_LOAN: 1000
    MIN_DAILY_LEND_RATE: 0.032
    # ... 其他参数

  UST:
    ENABLED: true
    MIN_LOAN: 1000
    MIN_DAILY_LEND_RATE: 0.03
    # ... 其他参数
```

## 故障排除

### 问题 1: 容器仍然无法启动

**检查日志**：
```bash
docker logs bitfinex-bot-web --tail 100
docker exec bitfinex-bot-web cat /var/log/supervisor/bitfinex-bot-stderr.log
```

**常见原因**：
1. 配置文件格式错误（YAML 缩进问题）
2. API 密钥无效或权限不足
3. 端口冲突

### 问题 2: Web 界面仍然显示 0

**解决方案**：
1. 强制刷新浏览器缓存（Ctrl+Shift+R）
2. 清除浏览器缓存
3. 使用无痕模式测试
4. 检查浏览器控制台是否有 JavaScript 错误

### 问题 3: API 错误仍然存在

**验证配置**：
```bash
docker exec bitfinex-bot-web /app/bitfinex-bot -c /app/config.yaml
```

如果看到错误，请检查：
- YAML 缩进（必须使用空格，不能用 Tab）
- 所有必需字段是否存在
- 字段值是否正确
- API 密钥是否有效

## 版本历史

### v2.3.4 (2026-02-18)
- 🐛 修复多币种 API 调用问题
- 🔧 重构 API handlers 以支持多币种查询
- 🔧 统一币种大小写处理

### v2.3.3 (2026-02-18)
- 🐛 修复配置验证错误，支持多币种配置
- 🔧 重构 LendingBot 结构，添加币种字段
- ✨ 简化 CurrencyManager 实现

### v2.3.2 (2026-02-17)
- 🐛 修复 Docker 镜像缓存问题

### v2.3.1 (2026-02-17)
- 🔧 更新 Web 文件版本号

### v2.3.0 (2026-02-17)
- ✨ 实现多币种同时放贷支持
- ✨ 添加币种选择器到 Web 界面

## 技术细节

### API 端点改进

**之前的问题**：
```go
// 使用空的 h.config.Currency
symbol := h.config.GetFundingSymbol()  // 返回 "f"
```

**现在的实现**：
```go
// 支持查询参数
currency := c.DefaultQuery("currency", "")
if currency == "" {
    // 返回所有币种
    h.getAllCurrenciesOffers(c)
} else {
    // 返回单个币种
    symbol := constants.FundingSymbolPrefix + strings.ToUpper(currency)  // 返回 "fUSD"
}
```

### 币种大小写统一

**之前的问题**：
```go
// 返回小写币种
for currency := range cm.lendingBots {
    currencies = append(currencies, currency)  // "usd", "ust"
}
```

**现在的实现**：
```go
// 返回大写币种
for currency := range cm.lendingBots {
    currencies = append(currencies, strings.ToUpper(currency))  // "USD", "UST"
}
```

## 支持

如有问题，请：
- 查看 [REMOTE_TROUBLESHOOTING.md](REMOTE_TROUBLESHOOTING.md)
- 查看 [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md)
- 提交 Issue: https://github.com/ApexLGF/BitfinexLBot/issues

## 下一步

1. 在部署环境执行更新命令
2. 验证容器正常启动
3. 检查日志确认多币种初始化成功
4. 访问 Web 界面测试功能
5. 监控运行状态

祝部署顺利！🚀
