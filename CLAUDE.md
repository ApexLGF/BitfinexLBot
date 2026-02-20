# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Go 语言编写的 Bitfinex 放贷机器人（需要 Go 1.23+），用于自动化保证金放贷策略：
- 监控放贷利率并自动下达优化的放贷订单
- 支持多币种同时放贷（USD、BTC、ETH 等）
- 实现智能策略和 K 线策略两种放贷模式
- 提供 Web 界面和 REST API，支持配置热重载
- Docker 一体化部署（Nginx + Go 应用 + Supervisor）

## 系统架构

**模块化架构**: 已从单文件应用重构为模块化结构，提高了可维护性和可测试性。

**核心模块**:
- `internal/config`: 使用 Viper 的配置管理（YAML 配置 + 环境变量）
- `internal/currency`: 多币种管理器，协调多个 LendingBot 实例
- `internal/strategy`: 放贷策略实现（LendingBot、SmartStrategy、MarketAnalyzer、LLMAIStrategy）
- `internal/bitfinex`: Bitfinex API v2 REST 客户端封装
- `internal/api`: Web API 服务器和处理器（REST API + 配置管理）
- `internal/rates`: 利率计算和转换工具

**Web 界面架构**:
- `web/index.html`: 响应式 Web 控制台主页面
- `web/static/`: 静态资源（CSS、JS、图片）
- Nginx 反向代理（端口 8089）到 Go API 服务（端口 8090）
- Supervisor 进程管理，确保服务稳定运行

