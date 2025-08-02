# BitfinexBot 生产环境部署指南

本指南将帮助您将 BitfinexBot 集成到现有的生产环境中。

## 前提条件

- Docker 和 Docker Compose 已安装
- 现有的 nginx、php-fpm、mysql 环境正在运行
- Bitfinex API 密钥和密钥已准备就绪
- 基本的 Linux 系统管理知识

## 部署步骤

### 1. 准备文件结构

在您的生产服务器上创建以下目录结构：

```bash
mkdir -p /opt/bitfinex-deployment
cd /opt/bitfinex-deployment

# 创建必要的目录
mkdir -p bitfinex-bot/{logs,database,web}
mkdir -p docker-scripts
mkdir -p nginx/conf.d
```

### 2. 复制配置文件

将以下文件复制到相应位置：

**2.1 复制 docker-compose 配置**
```bash
# 将 production-docker-compose.yml 复制为 docker-compose.yml
cp production-docker-compose.yml docker-compose.yml
```

**2.2 复制数据库文件**
```bash
# 复制数据库初始化脚本
cp database/schema.sql bitfinex-bot/database/
cp database/mysql.cnf bitfinex-bot/database/
cp docker-scripts/create-multiple-databases.sh docker-scripts/
chmod +x docker-scripts/create-multiple-databases.sh
```

**2.3 复制 Web 界面文件**
```bash
# 复制整个 web 目录
cp -r web/* bitfinex-bot/web/
```

**2.4 复制 Nginx 配置**
```bash
# 复制 Nginx 配置文件
cp nginx-config/bitfinex-bot.conf nginx/conf.d/
```

**2.5 创建 BitfinexBot 配置文件**
```bash
# 复制并编辑配置文件
cp production-config.yaml bitfinex-bot/config.yaml
```

### 3. 配置 BitfinexBot

编辑 `bitfinex-bot/config.yaml` 文件，设置以下关键参数：

```yaml
# 必须配置的参数
bitfinex:
  api_key: "YOUR_BITFINEX_API_KEY"     # 替换为你的 API Key
  secret_key: "YOUR_BITFINEX_SECRET"   # 替换为你的 Secret Key

# 数据库连接（保持默认，除非你修改了数据库密码）
database:
  dsn: "bitfinex_bot:BitfinexBot@2025@tcp(mysql:3306)/bitfinex_bot_db?charset=utf8mb4&parseTime=True&loc=Local"

# 生产环境设置
execution:
  test_mode: false  # 重要：生产环境必须设为 false

# 根据需要调整策略参数
strategy:
  min_daily_lend_rate: 0.0001  # 最低日利率
  max_daily_lend_rate: 0.1     # 最高日利率
  order_limit: 10              # 每次最大订单数
```

### 4. 更新现有的 docker-compose.yml

**方法一：完全替换（推荐）**
```bash
# 停止现有服务
docker compose down

# 备份现有配置
cp docker-compose.yml docker-compose.yml.backup

# 使用新的配置文件
cp production-docker-compose.yml docker-compose.yml
```

**方法二：手动合并**
如果您有自定义配置，请手动将 BitfinexBot 相关的服务添加到您现有的 docker-compose.yml 中：

```yaml
# 在现有的 docker-compose.yml 中添加以下服务
services:
  # ... 您现有的服务 ...

  # 添加 BitfinexBot 服务
  bitfinex-bot:
    image: apexlgf/bitfenix-bot:latest
    container_name: bitfinex-bot
    restart: unless-stopped
    volumes:
      - ./bitfinex-bot/config.yaml:/app/config.yaml
      - ./bitfinex-bot/logs:/app/logs
    depends_on:
      mysql:
        condition: service_healthy
    networks:
      - default
    environment:
      - CONFIG_PATH=/app/config.yaml
      - TZ=Asia/Shanghai
    command: ["./bitfinex-bot", "-c", "/app/config.yaml"]
```

### 5. 更新 MySQL 配置

**5.1 修改 MySQL 环境变量**
在 docker-compose.yml 中的 mysql 服务中添加：
```yaml
mysql:
  environment:
    MYSQL_MULTIPLE_DATABASES: appdb,bitfinex_bot_db  # 添加这行
  volumes:
    # 添加以下两行
    - ./bitfinex-bot/database/schema.sql:/docker-entrypoint-initdb.d/bitfinex-schema.sql
    - ./docker-scripts/create-multiple-databases.sh:/docker-entrypoint-initdb.d/create-multiple-databases.sh
```

