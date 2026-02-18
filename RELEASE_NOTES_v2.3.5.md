# BitfinexBot v2.3.5 发布说明

## 发布日期
2026-02-18

## 版本信息
- **版本号**: v2.3.5
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.3.5`
- **镜像大小**: ~50-60 MB

## 🐛 关键修复

### 修复 1: 日志中币种字段显示为空

**问题描述**：
v2.3.4 部署后，日志中显示：
```
Currency:   Available: 20000.000000
```
币种字段为空，无法识别是哪个币种的资金。

**根本原因**：
`internal/strategy/lending.go:97` 使用了 `lb.config.Currency`（在多币种模式下为空字符串），而不是 `lb.currency`（实际的币种字段）。

**解决方案**：
修改日志输出，使用正确的币种字段：
```go
// 修改前
log.Printf("Currency: %s  Available: %f", lb.config.Currency, fundsAvailable)

// 修改后
log.Printf("Currency: %s  Available: %f", lb.currency, fundsAvailable)
```

**修改的文件**：
- `internal/strategy/lending.go:97` - 修复日志输出使用正确的币种字段

---

### 修复 2: Web 界面配置管理显示错误数据

**问题描述**：
v2.3.4 部署后，Web 界面的配置管理功能刷新配置后，界面显示的数据不是配置文件中的内容。具体表现为：
- 显示的是旧的单币种配置格式（`CURRENCY`, `MIN_LOAN`, `MAX_LOAN` 等）
- 没有显示新的多币种配置（`CURRENCIES` 对象）
- 配置项不完整，缺少 K 线策略等新增配置

**根本原因**：
`internal/api/handlers.go` 中的 `GetConfig()` 方法仍然返回旧的单币种配置格式，没有适配多币种架构。

**解决方案**：
重构 `GetConfig()` 方法，返回正确的多币种配置格式：

```go
// 修改前 - 返回单币种格式
safeConfig := map[string]interface{}{
    "CURRENCY":           h.config.Currency,  // 空字符串
    "MIN_LOAN":           h.config.MinLoan,
    "MAX_LOAN":           h.config.MaxLoan,
    // ... 其他单币种字段
}

// 修改后 - 返回多币种格式
safeConfig := map[string]interface{}{
    // 基本设置
    "ORDER_LIMIT":        h.config.OrderLimit,
    "MINUTES_RUN":        h.config.MinutesRun,

    // 多币种配置
    "CURRENCIES":         h.config.Currencies,

    // 智能策略设置（全局）
    "ENABLE_SMART_STRATEGY":      h.config.EnableSmartStrategy,
    "VOLATILITY_THRESHOLD":       h.config.VolatilityThreshold,
    // ... 其他全局配置

    // K线策略设置（全局）
    "ENABLE_KLINE_STRATEGY": h.config.EnableKlineStrategy,
    "KLINE_TIME_FRAME":      h.config.KlineTimeFrame,
    // ... 其他 K 线配置
}
```

**修改的文件**：
- `internal/api/handlers.go:482-528` - 重构 `GetConfig()` 方法以返回多币种配置

---

## ✅ 验证结果

修复后，应该看到：

### 1. 日志正确显示币种
```
Currency: USD  Available: 10000.000000
Currency: UST  Available: 10000.000000
```

### 2. Web 界面配置管理正确显示
- 显示 `CURRENCIES` 对象，包含所有启用的币种配置
- 每个币种有独立的配置参数（`MIN_LOAN`, `MAX_LOAN`, `MIN_DAILY_LEND_RATE` 等）
- 显示全局配置（智能策略、K 线策略、系统设置等）
- 配置数据与 `config.yaml` 文件内容一致

---

## 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.3.5

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
docker pull apexlgf/bitfinexwebbot:v2.3.5

# 3. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.3.5
```

### 方法 3：使用部署脚本

