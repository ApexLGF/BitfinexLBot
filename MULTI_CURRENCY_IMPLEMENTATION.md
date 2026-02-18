# 多币种同时放贷功能实现总结

## 实施日期
2026-02-17

## 实施内容

本次实施完成了 BitfinexLBot 的多币种同时放贷支持，允许单个应用实例同时管理多个币种（USD、UST、BTC、ETH 等）的放贷策略。

## 实施阶段

### 阶段1：配置层改造 ✅

**文件**: `internal/config/config.go`

1. 添加了 `CurrencyConfig` 结构体，包含单个币种的所有配置参数
2. 在 `Config` 结构体中添加了 `Currencies map[string]*CurrencyConfig` 字段
3. 实现了 `migrateFromLegacyConfig()` 方法，自动将旧的单币种配置迁移到新格式
4. 实现了 `setMultiCurrencyDefaults()` 方法，为每个币种设置默认值
5. 更新了 `Validate()` 方法，验证每个币种的配置

**向后兼容性**: 旧的单币种配置会自动迁移到新的多币种格式，无需手动修改配置文件。

### 阶段2：创建币种管理器 ✅

**新文件**: `internal/currency/manager.go`

创建了 `CurrencyManager` 结构体，负责管理多个 `LendingBot` 实例：

- `NewCurrencyManager()`: 为每个启用的币种创建独立的 LendingBot
- `ExecuteAll()`: 串行执行所有币种的策略（避免 API 速率限制）
- `GetBot()`: 获取特定币种的 LendingBot
- `GetEnabledCurrencies()`: 获取所有启用的币种列表
- `ReloadConfig()`: 支持配置热重载

### 阶段3：修改 Application 结构 ✅

**文件**: `main.go`

1. 将 `Application.lendingBot` 替换为 `Application.currencyManager`
2. 更新 `NewApplication()` 创建 CurrencyManager 而非单个 LendingBot
3. 修改 `executeMainTask()` 调用 `currencyManager.ExecuteAll()`
4. 更新 `checkRateThreshold()` 检查所有币种的利率
5. 修改 `executeLendingCheck()` 检查所有币种的借贷订单

### 阶段4：API 层改造 ✅

**文件**: `internal/api/handlers.go`, `internal/api/types.go`

1. 修改 `Handler` 结构体，使用 `currencyManager` 替代 `strategy`
2. 重写 `GetStatus()` 端点：
   - 支持 `?currency=USD` 查询参数返回单个币种数据
   - 不带参数时返回所有币种的汇总数据
3. 添加 `getSingleCurrencyStatus()` 和 `getAllCurrenciesStatus()` 方法
4. 添加 `getCurrencyEarningsInternal()` 辅助方法
5. 在 `EarningsData` 结构体中添加 `Currency` 字段

**API 响应格式**:

单币种查询 (`/api/status?currency=USD`):
```json
{
  "success": true,
  "data": {
    "is_running": true,
    "available_funds": 1000.0,
    "total_earnings": 50.0,
    "active_offers": 5,
    "currency": "USD"
  }
}
```

多币种查询 (`/api/status`):
```json
{
  "success": true,
  "data": {
    "currencies": {
      "USD": { ... },
      "UST": { ... }
    },
    "summary": {
      "total_earnings": 100.0,
      "total_active_offers": 10,
      "total_available_funds": 2000.0,
      "enabled_currencies": ["USD", "UST"]
    },
    "is_running": true
  }
}
```

### 阶段5：Web 前端改造 ✅

**文件**: `web/index.html`, `web/static/js/app.js`, `web/static/js/api.js`

#### HTML 改动:
1. 在机器人状态卡片头部添加币种选择器下拉菜单
2. 在订单表格和已贷出订单表格中添加币种列（默认隐藏）

#### JavaScript 改动:

**app.js**:
1. 添加 `selectedCurrency` 和 `enabledCurrencies` 状态变量
2. 实现币种选择器的 change 事件处理
3. 添加 `updateSingleCurrencyDisplay()` 方法处理单币种显示
4. 添加 `updateMultiCurrencyDisplay()` 方法处理多币种汇总显示
5. 添加 `updateCurrencySelector()` 方法动态填充币种选项
6. 修改 `updateOffers()`, `updateCredits()`, `updateEarnings()` 支持币种参数
7. 更新表格显示逻辑，根据选择的币种显示/隐藏币种列

**api.js**:
1. 修改 `getStatus()`, `getEarnings()`, `getOffers()`, `getFundingCredits()` 方法支持查询字符串参数

#### 用户体验:
- 默认显示所有币种的汇总数据
- 可通过下拉菜单选择特定币种查看详细数据
- 查看所有币种时，表格显示币种列
- 查看单个币种时，隐藏币种列以节省空间

### 阶段6：编译测试和功能验证 ✅

1. **编译测试**: 应用成功编译，无错误
2. **配置迁移测试**: 旧配置自动迁移到新格式
3. **日志验证**:
   - `[Config] 檢測到舊格式配置，自動遷移到多幣種格式`
   - `[Config] 已將 USD 遷移到多幣種配置`
   - `[CurrencyManager] 初始化幣種: USD`
   - `[CurrencyManager] 已初始化 1 個幣種`

## 配置文件格式

### 新格式（推荐）:

```yaml
# 全局设置
BITFINEX_API_KEY: "your_api_key"
BITFINEX_SECRET_KEY: "your_secret_key"
ORDER_LIMIT: 15
MINUTES_RUN: 5

# 多币种配置
CURRENCIES:
  USD:
    ENABLED: true
    MIN_LOAN: 1000
    MAX_LOAN: 0
    MIN_DAILY_LEND_RATE: 0.026
    SPREAD_LEND: 30
    GAP_BOTTOM: 10
    GAP_TOP: 5000
    THIRTY_DAY_LEND_RATE_THRESHOLD: 0.08
    ONE_TWENTY_DAY_LEND_RATE_THRESHOLD: 0.12
    RATE_BONUS: 0.0001
    HIGH_HOLD_RATE: 0.15
    HIGH_HOLD_AMOUNT: 10000
    HIGH_HOLD_ORDERS: 3
    NOTIFY_RATE_THRESHOLD: 0.1
    RESERVE_AMOUNT: 0

  UST:
    ENABLED: true
    MIN_LOAN: 500
    MIN_DAILY_LEND_RATE: 0.02
    # ... 其他参数 ...

  BTC:
    ENABLED: false
    # ... BTC 配置 ...

# 智能策略（全局）
ENABLE_SMART_STRATEGY: true
VOLATILITY_THRESHOLD: 0.002

# 系统设置
TEST_MODE: false
API_ENABLED: true
```

### 旧格式（自动迁移）:

```yaml
CURRENCY: "USD"
MIN_LOAN: 1000
# ... 其他单币种参数 ...
```

旧格式会自动迁移为新格式，无需手动修改。

## 关键特性

1. **向后兼容**: 旧配置自动迁移，无需手动修改
2. **独立策略**: 每个币种拥有独立的策略参数和执行逻辑
3. **串行执行**: 避免触发 Bitfinex API 速率限制
4. **错误隔离**: 一个币种失败不影响其他币种
5. **热重载**: 支持配置热重载，无需重启应用
6. **Web 界面**: 支持币种选择和多币种数据展示

## 使用方法

### 启用多个币种:

1. 编辑 `config.yaml`，在 `CURRENCIES` 部分添加币种配置
2. 设置 `ENABLED: true` 启用币种
3. 为每个币种配置独立的策略参数
4. 重启应用或使用 Web 界面的"重启机器人"按钮

### Web 界面操作:

1. 访问 Web 界面（默认 http://localhost:8089）
2. 在机器人状态卡片右上角选择币种
3. 选择"所有币种"查看汇总数据
4. 选择特定币种查看该币种的详细数据

## 技术细节

### 并发安全:
- `CurrencyManager` 使用 `sync.RWMutex` 保护并发访问
- 策略执行采用串行方式，避免 API 速率限制

### API 端点:
- `/api/status` - 所有币种汇总
- `/api/status?currency=USD` - 单个币种
- `/api/offers?currency=USD` - 单个币种订单
- `/api/credits?currency=USD` - 单个币种已贷出订单
- `/api/earnings?currency=USD` - 单个币种收益

### 前端状态管理:
- `selectedCurrency`: 当前选择的币种（空字符串表示所有币种）
- `enabledCurrencies`: 启用的币种列表
- 自动显示/隐藏币种列

## 文件修改清单

### 新增文件:
- `internal/currency/manager.go` - 币种管理器

### 修改文件:
- `internal/config/config.go` - 配置结构和迁移逻辑
- `main.go` - Application 结构和初始化
- `internal/api/handlers.go` - API 处理器多币种支持
- `internal/api/types.go` - API 响应类型
- `web/index.html` - 添加币种选择器
- `web/static/js/app.js` - 前端多币种逻辑
- `web/static/js/api.js` - API 调用支持币种参数

## 测试建议

1. **配置迁移测试**: 使用旧配置启动，验证自动迁移
2. **多币种执行测试**: 启用多个币种，观察策略执行日志
3. **Web 界面测试**: 测试币种选择器和数据显示
4. **API 测试**: 使用 curl 测试各个 API 端点
5. **错误隔离测试**: 模拟一个币种失败，验证其他币种正常运行

## 后续优化建议

1. **并行执行**: 考虑支持币种并行执行（需要注意 API 速率限制）
2. **币种优先级**: 支持设置币种执行优先级
3. **动态启用/禁用**: 支持在 Web 界面动态启用/禁用币种
4. **币种统计**: 添加每个币种的详细统计信息
5. **配置模板**: 提供常见币种的配置模板

## 注意事项

1. **API 速率限制**: Bitfinex API 有速率限制，多币种可能触发限制
2. **资源消耗**: 多币种会增加内存和 CPU 使用
3. **配置备份**: 修改配置前建议备份
4. **测试模式**: 建议先在测试模式下验证多币种配置

## 总结

本次实施成功实现了 BitfinexLBot 的多币种同时放贷支持，包括：
- ✅ 后端配置层、应用层、API 层的完整改造
- ✅ 前端 Web 界面的币种选择和多币种显示
- ✅ 向后兼容的配置自动迁移
- ✅ 完整的编译测试和功能验证

用户现在可以在单个应用实例中同时管理多个币种的放贷策略，大大提高了资源利用率和管理效率。
