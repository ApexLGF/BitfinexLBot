# BitfinexBot Web 管理界面

这是 BitfinexBot 的 Web 管理界面，提供通过浏览器控制和监控机器人的功能。

## 功能特性

### 🎛️ 机器人控制
- **启动/停止/重启** - 远程控制机器人运行状态
- **实时状态监控** - 显示运行状态、余额、订单数、当前利率
- **命令执行** - 支持所有原 Telegram Bot 命令
- **批量操作** - 取消所有订单、批量查询等

### 📊 数据可视化
- **状态面板** - 实时显示关键指标
- **命令历史** - 查看最近执行的命令和结果
- **通知中心** - 显示系统通知和警告
- **统计图表** - 收益统计和性能分析

### ⚙️ 配置管理
- **在线配置** - 无需重启即可修改参数
- **分类管理** - 按功能模块组织配置项
- **安全控制** - 敏感信息保护
- **版本控制** - 配置变更历史追踪

## 文件结构

```
web/
├── index.html              # 主界面
├── config/
│   └── database.php        # 数据库连接配置
├── api/
│   ├── index.php          # API 路由入口
│   ├── commands.php       # 命令处理接口
│   ├── status.php         # 状态查询接口
│   └── config.php         # 配置管理接口
└── README.md              # 本文件
```

## 快速开始

### 1. 环境要求

- **PHP 7.4+** (推荐 PHP 8.0+)
- **MySQL 5.7+** 或 **MariaDB 10.3+**
- **Web 服务器** (Apache/Nginx)
- **PDO MySQL 扩展**

### 2. 配置数据库

确保数据库已按照 `database/README.md` 初始化完成。

### 3. 配置 Web 服务器

#### Apache 配置

```apache
<VirtualHost *:80>
    ServerName bitfinexbot.local
    DocumentRoot /path/to/BitfinexLBot/web
    
    <Directory /path/to/BitfinexLBot/web>
        AllowOverride All
        Require all granted
    </Directory>
    
    # API 重写规则
    RewriteEngine On
    RewriteCond %{REQUEST_FILENAME} !-f
    RewriteCond %{REQUEST_FILENAME} !-d
    RewriteRule ^api/(.*)$ api/index.php?endpoint=$1 [QSA,L]
</VirtualHost>
```

#### Nginx 配置

```nginx
server {
    listen 80;
    server_name bitfinexbot.local;
    root /path/to/BitfinexLBot/web;
    index index.html index.php;
    
    location / {
        try_files $uri $uri/ =404;
    }
    
    location /api/ {
        try_files $uri $uri/ /api/index.php?$query_string;
    }
    
    location ~ \.php$ {
        fastcgi_pass unix:/var/run/php/php8.0-fpm.sock;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
    }
}
```

### 4. 测试安装

1. **访问主界面**：
   ```
   http://bitfinexbot.local/
   ```

2. **测试 API**：
   ```bash
   curl http://bitfinexbot.local/api/index.php?endpoint=status&action=bot
   ```

3. **预期响应**：
   ```json
   {
     "success": true,
     "data": {
       "status": "stopped",
       "status_text": "已停止",
       ...
     }
   }
   ```

## API 接口文档

### 命令接口 (`/api/commands`)

#### 提交命令
```http
POST /api/index.php?endpoint=commands
Content-Type: application/json

{
  "action": "submit",
  "command_type": "start",
  "command_data": {}
}
```

#### 查询命令
```http
GET /api/index.php?endpoint=commands&action=list&limit=10&status=completed
```

### 状态接口 (`/api/status`)

#### 机器人状态
```http
GET /api/index.php?endpoint=status&action=bot
```

#### 系统状态
```http
GET /api/index.php?endpoint=status&action=system
```

#### 通知列表
```http
GET /api/index.php?endpoint=status&action=notifications&limit=5
```

### 配置接口 (`/api/config`)

#### 获取配置
```http
GET /api/index.php?endpoint=config&action=list
```

#### 更新配置
```http
PUT /api/index.php?endpoint=config
Content-Type: application/json

{
  "action": "update",
  "config_key": "MIN_DAILY_LEND_RATE",
  "config_value": "0.025",
  "config_type": "number"
}
```

## 支持的命令类型

| 命令 | 描述 | 参数 |
|------|------|------|
| `start` | 启动机器人 | 无 |
| `stop` | 停止机器人 | 无 |
| `restart` | 重启机器人 | 无 |
| `status` | 查询状态 | 无 |
| `balance` | 查询余额 | 无 |
| `rates` | 查询利率 | 无 |
| `orders` | 查询订单 | 无 |
| `config` | 查询配置 | 无 |
| `cancel_all` | 取消所有订单 | 无 |
| `lending_check` | 借贷检查 | 无 |
| `rate_check` | 利率检查 | 无 |

## 安全注意事项

### 🔒 访问控制
- 建议配置 HTTP 基础认证或 IP 白名单
- 使用 HTTPS 加密传输
- 定期更换数据库密码

### 🛡️ 数据保护
- 敏感配置信息不会在 API 中返回
- 所有数据库操作使用预处理语句防止 SQL 注入
- 输入验证和错误处理

### 📋 操作审计
- 所有操作都会记录到 `operation_logs` 表
- 包含用户ID、操作类型、IP地址等信息
- 支持操作历史追踪

## 故障排查

### 常见问题

1. **无法连接数据库**
   ```
   错误：数据库连接失败
   ```
   - 检查 `config/database.php` 中的连接参数
   - 确认 MySQL 服务正在运行
   - 验证用户权限

2. **API 返回 404**
   ```
   错误：Invalid endpoint
   ```
   - 检查 Web 服务器 URL 重写配置
   - 确认 API 文件存在且可读

3. **权限错误**
   ```
   错误：Access denied
   ```
   - 检查文件权限（755 for directories, 644 for files）
   - 确认 Web 服务器用户权限

### 调试模式

在 `config/database.php` 中启用调试：

```php
// 临时启用错误显示（生产环境请关闭）
ini_set('display_errors', 1);
error_reporting(E_ALL);
```

### 日志查看

```bash
# 查看 PHP 错误日志
tail -f /var/log/php/error.log

# 查看 Web 服务器日志
tail -f /var/log/apache2/error.log  # Apache
tail -f /var/log/nginx/error.log    # Nginx
```

## 性能优化

### 数据库优化
- 定期清理过期的命令记录和日志
- 为经常查询的字段添加索引
- 使用连接池减少连接开销

### 缓存策略
- 状态信息可以缓存 10-30 秒
- 配置信息可以缓存更长时间
- 使用 Redis 或 Memcached 提升性能

### 前端优化
- 启用 Gzip 压缩
- 使用 CDN 加载外部资源
- 合理设置缓存策略

## 扩展开发

### 添加新的 API 接口

1. 在 `api/` 目录创建新文件
2. 在 `api/index.php` 中注册路由
3. 遵循现有的响应格式标准

### 自定义前端界面

- 基于 Bootstrap 5 框架
- 支持响应式设计
- 易于主题定制和扩展

## 更新和维护

### 版本更新
- 备份数据库和配置文件
- 测试新版本兼容性
- 逐步部署和验证

### 定期维护
- 清理过期日志和临时文件
- 更新依赖包和安全补丁
- 监控性能和资源使用

---

如有问题或建议，请查看项目主 README 或提交 Issue。