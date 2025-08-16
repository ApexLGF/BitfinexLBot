# BitfinexBot Web版本部署指南

## 🎯 部署概述

BitfinexBot Web版本提供了一个现代化的Web界面管理系统，替代了原有的Telegram Bot功能。该版本使用Docker一体化部署，包含Go应用程序、Nginx Web服务器和完整的前端界面。

## 📋 部署前准备

### 系统要求
- **Docker**: >= 20.10
- **Docker Compose**: >= 2.0
- **内存**: 至少 512MB
- **存储**: 至少 1GB 可用空间
- **网络**: 需要访问 Bitfinex API (api.bitfinex.com)

### 端口要求
- **8089**: Web界面访问端口（可配置）
- **8090**: 内部API服务端口（容器内部）

## 🚀 快速部署

### 1. 获取项目代码
```bash
git clone https://github.com/ApexLGF/BitfinexLBot.git
cd BitfinexLBot
git checkout BitfinexLBot2  # 切换到Web版本分支
```

### 2. 配置文件准备
```bash
# 复制配置模板
cp config.yaml.example config.yaml

# 编辑配置文件
nano config.yaml
```

**必须配置的项目**：
```yaml
# Bitfinex API 凭证（必须）
BITFINEX_API_KEY: "your_api_key_here"
BITFINEX_SECRET_KEY: "your_secret_key_here"

# 交易配置
CURRENCY: "USD"
TEST_MODE: true  # 生产环境设为 false

# Web API 配置
API_ENABLED: true
API_PORT: 8090
API_HOST: "127.0.0.1"
```

### 3. 启动服务
```bash
# 使用 Docker Compose 启动
docker compose up -d

# 查看启动日志
docker compose logs -f
```

### 4. 访问Web界面
打开浏览器访问：http://localhost:8089

## 🔧 详细配置

### 配置文件详解

#### API服务配置
```yaml
# API 服务配置
API_ENABLED: true         # 启用Web API功能
API_PORT: 8090            # 内部API端口
API_HOST: "127.0.0.1"     # 绑定地址（容器内部）
API_CORS_ORIGINS: ["*"]   # CORS设置
API_AUTH_TOKEN: ""        # 可选的API认证令牌
```

#### 放贷策略配置
```yaml
# 基本策略
MIN_DAILY_LEND_RATE: 0.003200   # 最低日利率
SPREAD_LEND: 12                 # 资金分散数
ORDER_LIMIT: 20                 # 最大订单数

# 高级策略
ENABLE_SMART_STRATEGY: true     # 启用智能策略
HIGH_HOLD_RATE: 0.060          # 高额持有利率
HIGH_HOLD_AMOUNT: 2000         # 高额持有金额
```

### Docker Compose 配置

#### 基本配置
```yaml
version: '3.8'
services:
  bitfinex-bot:
    build: .
    ports:
      - "8089:8089"
    volumes:
      - ./config.yaml:/app/config.yaml:rw
      - bitfinex_logs:/app/logs
    restart: unless-stopped
```

#### 生产环境配置
```yaml
version: '3.8'
services:
  bitfinex-bot:
    image: your-registry/bitfinex-bot:latest
    ports:
      - "8089:8089"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - bitfinex_logs:/app/logs
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8089/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## 🔄 部署模式

### 开发环境部署
```bash
# 构建并启动
docker compose up --build

# 查看实时日志
docker compose logs -f bitfinex-bot

# 重新构建
docker compose down
docker compose build --no-cache
docker compose up -d
```

### 生产环境部署
```bash
# 拉取预构建镜像
docker pull your-registry/bitfinex-bot:latest

# 使用生产配置启动
docker compose -f docker-compose.prod.yml up -d

# 配置自动重启
systemctl enable docker
```

### 集群部署
```bash
# 使用 Docker Swarm
docker swarm init
docker stack deploy -c docker-compose.swarm.yml bitfinex