**关键算法**:
- `strategy.LendingBot`: 核心策略引擎，计算最优放贷订单
- `strategy.SmartStrategy`: 高级智能策略，支持市场分析和 LLM AI 预测
- `strategy.MarketAnalyzer`: 市场深度和趋势分析
- `strategy.LLMAIStrategy`: LLM AI 利率预测策略（支持 OpenAI 兼容 API）
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
go test ./internal/strategy/... -v -run TestSmartStrategy  # 运行单个测试
```

**代码格式化和验证**:
```bash
make format                       # 格式化代码
make lint                         # 静态分析
make mod-tidy                     # 清理依赖
gofmt -w .                        # 格式化所有 Go 文件
go vet ./...                      # 静态分析
```

**Docker 和 Web 界面**:
```bash
make docker-build                 # 构建 Docker 镜像
make build-linux                  # 构建 Linux 可执行文件
docker compose up -d              # 启动 Web 服务（推荐）
docker compose down               # 停止 Web 服务
docker compose up -d --build      # 重新构建并启动
docker compose logs -f            # 查看实时日志
docker pull apexlgf/bitfinexwebbot:latest  # 拉取官方镜像
```

**Web 界面访问**:
- 启动后访问: http://localhost:8089
- 功能包括：状态监控、收益统计、订单管理、配置编辑、实时日志

**其他工具**:
```bash
make clean                        # 清理构建产物
make config-example               # 从示例创建配置文件
make security-check               # 检查敏感信息
make release                      # 构建发布包
```

## 配置管理

**主要配置**: `config.yaml` 包含所有机器人参数。支持多币种配置，每个币种可独立设置策略参数。

**关键设置**:
- `BITFINEX_API_KEY` / `BITFINEX_SECRET_KEY`: 交易凭证
- `CURRENCIES`: 多币种配置块，每个币种包含独立的策略参数
- `MIN_DAILY_LEND_RATE`: 最低可接受放贷利率（安全阈值）
- `ENABLE_SMART_STRATEGY`: 启用智能策略（否则使用 K 线策略）
- `HIGH_HOLD_*`: 大额资金的高利率持有策略
- `TEST_MODE`: 测试模式开关（true=不执行真实交易）
- `API_ENABLED`: 启用 Web API 和界面（默认 true）

## 项目结构

核心入口点和模块：
- `main.go` - 应用程序入口，`Application` 结构体协调所有组件
- `internal/currency/manager.go` - `CurrencyManager` 管理多币种 LendingBot 实例
- `internal/strategy/lending.go` - `LendingBot` 核心放贷策略引擎
- `internal/strategy/smart_strategy.go` - `SmartStrategy` 智能策略实现（集成 LLM 预测）
- `internal/strategy/llm_ai_strategy.go` - `LLMAIStrategy` LLM AI 利率预测（三种提示词方案）
- `internal/strategy/market_analyzer.go` - `MarketAnalyzer` 市场深度分析
- `internal/bitfinex/client.go` - Bitfinex API v2 客户端封装
- `internal/api/handlers.go` - REST API 端点处理器

## 关键模块参考

**Bitfinex API 集成** (`internal/bitfinex/client.go`):
- `GetFundingOffers()` - 获取活跃的资金订单
- `CancelAllOffers()` - 取消所有资金订单
- `GetAvailableFunds()` - 获取钱包余额
- `GetLendingRate()` - 获取当前资金利率
- `GetFundingCandles()` - 获取历史 K 线数据
- `SubmitFundingOffer()` - 提交放贷订单

**Web API 端点** (`internal/api/handlers.go`):
- `/api/status` - 机器人状态
- `/api/earnings` - 收益统计
- `/api/offers` - 活跃订单
- `/api/config` - 配置管理（GET/POST）
- `/api/control` - 机器人控制（启动/停止/重启）
- `/health` - 健康检查

## 测试

测试文件位于各模块目录下（`*_test.go`）。运行 `make test` 生成覆盖率报告（`coverage.html`）。

## V2 API 注意事项

- 资金符号使用 "f" 前缀（例如 "fUSD" 而不是 "USD"）
- 利率值是日利率（不是年化利率）
- 资金订单通过 `client.Funding.Offers(symbol)` 访问
- 钱包余额通过 `client.Wallet.Wallet()` 访问

## 开发技巧

- 使用 `make dev` 进行开发，启用测试模式
- 提交更改前使用 `make test`
- 使用 `make security-check` 验证提交中无敏感数据
- Docker 镜像: `apexlgf/bitfinexwebbot:latest`

## 应用程序架构

**并发模型**: 主应用程序使用 `context.Context` 和 `sync.WaitGroup` 管理多个 goroutine，支持优雅关闭。

**三个独立调度器**:
1. **主任务** (`MINUTES_RUN`): 执行放贷策略、下单、取消订单
2. **借贷检查** (`LENDING_CHECK_MINUTES`): 检查新借贷成交，记录收益
3. **利率监控** (每小时): 检查利率阈值，发送 Telegram 提醒

## 安全注意事项

- API 凭证存储在 `config.yaml` 中（确保不被提交到版本控制）
- Web API 支持可选的认证令牌（`API_AUTH_TOKEN`）
- 测试模式（`TEST_MODE: true`）防止真实交易操作
- 生产环境建议使用 HTTPS 和防火墙规则限制 8089 端口访问

## 两种核心放贷策略

**CalculateSmartOffers (智能策略)** - `ENABLE_SMART_STRATEGY: true`
- 市场驱动型：基于实时市场深度数据和竞争分析
- 自适应算法：根据市场波动率、趋势动态调整利率和资金配置
- 智能期间选择：根据市场趋势决定短期或长期放贷

**calculateKlineOffers (K线策略)** - `ENABLE_SMART_STRATEGY: false`
- 技术分析导向：基于历史 K 线数据（支持 SMA、EMA、HLA、P90 等平滑方法）
- 趋势跟随：通过历史价格模式预测走势
- 固定期间选择：基于利率阈值选择 2/30/120 天期间

智能策略适合复杂多变的市场，K 线策略适合趋势明确的市场环境。

## LLM AI 利率预测策略

**功能概述**: 使用 OpenAI 兼容 API 预测未来 4 小时的放贷利率走势，替代合成利率算法。

**三种提示词方案**:
1. `StrategySimple` (方案一): 结构化数据 + 简洁提示词（~500 tokens）
2. `StrategyDetailed` (方案二): 完整市场深度 + 多维分析（~1500 tokens）
3. `StrategyChainOfThought` (方案三): 时序特征 + 思维链推理（~800 tokens）

**配置参数**:
- `OPENAI_API_KEY`: OpenAI API Key（必需）
- `OPENAI_MODEL`: 模型名称（默认 gpt-4o）
- `OPENAI_BASE_URL`: API Base URL（默认 https://api.openai.com/v1）
- `ENABLE_LLM_STRATEGY`: 启用 LLM 策略替代合成利率（默认 false）
- `LLM_DEFAULT_STRATEGY`: 默认策略类型 1/2/3（默认 1）
- `LLM_CACHE_HOURS`: 预测缓存时间（默认 3 小时）
- `LLM_MAX_RETRIES`: 最大重试次数（默认 3 次）
- `LLM_TIMEOUT_SECONDS`: API 超时时间（默认 30 秒）

**使用场景**: 当市场数据不足（fundingBook 为空、利率差距过小等）时，LLM 预测替代合成利率算法。

**测试命令**:
```bash
go test ./internal/strategy/... -v -run TestLLMAIStrategy      # 测试 LLM 策略
go test ./internal/strategy/... -v -run TestSmartStrategyLLM   # 测试 LLM 整合
```

## Web 界面和 Docker 部署

**Docker 架构**: 单容器多服务（Nginx 8089 + Go 应用 8090 + Supervisor）

**快速部署**:
```bash
docker compose up -d              # 启动服务
docker compose logs -f            # 查看日志
```

访问 http://localhost:8089 使用 Web 控制台（状态监控、收益统计、订单管理、配置编辑）。