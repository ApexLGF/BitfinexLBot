# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个用 Go 语言编写的 Bitfinex 放贷机器人，用于自动化保证金放贷策略。该机器人：
- 监控放贷利率并自动下达优化的放贷订单
- 实现复杂的多层放贷策略，支持动态利率调整
- 提供 Telegram 机器人集成，实现实时监控和配置
- 支持大额资金的高利率持有策略
- 包含基于利率的期间选择功能（2天、30天、120天贷款）

## 系统架构

**模块化架构**: 已从单文件应用重构为模块化结构，提高了可维护性和可测试性。

**核心模块**:
- `internal/config`: 使用 Viper 的配置管理（YAML 配置 + 环境变量）
- `internal/strategy`: 放贷策略实现和市场分析
- `internal/bitfinex`: Bitfinex API v2 REST 客户端封装
- `internal/telegram`: Telegram 机器人集成，用于监控和配置
- `internal/rates`: 利率计算和转换工具
- `internal/constants`: 应用常量和枚举
- `internal/errors`: 统一错误处理

**关键算法**:
- `strategy.LendingBot`: 核心策略引擎，计算最优放贷订单
- `strategy.SmartStrategy`: 高级智能策略，支持市场分析
- `strategy.MarketAnalyzer`: 市场深度和趋势分析
- 利率阶梯算法，使用资金簿位置（`GAP_BOTTOM` 到 `GAP_TOP`）
- 大额资金高利率持有策略
- 基于利率阈值的动态期间选择

## 开发命令

**构建和运行**:
```bash
make dev                          # 开发模式（启用测试模式）
make build                        # 构建可执行文件
make run                          # 使用生产配置运行
go run . -c config.yaml           # 直接用 Go 运行
```

**测试**:
```bash
make test                         # 运行完整测试套件（包含覆盖率）
make test-quick                   # 快速测试（不含覆盖率）
make test-verbose                 # 详细测试输出
./test.sh                         # 运行完整测试脚本
go test ./... -v                  # 手动运行测试
```

**代码格式化和验证**:
```bash
make format                       # 格式化代码
make lint                         # 静态分析
make mod-tidy                     # 清理依赖
gofmt -w .                        # 格式化所有 Go 文件
go vet ./...                      # 静态分析
```

**其他工具**:
```bash
make clean                        # 清理构建产物
make config-example               # 从示例创建配置文件
make security-check               # 检查敏感信息
make release                      # 构建发布包
```

## 配置管理

**主要配置**: `config.yaml` 包含所有机器人参数，包括 API 密钥、放贷策略和 Telegram 设置。

**关键设置**:
- `BITFINEX_API_KEY` / `BITFINEX_SECRET_KEY`: 交易凭证
- `MIN_DAILY_LEND_RATE`: 最低可接受放贷利率（安全阈值）
- `ORDER_LIMIT`: 单次执行周期的最大订单数（风险管理）
- `SPREAD_LEND`: 资金分散的订单数量
- `GAP_BOTTOM` / `GAP_TOP`: 利率计算的市场深度范围
- `HIGH_HOLD_*`: 大额资金的高级放贷策略
- `TEST_MODE`: 测试模式开关（true=不执行真实交易）

## 项目结构

```
BitfinexLendingBot/
├── main.go                    # 应用程序入口点
├── config.yaml               # 配置文件
├── Makefile                  # 构建和开发命令
├── test.sh                   # 测试运行脚本
├── go.mod                    # Go 模块依赖
└── internal/                 # 内部包
    ├── bitfinex/            # Bitfinex API 客户端
    ├── config/              # 配置管理
    ├── constants/           # 应用常量
    ├── errors/              # 错误定义
    ├── rates/               # 利率转换工具
    ├── strategy/            # 放贷策略
    └── telegram/            # Telegram 机器人集成
```

## 关键模块参考

**主应用程序**:
- `main.go` - 应用程序入口点和 CLI 处理
- `Application` 结构体 - 主应用程序协调器，支持优雅关闭

**策略实现**:
- `strategy.LendingBot` - 核心放贷策略引擎
- `strategy.SmartStrategy` - 高级智能策略
- `strategy.MarketAnalyzer` - 市场深度和趋势分析
- `strategy.GetLoanOffers()` - 计算最优放贷订单

**Bitfinex API 集成**:
- `bitfinex.Client` - Bitfinex API v2 REST 客户端封装
- `bitfinex.GetFundingOffers()` - 获取活跃的资金订单
- `bitfinex.CancelAllOffers()` - 取消所有资金订单
- `bitfinex.GetAvailableFunds()` - 获取钱包余额
- `bitfinex.GetLendingRate()` - 获取当前资金利率

**Telegram 机器人**:
- `telegram.Bot` - Telegram 机器人接口
- `telegram.HandleMessage()` - 处理传入消息
- `telegram.SendNotification()` - 发送通知
- 通过聊天命令进行动态配置更新