# 使用 Kubernetes
kubectl apply -f k8s/
```

## 🔒 安全配置

### 1. 网络安全
```yaml
# 限制访问来源
networks:
  bitfinex-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### 2. 文件权限
```bash
# 设置配置文件权限
chmod 600 config.yaml
chown root:root config.yaml

# 设置日志目录权限
chmod 755 logs/
```

### 3. 反向代理（推荐）
```nginx
server {
    listen 443 ssl;
    server_name bitfinex-bot.yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8089;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 📊 监控和维护

### 健康检查
```bash
# 检查服务状态
curl http://localhost:8089/health

# 检查API响应
curl http://localhost:8089/api/status

# 检查容器状态
docker compose ps
```

### 日志管理
```bash
# 查看应用日志
docker compose logs bitfinex-bot

# 查看系统日志
docker exec bitfinex-bot-web tail -f /var/log/supervisor/bitfinex-bot.out.log

# 日志轮转配置
logrotate /etc/logrotate.d/bitfinex-bot
```

### 备份策略
```bash
# 备份配置文件
cp config.yaml config.yaml.backup.$(date +%Y%m%d)

# 备份日志数据
docker run --rm -v bitfinex_logs:/data -v $(pwd):/backup \
  alpine tar czf /backup/logs-backup-$(date +%Y%m%d).tar.gz /data

# 备份整个部署
tar czf bitfinex-deployment-$(date +%Y%m%d).tar.gz \
  config.yaml docker-compose.yml
```

## 🛠️ 故障排除

### 常见问题

#### 1. 无法访问Web界面
```bash
# 检查端口占用
netstat -tlnp | grep 8089

# 检查容器状态
docker compose ps

# 检查Nginx日志
docker compose logs bitfinex-bot | grep nginx
```

#### 2. API请求失败
```bash
# 检查Go应用状态
docker exec bitfinex-bot-web supervisorctl status

# 检查应用日志
docker compose logs bitfinex-bot | grep bitfinex-bot

# 测试内部API
docker exec bitfinex-bot-web curl http://localhost:8090/health
```

#### 3. 配置更新失败
```bash
# 检查配置文件格式
docker exec bitfinex-bot-web /app/bitfinex-bot -c /app/config.yaml --help

# 重启应用
docker exec bitfinex-bot-web supervisorctl restart bitfinex-bot

# 重载配置
docker compose restart
```

### 调试模式
```bash
# 启用详细日志
docker compose up --build -d
docker compose logs -f

# 进入容器调试
docker exec -it bitfinex-bot-web sh

# 查看进程状态
docker exec bitfinex-bot-web ps aux
```

## 📈 性能优化

### 资源配置
```yaml
# Docker资源限制
services:
  bitfinex-bot:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
        reservations:
          memory: 256M
          cpus: '0.25'
```

### 网络优化
```yaml
# 网络配置优化
networks:
  default:
    driver: bridge
    driver_opts:
      com.docker.network.driver.mtu: 1500
```

## 🔄 升级指南

### 滚动升级
```bash
# 拉取新版本
docker pull your-registry/bitfinex-bot:latest

# 优雅停止
docker compose down

# 更新并启动
docker compose up -d

# 验证升级
curl http://localhost:8089/api/info
```

### 回滚操作
```bash
# 回滚到上一版本
docker compose down
docker tag bitfinex-bot:latest bitfinex-bot:rollback
docker tag bitfinex-bot:previous bitfinex-bot:latest
docker compose up -d
```

## 📞 支持和帮助

### 获取帮助
- 📖 查看 [README-WEB.md](./README-WEB.md) 了解功能详情
- 🐛 提交 Issue 报告问题
- 💬 参与 Discussions 讨论

### 技术支持
- 确保您的问题包含完整的错误日志
- 提供配置文件（请移除敏感信息）
- 说明部署环境和Docker版本

---

🎉 **恭喜！您已成功部署 BitfinexBot Web版本**

现在您可以通过现代化的Web界面管理您的Bitfinex放贷策略了！