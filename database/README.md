# BitfinexBot 数据库设置

本目录包含 BitfinexBot 项目的数据库相关文件。

## 文件说明

- `schema.sql` - 数据库表结构定义
- `init.sh` - 数据库初始化脚本
- `README.md` - 本说明文件

## 快速开始

### 1. 运行初始化脚本

```bash
cd database
./init.sh [mysql_host] [mysql_user] [mysql_password]
```

**参数说明:**
- `mysql_host` - MySQL 主机地址（默认: localhost）
- `mysql_user` - MySQL 管理员用户（默认: root）
- `mysql_password` - MySQL 管理员密码（默认: 空）

**示例:**
```bash
# 使用默认参数（localhost, root, 无密码）
./init.sh

# 指定参数
./init.sh localhost root mypassword
```

### 2. 手动执行（可选）

如果你偏好手动设置，可以直接执行 SQL 文件：

```bash
mysql -u root -p < schema.sql
```

## 数据库架构

### 核心表结构

1. **users** - 用户管理
   - 管理 Web 界面用户账户
   - 支持角色权限（admin/user）

2. **web_commands** - 命令队列
   - 存储从 Web 界面提交的命令
   - 支持多种命令类型（start、stop、config等）
   - 包含执行状态和结果

3. **bot_realtime_status** - 实时状态
   - 机器人当前运行状态
   - 余额、订单、利率等关键指标
   - 性能监控数据

4. **notifications** - 通知历史
   - 系统通知和警告消息
   - 替换原有的 Telegram 通知功能

5. **system_config** - 系统配置
   - 动态配置存储
   - 支持在线修改机器人参数

6. **operation_logs** - 操作日志
   - 记录用户操作和系统事件
   - 用于审计和故障排查

7. **order_history** - 订单历史（可选）
   - 历史订单数据存储
   - 用于统计分析

### 默认数据

- **管理员账户**: admin / admin123
- **数据库用户**: bitfinex_bot / BitfinexBot@2025
- **默认配置**: 从原 config.yaml 导入的基础参数

## 连接配置

初始化完成后，请更新 `config.yaml` 中的数据库配置：

```yaml
database:
  host: "localhost"
  port: 3306
  username: "bitfinex_bot"
  password: "BitfinexBot@2025"
  database: "bitfinex_bot_db"
  charset: "utf8mb4"
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime: "1h"
```

## 安全注意事项

1. **密码管理**: 
   - 生产环境请修改默认密码
   - 使用强密码策略

2. **权限控制**:
   - 数据库用户仅有必要权限
   - 定期审查用户权限

3. **数据备份**:
   - 定期备份数据库
   - 测试恢复流程

## 故障排查

### 常见问题

1. **连接失败**
   ```
   Error: 无法连接到 MySQL 服务器
   ```
   - 检查 MySQL 服务是否运行
   - 验证连接参数是否正确
   - 确认防火墙设置

2. **权限错误**
   ```
   Access denied for user 'bitfinex_bot'
   ```
   - 重新运行初始化脚本
   - 手动检查用户权限

3. **字符集问题**
   - 确保使用 utf8mb4 字符集
   - 检查 MySQL 配置

### 维护命令

```bash
# 查看表状态
mysql -u bitfinex_bot -p bitfinex_bot_db -e "SHOW TABLES;"

# 检查表结构
mysql -u bitfinex_bot -p bitfinex_bot_db -e "DESCRIBE web_commands;"

# 清理命令历史（保留最近1000条）
mysql -u bitfinex_bot -p bitfinex_bot_db -e "DELETE FROM web_commands WHERE id NOT IN (SELECT id FROM (SELECT id FROM web_commands ORDER BY id DESC LIMIT 1000) temp);"
```