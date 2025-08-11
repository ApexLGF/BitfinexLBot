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
- `internal/bitfinex`: Bitfinex API v2 REST 客户端封装，包含线程安全的nonce生成器
- `internal/database`: MySQL 数据库客户端和命令处理系统
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
make build-linux                  # 构建 Linux 可执行文件
make run                          # 使用生产配置运行
go run . -c config.yaml           # 直接用 Go 运行
./bitfinex-lending-bot -c config.yaml  # 运行已构建的二进制文件
```

**测试**:
```bash
make test                         # 运行完整测试套件（包含覆盖率）
make test-quick                   # 快速测试（不含覆盖率）
make test-verbose                 # 详细测试输出
./test.sh                         # 运行完整测试脚本
./test_integration.sh             # 运行集成测试
go test ./internal/strategy -v -run TestSmartStrategy  # 运行特定测试
```

**代码格式化和验证**:
```bash
make format                       # 格式化代码
make lint                         # 静态分析
make mod-tidy                     # 清理依赖
make security-check               # 检查敏感信息泄漏
gofmt -w .                        # 格式化所有 Go 文件
go vet ./...                      # 静态分析
```

**Docker 开发**:
```bash
docker compose up -d              # 启动所有服务
docker compose down               # 停止所有服务
docker compose build --no-cache   # 强制重新构建镜像
docker compose logs bitfinex-bot  # 查看机器人日志
docker compose restart bitfinex-bot # 重启机器人服务
```

**其他工具**:
```bash
make clean                        # 清理构建产物
make config-example               # 从示例创建配置文件
make security-check               # 检查敏感信息
make release                      # 构建发布包
make deps                         # 检查和更新依赖
make install                      # 安装到系统路径
make uninstall                    # 从系统路径卸载
make docker-build                 # 构建 Docker 镜像
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
├── main.go                    # 应用程序入口点，Application 结构体协调所有服务
├── config.yaml               # 配置文件
├── docker-compose.yml         # Docker 服务编排
├── Dockerfile                # Go 应用容器化配置
├── Makefile                  # 构建和开发命令
├── test.sh                   # 测试运行脚本
├── go.mod                    # Go 模块依赖
├── database/                 # 数据库相关文件
│   ├── schema.sql           # MySQL 数据库结构
│   └── mysql.cnf            # MySQL 配置
├── web/                     # Web 管理界面
│   ├── index.html           # 主页面 (Bootstrap + JavaScript)
│   └── api/                 # Web API 接口
│       ├── index.php        # 主 API 入口点
│       ├── status.php       # 机器人状态 API
│       ├── config.php       # 配置管理 API
│       ├── commands.php     # 命令处理 API
│       └── earnings.php     # 收益数据 API
└── internal/                 # 内部包
    ├── bitfinex/            # Bitfinex API 客户端，包含自定义nonce生成器
    ├── config/              # 配置管理
    ├── constants/           # 应用常量
    ├── database/            # 数据库客户端和命令处理
    ├── errors/              # 错误定义
    ├── rates/               # 利率转换工具
    ├── strategy/            # 放贷策略
    └── telegram/            # Telegram 机器人集成
```

## 关键模块参考

**主应用程序**:
- `main.go:24` - Application 结构体定义，协调所有服务
- `main.go:39` - NewApplication() - 应用程序初始化流程
- `main.go:85` - Run() - 主运行循环和优雅关闭处理

**策略实现**:
- `strategy.LendingBot` - 核心放贷策略引擎，在 `internal/strategy/lending.go`
- `strategy.SmartStrategy` - 高级智能策略，在 `internal/strategy/smart_strategy.go`
- `strategy.MarketAnalyzer` - 市场深度和趋势分析，在 `internal/strategy/market_analyzer.go`
- `strategy.GetLoanOffers()` - 计算最优放贷订单，入口函数

**Bitfinex API 集成**:
- `bitfinex.Client` - Bitfinex API v2 REST 客户端封装，在 `internal/bitfinex/client.go`
- `bitfinex.CustomNonceGenerator` - 线程安全的nonce生成器，在 `internal/bitfinex/nonce.go`
- `bitfinex.GetFundingOffers()` - 获取活跃的资金订单
- `bitfinex.GetFundingCredits()` - 获取借贷记录，支持直接HTTP调用
- `bitfinex.GetDailyFundingEarnings()` - 获取24小时收益，带时间范围过滤
- `bitfinex.GetWeeklyFundingEarnings()` - 获取1周收益
- `bitfinex.GetCurrentFundingRate()` - 获取当前资金利率

**Telegram 机器人**:
- `telegram.Bot` - Telegram 机器人接口（注：已被重构为数据库命令系统）
- `database.CommandHandler` - 异步命令处理系统，替代直接Telegram交互
- `database.SaveNotification()` - 通知存储系统

**数据库系统**:
- `database.Client` - MySQL 数据库客户端，连接管理在 `internal/database/database.go`
- `database.CommandHandler` - 异步命令处理系统，在 `internal/database/command_handler.go`
- `database.BotStatus` - 机器人状态管理和持久化

**配置和工具**:
- `config.LoadConfig()` - 加载和验证配置，在 `internal/config/config.go`
- `rates.Converter` - 利率转换工具，在 `internal/rates/converter.go`
- `constants` - 应用常量和枚举，在 `internal/constants/constants.go`
- `errors` - 统一错误定义，在 `internal/errors/errors.go`

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

## Docker 架构

**服务组件**:
- `bitfinex-bot`: Go 应用主服务，处理放贷策略和API调用
- `mysql`: MySQL 8.0 数据库，存储状态、命令和通知
- `web`: Nginx + PHP 8.1 Web 管理界面 (端口 8080)
- `phpmyadmin`: 数据库管理界面 (端口 8081)

**服务访问**:
```bash
http://localhost:8080    # Web 管理界面
http://localhost:8081    # phpMyAdmin 数据库管理
```

**重要Docker注意事项**:
- 修改代码后必须使用 `docker compose build --no-cache` 强制重新构建
- 使用 `docker compose down` 完全停止服务，然后重新启动以确保应用最新代码
- 数据库数据持久化在 `mysql_data` volume 中

## Bitfinex API v2 重要变更

- 资金符号使用 "f" 前缀（例如 "fUSD" 而不是 "USD"）
- 资金订单通过 `client.Funding.Offers(symbol)` 访问
- 钱包余额通过 `client.Wallet.Wallet()` 访问
- 资金簿通过 `client.Book.All(symbol, precision, limit)` 访问
- 利率值是日利率（在 v2 API 中不是年化利率）

**Nonce 管理重要说明**:
- 使用自定义 `CustomNonceGenerator` 确保线程安全
- 初始值使用微秒级时间戳: `time.Now().UnixNano() / 1000`
- 可配置递增步长（默认100）避免高频API调用冲突
- 所有API调用（SDK和直接HTTP）使用统一nonce生成器实例

## 开发技巧与故障排除

**常用调试命令**:
```bash
# 查看实时日志
docker compose logs -f bitfinex-bot

