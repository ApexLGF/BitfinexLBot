# BitfinexBot v2.3.3 发布说明

## 发布日期
2026-02-18

## 版本信息
- **版本号**: v2.3.3
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.3.3`
- **镜像大小**: ~50-60 MB

## 🐛 关键修复

### 修复配置验证错误

**问题描述**：
v2.3.2 在远程部署环境启动失败，错误信息：
```
Failed to create application: failed to load config: [INVALID_INPUT] CURRENCY is required
```

**根本原因**：
虽然实现了多币种支持，但 `LendingBot` 结构仍然依赖旧的单币种配置字段（`config.Currency`），导致配置验证失败。

**解决方案**：
1. 重构 `LendingBot` 结构，添加 `currency` 和 `currencyConfig` 字段
2. 修改 `NewLendingBot` 构造函数，接收币种参数
3. 添加 `GetFundingSymbol()` 和 `GetMinDailyRateDecimal()` 方法
4. 将所有币种特定配置访问从 `lb.config` 改为 `lb.currencyConfig`
5. 简化 `CurrencyManager`，直接传递币种配置而不是创建临时配置对象

**修改的文件**：
- `internal/strategy/lending.go` - 重构 LendingBot 结构
- `internal/currency/manager.go` - 简化币种管理器

## ✅ 验证结果

本地测试通过：
```bash
$ ./bitfinex-bot -c config.yaml
[Config] 已為幣種 ust 設置預設值
[Config] 已為幣種 usd 設置預設值
[CurrencyManager] 初始化幣種: ust
[CurrencyManager] 初始化幣種: usd
[CurrencyManager] 已初始化 2 個幣種
🚀 === 正式模式啟動 ===
```

## 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.3.3

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
docker pull apexlgf/bitfinexwebbot:v2.3.3

# 3. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.3.3
```

### 方法 3：使用部署脚本

```bash
./deploy.sh v2.3.3
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
```

### 4. 访问 Web 界面
- 打开浏览器访问：http://localhost:8089
- **强制刷新浏览器缓存**（Ctrl+Shift+R 或 Cmd+Shift+R）
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

### 问题 2: 币种选择器不显示

**解决方案**：
1. 强制刷新浏览器缓存（Ctrl+Shift+R）
2. 清除浏览器缓存
3. 使用无痕模式测试

### 问题 3: 配置文件格式问题

**验证配置**：
```bash
docker exec bitfinex-bot-web /app/bitfinex-bot -c /app/config.yaml
```

如果看到错误，请检查：
- YAML 缩进（必须使用空格，不能用 Tab）
- 所有必需字段是否存在
- 字段值是否正确

## 版本历史

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

### 架构改进

**之前的问题**：
```go
// LendingBot 依赖 config.Currency（旧字段）
fundingSymbol := lb.config.GetFundingSymbol()  // 使用 config.Currency
minRate := lb.config.MinDailyLendRate          // 使用全局配置
```

**现在的实现**：
```go
// LendingBot 有自己的币种字段
type LendingBot struct {
    config         *config.Config          // 全局配置
    currencyConfig *config.CurrencyConfig  // 币种特定配置
    currency       string                  // 当前币种
    // ...
}

// 使用币种特定配置
fundingSymbol := lb.GetFundingSymbol()         // 使用 lb.currency
minRate := lb.currencyConfig.MinDailyLendRate  // 使用币种配置
```

### 配置访问模式

- **全局配置**：`lb.config.EnableSmartStrategy`、`lb.config.OrderLimit` 等
- **币种配置**：`lb.currencyConfig.MinLoan`、`lb.currencyConfig.MinDailyLendRate` 等
- **币种信息**：`lb.currency`、`lb.GetFundingSymbol()` 等

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
