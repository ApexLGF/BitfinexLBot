# BitfinexWebBot Docker镜像使用说明

## 镜像信息
- **Docker Hub**: `apexlgf/bitfinexwebbot:latest`
- **镜像大小**: 125MB
- **基础镜像**: nginx:alpine + golang:1.23-alpine
- **架构**: 多阶段构建，生产优化

## 快速启动

### 1. 拉取镜像
```bash
docker pull apexlgf/bitfinexwebbot:latest
```

### 2. 创建配置文件
创建 `config.yaml` 文件（参考项目中的 config.yaml.example）：

```yaml
BITFINEX_API_KEY: "your_api_key_here"
BITFINEX_SECRET_KEY: "your_secret_key_here" 
CURRENCY: "USD"
MIN_DAILY_LEND_RATE: 0.00028
# ... 其他配置
```

### 3. 运行容器
```bash
docker run -d \
  --name bitfinex-webbot \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  apexlgf/bitfinexwebbot:latest
```

### 4. 访问Web界面
打开浏览器访问: `http://localhost:8089`

## 详细配置

### 使用Docker Compose
创建 `docker-compose.yml`：

```yaml
services:
  bitfinex-webbot:
    image: apexlgf/bitfinexwebbot:latest
    container_name: bitfinex-webbot
    ports:
      - "8089:8089"
    volumes:
      - ./config.yaml:/app/config.yaml:rw
      - bitfinex_logs:/app/logs
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8089/health"]
      interval: 30s
      timeout: 10s
      start_period: 60s
      retries: 3

volumes:
  bitfinex_logs:
    driver: local
```

启动：
```bash
docker-compose up -d
```

## 端口说明
- **8089**: Web界面和API端口（nginx前端 + Go后端API）
- **8090**: Go应用内部API端口（容器内部，通过nginx代理）

## 自定义端口配置

### 方法1: 修改Docker端口映射
如果要改变外部访问端口，只需修改docker运行命令：

```bash
# 映射到其他端口（例如80端口）
docker run -d \
  --name bitfinex-webbot \
  -p 80:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:rw \
  apexlgf/bitfinexwebbot:latest

# 或者映射到8080端口
docker run -d \
  --name bitfinex-webbot \
  -p 8080:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:rw \
  apexlgf/bitfinexwebbot:latest
```

### 方法2: 修改nginx配置
如果需要修改容器内部端口，需要自定义nginx配置：

1. 创建自定义nginx配置文件 `custom-nginx.conf`：
```nginx
server {
    listen 80 default_server;  # 修改为所需端口
    listen [::]:80 default_server;
    
    server_name _;
    root /usr/share/nginx/html;
    index index.html;
    
    # API 代理到 Go 应用程序  
    location /api/ {
        proxy_pass http://127.0.0.1:8090/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # 健康检查端点
    location /health {
        proxy_pass http://127.0.0.1:8090/health;
    }
    
    # 静态文件
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

2. 挂载自定义配置并修改端口映射：
```bash
docker run -d \
  --name bitfinex-webbot \
  -p 80:80 \
  -v $(pwd)/config.yaml:/app/config.yaml:rw \
  -v $(pwd)/custom-nginx.conf:/etc/nginx/conf.d/default.conf \
  apexlgf/bitfinexwebbot:latest
```

### Docker Compose端口配置
在docker-compose.yml中修改端口映射：

```yaml
services:
  bitfinex-webbot:
    image: apexlgf/bitfinexwebbot:latest
    ports:
      - "80:8089"      # 外部80端口映射到容器8089端口
      # 或者
      - "8080:8089"    # 外部8080端口映射到容器8089端口
    # ... 其他配置
```

## API端点
- `GET /health` - 健康检查
- `GET /api/status` - 机器人状态
- `GET /api/earnings` - 收益数据
- `GET /api/offers` - 放贷订单
- `GET /api/credits` - 已贷出订单
- `GET /api/logs` - 系统日志

## 数据持久化
- `/app/logs` - 应用日志目录
- `/app/config.yaml` - 配置文件（建议只读挂载）

## 环境变量
- `TZ` - 时区设置（默认: Asia/Shanghai）

## 日志查看
```bash
# 查看容器日志
docker logs bitfinex-webbot

# 查看应用日志
docker exec bitfinex-webbot tail -f /app/logs/bitfinex-bot.err.log
```

## 安全建议
1. 配置文件使用读写挂载（`:rw`）
2. 不要在环境变量中存储API密钥
3. 使用Docker secrets管理敏感信息（生产环境）
4. 定期更新镜像版本

## 故障排除
1. 检查配置文件格式是否正确
2. 确认API密钥有效且有足够权限
3. 查看容器日志排查启动问题
4. 访问 `/health` 端点检查服务状态

## 构建信息
- **构建时间**: 最新版本
- **Go版本**: 1.23
- **包含修复**: API解析、智能策略、数组越界等关键问题
- **功能**: Web界面、智能放贷、实时监控、收益统计