**配置和工具**:
- `config.LoadConfig()` - 加载和验证配置
- `rates.Converter` - 利率转换工具
- `constants` - 应用常量和枚举
- `errors` - 统一错误定义

## 测试

**测试结构**:
- `internal/config/config_test.go` - 配置加载和验证测试
- `internal/rates/converter_test.go` - 利率转换工具测试
- `internal/strategy/smart_strategy_test.go` - 智能策略算法测试
- `test.sh` - 完整测试运行脚本
- `test_graceful_shutdown.sh` - 优雅关闭测试

**覆盖率报告**:
- 测试生成 `coverage.out` 和 `coverage.html` 文件
- 运行 `make test` 生成覆盖率报告
- 目标：核心业务逻辑保持 >80% 测试覆盖率

**测试命令**:
```bash
go test ./... -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## V2 API 重要变更

- 资金符号使用 "f" 前缀（例如 "fUSD" 而不是 "USD"）
- 资金订单通过 `client.Funding.Offers(symbol)` 访问
- 钱包余额通过 `client.Wallet.Wallet()` 访问
- 资金簿通过 `client.Book.All(symbol, precision, limit)` 访问
- 利率值是日利率（在 v2 API 中不是年化利率）

## 开发技巧

- 使用 `make dev` 进行开发，启用测试模式
- 提交更改前使用 `make test`
- 配置更改可以通过 Telegram 机器人实时进行
- 运行测试后检查 `coverage.html` 进行覆盖率分析
- 使用 `make security-check` 验证提交中无敏感数据
- 原始单文件版本保存为 `backup/main_original.go`

## 安全注意事项

- API 凭证存储在 config.yaml 中（确保此文件不被提交到版本控制）
- Telegram 机器人令牌在 config.yaml 中
- Telegram 访问使用单一聊天 ID 认证
- `make security-check` 验证提交中无敏感信息
- 配置验证防止无效/危险设置
- 测试模式（`TEST_MODE: true`）防止真实交易操作

##两种核心放贷策略的解释
CalculateSmartOffers (智能策略)

  - 市场驱动型策略：基于实时市场数据和深度分析
  - 自适应算法：根据市场状况动态调整策略参数
  - 多维度决策：综合考虑趋势、波动率、竞争状况等因素

calculateKlineOffers (K线策略)

  - 技术分析导向：基于历史K线数据的技术分析
  - 趋势跟随策略：通过历史价格模式预测未来走势
  - 相对简单：专注于价格历史数据的统计分析

具体实现差异

1. 数据输入源

// 智能策略 - 使用实时市场深度数据
func (ss *SmartStrategy) CalculateSmartOffers(fundsAvailable float64, fundingBook []*bitfinex.FundingBookEntry)

// K线策略 - 使用历史K线数据  
candles, _ := lb.client.GetFundingCandles(lb.config.GetFundingSymbol(), lb.config.KlineTimeFrame,
lb.config.KlinePeriod)

2. 利率计算方法

智能策略：
- 市场状况分析：marketCondition := ss.analyzer.AnalyzeMarket(fundingBook)
- 动态利率调整：calculateDynamicHighHoldRate() 和 calculateProgressiveRate()
- 竞争分析优化：competitiveRate := ss.analyzer.AnalyzeCompetition(fundingBook)

K线策略：
- 技术指标分析：支持 max、SMA、EMA、HLA、P90 等多种平滑方法
- 简单利率加成：targetRate := highestRate * spreadMultiplier

3. 资金配置策略

智能策略：
// 动态资金配置
highHoldRatio, spreadRatio := ss.calculateOptimalAllocation(marketCondition)
highHoldAmount := fundsAvailable * highHoldRatio
spreadAmount := fundsAvailable * spreadRatio

K线策略：
// 固定配置逻辑
splitFundsAvailable := fundsAvailable
// 先处理高额持有，然后处理剩余资金

4. 期间选择逻辑

智能策略：
func (ss *SmartStrategy) calculateSmartPeriod(dailyRate float64, condition *MarketCondition) int {
    // 根据市场趋势智能调整期间
    switch condition.Trend {
    case "rising":
        // 利率上升趋势，偏向短期以便重新定价
    case "falling":
        // 利率下降趋势，锁定当前较高利率
    }
}

K线策略：
// 使用传统的基于利率阈值的期间选择
period := lb.calculatePeriod(rate)

5. 市场适应性

智能策略：
- 实时市场状况分析
- 根据波动率、趋势、利率比例动态调整
- 支持市场竞争分析

K线策略：
- 基于历史数据的趋势判断
- 相对固定的参数配置
- 专注于技术分析指标

总结

- CalculateSmartOffers 是一个复杂的自适应策略，适合对市场有深度理解且需要精细化管理的场景
- calculateKlineOffers 是一个简洁的技术分析策略，适合基于历史趋势进行决策的场景

两种策略可以根据不同的市场环境和用户偏好进行选择，智能策略更适合复杂多变的市场，K线策略更适合趋势明确的市场环境。