# BitfinexWebBot v2.2.0 发布说明

## 🚀 Docker Hub 镜像

```bash
# 拉取最新版本
docker pull apexlgf/bitfinexwebbot:v2.2.0
docker pull apexlgf/bitfinexwebbot:latest

# 运行容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v ./config.yaml:/app/config.yaml \
  -v bitfinex_logs:/app/logs \
  apexlgf/bitfinexwebbot:v2.2.0
```

## 🌟 主要更新 (v2.2.0)

### 🔄 真正的重启功能
- **重构**: 完全重写了Web界面重启按钮的后端逻辑
- **热重载**: 重启时自动重新加载配置文件 `config.yaml`
- **智能检测**: 根据配置变化类型选择相应的重启策略
  - API密钥变化：重新初始化Bitfinex客户端
  - 普通配置变化：仅更新策略配置
- **安全回滚**: 配置加载失败时自动回滚到旧配置
- **执行验证**: 重启后立即执行一次策略验证新配置

### 📋 配置重载逻辑

#### 关键配置变化（需重新初始化客户端）
- `BITFINEX_API_KEY` - Bitfinex API密钥
- `BITFINEX_SECRET_KEY` - Bitfinex私钥
- `CURRENCY` - 交易货币

#### 普通配置变化（热更新）
- `MIN_DAILY_LEND_RATE` - 最小放贷利率
- `MINUTES_RUN` - 运行间隔
- `ORDER_LIMIT` - 订单限制
- `SPREAD_LEND` - 资金分散数
- `GAP_BOTTOM` / `GAP_TOP` - 深度范围
- 智能策略相关参数
- 其他所有业务配置

### 🎨 界面改进
- **按钮文本**: "重启" → "重启机器人"，更明确的功能说明
- **工具提示**: 添加详细的功能描述
- **状态反馈**: 重启过程中显示加载动画和进度提示
- **自动刷新**: 重启成功后自动刷新所有界面数据

### 🔧 技术改进
- **新增方法**: `LendingBot.UpdateConfig()` - 策略配置热更新
- **新增方法**: `Handler.restartBot()` - 完整的重启逻辑
- **新增方法**: `Handler.hasConfigChanged()` - 智能配置变化检测
- **错误处理**: 完善的错误捕获和状态回滚机制
- **日志增强**: 详细的重启过程日志记录

## 📈 重启功能对比

### v2.1.x 及之前版本
- ❌ 配置文件不会重新加载
- ❌ 组件不会重新初始化  
- ❌ 只执行一次策略
- ❌ 修改配置需要重启容器

### v2.2.0 新版本
- ✅ **自动重新加载配置文件**
- ✅ **智能重新初始化组件**
- ✅ **配置验证和错误回滚**
- ✅ **真正的热重载功能**

## 🛠 使用场景

### 配置修改后重启
1. 修改 `config.yaml` 文件
2. 在Web界面点击"重启机器人"按钮
3. 系统自动：
   - 重新读取配置文件
   - 检测配置变化类型
   - 重新初始化必要组件
   - 验证新配置可用性
   - 执行一次策略测试

### 支持的配置热更新
```yaml
# 这些配置修改后可以通过Web重启立即生效
MIN_DAILY_LEND_RATE: 0.025      # ✅ 热更新
MINUTES_RUN: 5                  # ✅ 热更新  
ORDER_LIMIT: 15                 # ✅ 热更新
SPREAD_LEND: 25                 # ✅ 热更新
ENABLE_SMART_STRATEGY: false    # ✅ 热更新
VOLATILITY_THRESHOLD: 0.003     # ✅ 热更新

# 这些配置修改需要重新初始化客户端
BITFINEX_API_KEY: "new_key"     # ✅ 重新初始化
CURRENCY: "UST"                 # ✅ 重新初始化
```

## 🔄 升级指南

### 从 v2.1.x 升级到 v2.2.0

1. **停止现有容器**
   ```bash
   docker stop bitfinex-bot-web
   docker rm bitfinex-bot-web
   ```

2. **拉取新镜像**
   ```bash
   docker pull apexlgf/bitfinexwebbot:v2.2.0
   ```

3. **启动新容器**
   ```bash
   docker run -d \
     --name bitfinex-bot-web \
     -p 8089:8089 \
     -v ./config.yaml:/app/config.yaml \
     -v bitfinex_logs:/app/logs \
     apexlgf/bitfinexwebbot:v2.2.0
   ```

4. **测试重启功能**
   - 访问 http://localhost:8089
   - 修改配置文件中的任意参数
   - 点击"重启机器人"按钮
   - 观察日志确认配置重新加载

## 📊 功能验证

### 重启日志示例
```
[API] 开始重启机器人...
[API] 重新加载配置文件: /app/config.yaml
[API] 检测到配置变化，重新初始化客户端和策略
[API] 重新初始化Bitfinex客户端
[API] 重新创建放贷策略实例
[API] 执行一次策略验证新配置
[Strategy] 更新配置...
[Strategy] 配置更新完成
開始執行貸出機器人...
[API] 机器人重启成功
```

## 🌐 访问地址

- **Web界面**: http://localhost:8089
- **API接口**: http://localhost:8089/api/
- **重启接口**: POST http://localhost:8089/api/control `{"action": "restart"}`

## 🔗 相关链接

- **Docker Hub**: https://hub.docker.com/r/apexlgf/bitfinexwebbot
- **GitHub**: https://github.com/ApexLGF/BitfinexLBot  
- **分支**: BitfinexLBot2

---

*🤖 Generated with [Claude Code](https://claude.ai/code)*