### 6. 更新 Nginx 配置

**6.1 添加 BitfinexBot Web 界面支持**
在 nginx 服务的 volumes 中添加：
```yaml
nginx:
  volumes:
    - ./bitfinex-bot/web:/var/www/bitfinex-bot  # 添加这行
    - ./nginx/conf.d:/etc/nginx/conf.d          # 确保这行存在
```

**6.2 更新 PHP-FPM 配置**
在 php-fpm 服务的 volumes 中添加：
```yaml
php-fpm:
  volumes:
    - ./bitfinex-bot/web:/var/www/bitfinex-bot  # 添加这行
```

### 7. 部署和启动

**7.1 拉取最新镜像**
```bash
docker pull apexlgf/bitfenix-bot:latest
```

**7.2 启动服务**
```bash
# 启动所有服务
docker compose up -d

# 检查服务状态
docker compose ps

# 查看 BitfinexBot 日志
docker compose logs -f bitfinex-bot
```

**7.3 验证部署**
```bash
# 检查数据库是否创建成功
docker compose exec mysql mysql -u root -prootpassword -e "SHOW DATABASES;"

# 检查 BitfinexBot 数据库表
docker compose exec mysql mysql -u bitfinex_bot -p'BitfinexBot@2025' bitfinex_bot_db -e "SHOW TABLES;"

# 访问 Web 界面
curl http://localhost/bitfinex/
```

### 8. 访问和管理

**8.1 Web 界面访问**
- 主要应用：`http://your-server/`
- BitfinexBot 管理界面：`http://your-server/bitfinex/`
- phpMyAdmin：`http://your-server:8081`
- Web 终端：`http://your-server:3000`

**8.2 默认登录信息**
- Web 管理界面用户名：`admin`
- 默认密码：`admin123`
- 数据库root密码：`rootpassword`

### 9. 监控和维护

**9.1 日志查看**
```bash
# BitfinexBot 日志
docker compose logs -f bitfinex-bot

# 所有服务日志
docker compose logs

# 查看日志文件
tail -f bitfinex-bot/logs/bitfinex-bot.log
```

**9.2 服务管理**
```bash
# 重启 BitfinexBot
docker compose restart bitfinex-bot

# 停止所有服务
docker compose down

# 更新 BitfinexBot
docker compose pull bitfinex-bot
docker compose up -d bitfinex-bot
```

**9.3 数据备份**
```bash
# 备份数据库
docker compose exec mysql mysqldump -u root -prootpassword bitfinex_bot_db > bitfinex_backup_$(date +%Y%m%d).sql

# 备份配置文件
tar -czf bitfinex_config_backup_$(date +%Y%m%d).tar.gz bitfinex-bot/config.yaml bitfinex-bot/logs/
```

## 故障排除

### 常见问题

**1. BitfinexBot 无法启动**
```bash
# 检查配置文件
docker compose exec bitfinex-bot cat /app/config.yaml

# 检查日志
docker compose logs bitfinex-bot
```

**2. 数据库连接失败**
```bash
# 检查数据库是否运行
docker compose ps mysql

# 测试数据库连接
docker compose exec mysql mysql -u bitfinex_bot -p'BitfinexBot@2025' bitfinex_bot_db
```

**3. Web 界面无法访问**
```bash
# 检查 Nginx 配置
docker compose exec nginx nginx -t

# 检查文件权限
ls -la bitfinex-bot/web/
```

**4. API 连接问题**
- 确认 Bitfinex API 密钥正确
- 检查网络连接
- 查看 BitfinexBot 日志中的具体错误信息

### 性能优化

**1. 资源分配**
根据服务器配置调整以下参数：
- MySQL `innodb_buffer_pool_size`
- BitfinexBot `execution_interval`
- 日志文件大小限制

**2. 监控设置**
- 启用健康检查
- 设置日志轮转
- 配置磁盘空间监控

## 安全注意事项

1. **API 密钥安全**
   - 确保配置文件权限正确（600）
   - 定期轮换 API 密钥
   - 监控 API 使用情况

2. **网络安全**
   - 使用防火墙限制端口访问
   - 考虑使用 HTTPS
   - 定期更新密码

3. **数据安全**
   - 定期备份数据库
   - 监控异常登录
   - 定期检查日志

## 升级和更新

定期检查更新：
```bash
# 拉取最新镜像
docker pull apexlgf/bitfenix-bot:latest

# 重启服务应用更新
docker compose up -d bitfinex-bot
```

---

如有问题，请查看项目文档或联系技术支持。