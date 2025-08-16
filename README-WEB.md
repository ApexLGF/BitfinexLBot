# BitfinexBot Web 版本使用指南

BitfinexBot 现已支持 Web 界面管理，通过现代化的 REST API 和响应式 Web 界面替代了原有的 Telegram Bot 功能。

## 🌟 新特性

- **Web 界面管理** - 美观的响应式 Web 控制台
- **REST API** - 完整的 RESTful API 支持
- **实时监控** - 实时状态监控和数据更新
- **配置管理** - 在线配置编辑和保存
- **Docker 一体化** - 单个 Docker 镜像包含所有服务

## 🚀 快速开始

### 1. 准备配置文件

复制配置示例并编辑：
```bash
cp config.yaml.example config.yaml
# 编辑 config.yaml，设置您的 Bitfinex API 密钥
```

### 2. 使用 Docker 运行

#### 使用 Docker Compose（推荐）
```bash
# 启动服务
docker compose up -d

# 查看日志
docker compose logs -f

# 停止服务
docker compose down
```

#### 使用 Docker 直接运行
```bash
# 构建镜像
docker build -t bitfinex-bot .

# 运行容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  bitfinex-bot
```

### 3. 访问 Web 界面

打开浏览器访问：http://localhost:8089

## 🖥️ Web 界面功能

### 控制台面板
- **机器人状态** - 实时显示运行状态、可用资金、活跃订单等
- **控制按钮** - 启动、停止、重启机器人
- **状态指示器** - 直观的状态指示和连接状态

### 收益监控
- **收益统计** - 日、周、月收益汇总
- **收益图表** - 可视化收益趋势图表
- **历史数据** - 详细的收益历史记录

### 订单管理
- **活跃订单** - 查看当前所有放贷订单
- **订单详情** - 订单ID、金额、利率、期限等信息
- **实时更新** - 订单状态实时更新

### 配置管理
- **在线编辑** - 直接在Web界面编辑配置
- **实时保存** - 配置更改立即生效
- **参数验证** - 配置参数格式验证

### 系统监控
- **实时日志** - 系统运行日志实时显示
- **系统信息** - 版本信息、运行时间等
- **健康检查** - 系统健康状态监控

## 🔧 API 接口

BitfinexBot 提供完整的 REST API，支持所有 Web 界面功能：

### 核心端点
- `GET /api/status` - 获取机器人状态
- `GET /api/earnings` - 获取收益数据
- `GET /api/offers` - 获取放贷订单
- `GET /api/config` - 获取配置信息
- `POST /api/config` - 更新配置
- `POST /api/control` - 控制机器人（启动/停止/重启）
- `GET /api/logs` - 获取系统日志
- `GET /health` - 健康检查

### API 使用示例
```bash
# 获取机器人状态
curl http://localhost:8089/api/status

# 启动机器人
curl -X POST http://localhost:8089/api/control \
  -H "Content-Type: application/json" \
  -d '{"action": "start"}'

# 获取收益数据
curl http://localhost:8089/api/earnings
```

## ⚙️ 配置说明

### API 服务配置
```yaml
# API 服务配置
API_ENABLED: true         # 启用REST API和Web界面
API_PORT: 8090            # API服务端口（内部端口）
API_HOST: "127.0.0.1"     # 绑定地址
API_CORS_ORIGINS: ["*"]   # CORS允许的源
API_AUTH_TOKEN: ""        # API认证令牌（可选）
```

### 重要配置项
- **API_ENABLED**: 控制是否启用Web API功能
- **API_PORT**: API服务内部端口（默认8090，Nginx代理到8089）
- **API_HOST**: API服务绑定地址（容器内建议使用127.0.0.1）
- **TEST_MODE**: 测试模式开关（true=不执行真实交易）

## 🐳 Docker 架构

BitfinexBot Web 版本采用单容器多服务架构：

### 服务组件
- **Go 应用程序** - 核心放贷逻辑和 REST API 服务（端口8090）
- **Nginx** - Web 服务器和反向代理（端口8089）
- **Supervisor** - 进程管理和监控

### 端口说明
- **8089** - 对外Web访问端口（Nginx）
- **8090** - 内部API服务端口（Go应用）

### 目录结构
```
/app/
├── bitfinex-bot          # Go应用程序二进制文件
├── config.yaml           # 配置文件（需要挂载）
├── logs/                 # 应用日志目录
└── entrypoint.sh         # 启动脚本

/usr/share/nginx/html/    # Web静态文件目录
├── index.html            # 主页面
└── static/               # 静态资源
    ├── css/
    ├── js/
    └── img/
```

## 🛠️ 开发指南

### 本地开发
```bash
# 安装依赖
go mod tidy

# 运行程序
go run . -c config.yaml

# 构建程序
go build -o bitfinex-bot .

# 在单独终端启动Web服务器（开发模式）
cd web && python -m http.server 8089
```

### 构建 Docker 镜像
```bash
# 构建镜像
docker build -t bitfinex-bot:latest .

# 推送到仓库
docker tag bitfinex-bot:latest your-registry/bitfinex-bot:latest
docker push your-registry/bitfinex-bot:latest
```

## 🔒 安全注意事项

1. **API 密钥安全** - 确保 `config.yaml` 文件权限正确，不要提交到版本控制
2. **网络访问** - 建议在生产环境中限制访问来源
3. **HTTPS** - 生产环境建议使用 HTTPS 和反向代理
4. **防火墙** - 配置适当的防火墙规则

## 📊 监控和日志

### 日志文件
- **应用日志**: `/app/logs/` 目录
- **Nginx日志**: `/var/log/nginx/`
- **Supervisor日志**: `/var/log/supervisor/`

### 健康检查
- **Web健康检查**: `http://localhost:8089/health`
- **Docker健康检查**: 自动检查服务状态
- **Supervisor监控**: 自动重启异常服务

## ❓ 故障排除

### 常见问题

1. **无法访问Web界面**
   - 检查端口8089是否被占用
   - 确认Docker容器正常运行
   - 查看Nginx错误日志

2. **API请求失败**
   - 检查Go应用程序是否正常运行
   - 验证配置文件格式
   - 查看应用程序日志

3. **配置更新失败**
   - 确认配置文件权限
   - 检查配置文件格式
   - 重启应用程序

### 调试命令
```bash
# 查看容器日志
docker logs bitfinex-bot-web

# 进入容器调试
docker exec -it bitfinex-bot-web sh

# 检查服务状态
docker exec bitfinex-bot-web supervisorctl status

# 重启服务
docker exec bitfinex-bot-web supervisorctl restart all
```

## 🔄 从 Telegram Bot 迁移

如果您之前使用 Telegram Bot 版本，Web 版本提供了所有相同的功能：

| Telegram 命令 | Web 界面功能 |
|--------------|------------|
| `/status` | 控制台状态面板 |
| `/start` | 启动按钮 |
| `/stop` | 停止按钮 |
| `/restart` | 重启按钮 |
| `/earnings` | 收益统计面板 |
| `/offers` | 活跃订单表格 |
| `/config` | 配置管理模态框 |

Web 版本具有更好的用户体验、实时更新和可视化图表等优势。

## 📝 版本历史

- **v2.0.0** - 引入Web界面和REST API，移除Telegram Bot依赖
- **v1.x** - 基于Telegram Bot的版本

---

如需更多帮助，请参考项目文档或提交Issue。