```bash
./deploy.sh v2.3.5
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

### 3. 查看日志 - 验证币种显示正确
```bash
docker logs bitfinex-bot-web --tail 50
# 应该看到：
# Currency: USD  Available: 10000.000000
# Currency: UST  Available: 10000.000000
# （不再是空字符串）
```

### 4. 访问 Web 界面 - 验证配置管理
- 打开浏览器访问：http://localhost:8089
- **强制刷新浏览器缓存**（Ctrl+Shift+R 或 Cmd+Shift+R）
- 点击"配置管理"标签
- 点击"刷新配置"按钮
- 确认显示的配置数据与 `config.yaml` 文件内容一致
- 确认显示 `CURRENCIES` 对象，包含所有币种配置
- 确认显示全局配置（智能策略、K 线策略等）

### 5. 测试配置编辑功能
- 在配置管理界面修改某个参数
- 点击"保存配置"
- 验证配置文件已更新
- 验证机器人使用新配置运行

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

### 问题 1: 日志仍然显示空币种

**检查日志**：
```bash
docker logs bitfinex-bot-web --tail 100
```

**可能原因**：
1. 容器没有正确重启，仍在使用旧版本
2. 配置文件格式错误

**解决方案**：
```bash
# 强制重新创建容器
docker compose down
docker compose pull
docker compose up -d --force-recreate
```

### 问题 2: Web 界面配置管理仍然显示错误

**解决方案**：
1. 强制刷新浏览器缓存（Ctrl+Shift+R）
2. 清除浏览器缓存
3. 使用无痕模式测试
4. 检查浏览器控制台是否有 JavaScript 错误
5. 验证 API 响应：访问 http://localhost:8089/api/config

### 问题 3: 配置保存失败

**验证配置文件权限**：
```bash
# 检查配置文件挂载模式
docker inspect bitfinex-bot-web | grep -A 5 Mounts

# 如果是只读模式（:ro），改为读写模式（:rw）
# 修改 docker-compose.yml:
# - ./config.yaml:/app/config.yaml:rw
```

---

## 版本历史

### v2.3.5 (2026-02-18)
- 🐛 修复日志中币种字段显示为空的问题
- 🐛 修复 Web 界面配置管理显示错误数据的问题
- 🔧 重构 `GetConfig()` 方法以返回多币种配置格式

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

---

## 技术细节

### 修复 1: 日志币种字段

**问题代码**：
```go
// internal/strategy/lending.go:97
log.Printf("Currency: %s  Available: %f", lb.config.Currency, fundsAvailable)
```

**修复后**：
```go
// internal/strategy/lending.go:97
log.Printf("Currency: %s  Available: %f", lb.currency, fundsAvailable)
```

**原因**：
- `lb.config.Currency` 是全局配置字段，在多币种模式下为空字符串
- `lb.currency` 是 `LendingBot` 实例的币种字段，在创建时传入，始终有值

### 修复 2: 配置管理 API

**问题代码**：
```go
// internal/api/handlers.go:482-528
safeConfig := map[string]interface{}{
    "CURRENCY":           h.config.Currency,  // 空字符串
    "MIN_LOAN":           h.config.MinLoan,   // 全局字段，不适用于多币种
    // ... 其他单币种字段
}
```

**修复后**：
```go
// internal/api/handlers.go:482-528
safeConfig := map[string]interface{}{
    "ORDER_LIMIT":        h.config.OrderLimit,
    "MINUTES_RUN":        h.config.MinutesRun,
    "CURRENCIES":         h.config.Currencies,  // 多币种配置对象
    "ENABLE_SMART_STRATEGY": h.config.EnableSmartStrategy,
    "ENABLE_KLINE_STRATEGY": h.config.EnableKlineStrategy,
    // ... 其他全局配置
}
```

**原因**：
- 旧代码返回单币种配置格式，不适用于多币种架构
- 新代码返回 `CURRENCIES` 对象，包含所有币种的独立配置
- 新代码包含全局配置（智能策略、K 线策略等）

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
3. 检查日志确认币种显示正确
4. 访问 Web 界面测试配置管理功能
5. 监控运行状态

祝部署顺利！🚀
