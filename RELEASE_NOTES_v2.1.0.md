# BitfinexWebBot v2.1.0 发布说明

## 🚀 Docker Hub 镜像

```bash
# 拉取最新版本
docker pull apexlgf/bitfinexwebbot:v2.1.0
docker pull apexlgf/bitfinexwebbot:latest

# 运行容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v ./config.yaml:/app/config.yaml \
  -v bitfinex_logs:/app/logs \
  apexlgf/bitfinexwebbot:v2.1.0
```

## 🌟 主要更新 (v2.1.0)

### 💰 总资金显示功能
- **新增**: 钱包API端点 `/api/wallets`
- **改进**: 页面显示真实总资金而不是活跃订单数量
- **计算**: 总资金 = 可用资金 + 已贷出订单总额 + 贷出挂单总额
- **支持**: 多钱包类型和货币格式化

### 🎨 界面优化
- **简化**: 移除启动/停止按钮，只保留重启功能
- **移动**: 手动刷新按钮移至机器人操作区域
- **修复**: 刷新按钮样式问题（浅色背景下不可见）
- **更新**: 收益统计标签更直观
  - "今日收益" → "昨日收益"
  - "本周收益" → "7日收益"  
  - "本月收益" → "30日收益"

### 🔧 技术改进
- **修复**: UST等非标准货币的格式化显示
- **增强**: 一键刷新所有界面数据功能
- **优化**: JavaScript异步数据获取和错误处理
- **更新**: 静态资源版本号，确保浏览器缓存更新

### 🛠 API 增强
- **新增**: `GetWallets` 处理器，支持完整钱包信息获取
- **修复**: 货币格式化函数，正确处理非ISO货币代码
- **改进**: 错误处理和容错机制

## 📋 修复的问题

- ✅ 总资金显示错误（之前显示活跃订单数量）
- ✅ 刷新按钮在浅色背景下不可见
- ✅ 非标准货币代码（UST/USDT）格式化异常
- ✅ 数据源同步更新问题

## 🔄 升级指南

### 从 v1.0.0 升级到 v2.1.0

1. **停止现有容器**
   ```bash
   docker stop bitfinex-bot-web
   docker rm bitfinex-bot-web
   ```

2. **拉取新镜像**
   ```bash
   docker pull apexlgf/bitfinexwebbot:v2.1.0
   ```

3. **启动新容器**
   ```bash
   docker run -d \
     --name bitfinex-bot-web \
     -p 8089:8089 \
     -v ./config.yaml:/app/config.yaml \
     -v bitfinex_logs:/app/logs \
     apexlgf/bitfinexwebbot:v2.1.0
   ```

### Docker Compose 用户

更新 `docker-compose.yml`:
```yaml
services:
  bitfinex-bot:
    image: apexlgf/bitfinexwebbot:v2.1.0
    # ... 其他配置保持不变
```

然后运行：
```bash
docker compose pull
docker compose up -d
```

## 🌐 访问地址

- **Web界面**: http://localhost:8089
- **API接口**: http://localhost:8089/api/
- **健康检查**: http://localhost:8089/health
- **钱包信息**: http://localhost:8089/api/wallets (新增)

## 📊 技术规格

- **基础镜像**: nginx:alpine + golang:1.23-alpine
- **镜像大小**: ~125MB
- **架构支持**: ARM64/AMD64
- **时区**: Asia/Shanghai
- **端口**: 8089 (Web + API)

## 🔗 相关链接

- **Docker Hub**: https://hub.docker.com/r/apexlgf/bitfinexwebbot
- **GitHub**: https://github.com/ApexLGF/BitfinexLBot
- **分支**: BitfinexLBot2

---

*🤖 Generated with [Claude Code](https://claude.ai/code)*