# 检查数据库状态
docker compose exec mysql mysql -u root -p bitfinex_bot_db -e "SELECT * FROM bot_status;"

# 测试API连接
go run . -c config.yaml --dry-run

# 查看配置文件验证结果
go run . -c config.yaml --validate-config

# 初始化 Docker 环境
./docker-setup.sh

# 运行 Docker 测试
./docker-test.sh
```

**常见错误处理**:
- **Nonce错误**: 检查 `internal/bitfinex/nonce.go` 中的步长配置，减少并发请求
- **API限流**: 调整 `config.yaml` 中的 `REQUEST_INTERVAL` 参数
- **数据库连接失败**: 确认 MySQL 容器已启动，检查 `DATABASE_DSN` 配置
- **配置验证失败**: 使用 `make config-example` 重新生成配置模板

**性能优化建议**:
- 使用 `make dev` 进行开发，启用测试模式
- 提交更改前使用 `make test`
- Docker开发时，代码修改后必须重新构建: `docker compose down && docker compose build --no-cache && docker compose up -d`
- 配置更改可以通过 Web界面管理 (http://localhost:8080)
- 运行测试后检查 `coverage.html` 进行覆盖率分析
- 使用 `make security-check` 验证提交中无敏感数据
- Web界面提供实时状态监控，包括24小时/1周收益展示
- 收益数据每小时自动更新，避免频繁API调用影响性能

**测试最佳实践**:
- 单元测试：专注于 `internal/` 目录下的业务逻辑
- 集成测试：使用 `test_integration.sh` 验证完整工作流
- 性能测试：使用 `go test -bench` 测试策略计算性能
- 覆盖率目标：核心业务逻辑保持 >80% 测试覆盖率

**配置验证检查点**:
- 验证 API 密钥格式 (以 `Bfx` 开头)
- 检查利率范围 (建议 0.0001-0.1 日利率)
- 确认 Telegram Chat ID 为数字格式
- 验证数据库连接字符串格式

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

## 生产环境部署

**Docker Hub 镜像**: `apexlgf/bitfenix-bot:latest` - 官方构建的生产就绪镜像

**生产环境集成**:
- 支持与现有 nginx/php/mysql 环境无缝集成
- 提供完整的生产环境 Docker Compose 配置
- 包含数据库权限修复和初始化工具
- Web 管理界面可集成到现有 Nginx 配置中

**生产环境文件**:
- `production-docker-compose.yml` - 适配现有环境的 Docker Compose 配置
- `production-config.yaml` - 生产环境优化的配置模板
- `fix-production-database.sh` - 数据库权限修复工具
- `PRODUCTION_DEPLOYMENT_GUIDE.md` - 完整的生产环境部署指南

**生产环境部署命令**:
```bash
# 拉取生产镜像
docker pull apexlgf/bitfenix-bot:latest

# 修复数据库权限（如果需要）
./fix-production-database.sh

# 部署到生产环境
docker compose -f production-docker-compose.yml up -d

# 查看生产环境日志
docker compose logs -f bitfinex-bot
```

**生产环境访问地址**:
- BitfinexBot Web 管理界面: `http://your-server/bitfinex/`
- 数据库管理: `http://your-server:8081` (phpMyAdmin)
- 原有应用保持: `http://your-server/`

**生产环境关键配置**:
- 必须设置 `TEST_MODE: false` 启用真实交易
- 配置真实的 Bitfinex API 密钥
- 调整利率策略参数适应生产环境
- 启用数据库持久化和备份

**生产环境故障排除**:
- 数据库连接问题: 运行 `fix-production-database.sh`
- 权限问题: 检查 Docker 容器网络配置
- API 连接失败: 验证 API 密钥和网络连接
- Web 界面无法访问: 检查 Nginx 配置和文件权限

## Docker 镜像构建和发布

**本地构建镜像**:
```bash
# 构建本地镜像
docker build -t apexlgf/bitfenix-bot:latest .

# 推送到 Docker Hub (需要登录)
docker push apexlgf/bitfenix-bot:latest
```

**镜像特性**:
- 基于 Alpine Linux 的多阶段构建
- 优化的镜像大小 (~83MB)
- 包含时区设置 (Asia/Shanghai)
- 非 root 用户运行增强安全性
- 内置健康检查和信号处理