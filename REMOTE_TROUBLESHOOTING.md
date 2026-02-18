# 远程部署环境故障排查指南

## 问题描述

容器启动后，bitfinex-bot 进程一直崩溃退出（exit status 1），supervisor 显示：
```
WARN exited: bitfinex-bot (exit status 1; not expected)
INFO gave up: bitfinex-bot entered FATAL state, too many start retries too quickly
```

## 诊断步骤

### 1. 运行诊断脚本

在远程机器上执行：
```bash
# 下载诊断脚本
curl -O https://raw.githubusercontent.com/ApexLGF/BitfinexLBot/main/diagnose.sh
chmod +x diagnose.sh

# 运行诊断
./diagnose.sh > diagnose_output.txt 2>&1

# 查看结果
cat diagnose_output.txt
```

或者手动执行以下命令：

### 2. 查看详细错误日志

```bash
# 查看 supervisor 错误日志
docker exec bitfinex-bot-web cat /var/log/supervisor/bitfinex-bot-stderr.log

# 查看 supervisor 标准输出
docker exec bitfinex-bot-web cat /var/log/supervisor/bitfinex-bot-stdout.log

# 查看容器日志
docker logs bitfinex-bot-web --tail 100
```

### 3. 手动测试 bot 启动

```bash
# 进入容器手动运行 bot
docker exec -it bitfinex-bot-web /app/bitfinex-bot -c /app/config.yaml
```

这会显示详细的启动错误信息。

### 4. 验证配置文件

```bash
# 查看配置文件内容
docker exec bitfinex-bot-web cat /app/config.yaml

# 检查配置文件格式
docker exec bitfinex-bot-web /app/bitfinex-bot -c /app/config.yaml --validate
```

## 常见问题和解决方案

### 问题 1: API 密钥无效

**症状**：
```
failed to authenticate: API request failed with status 401
```

**解决方案**：
1. 确认 API 密钥完整且正确
2. 检查 API 密钥权限（需要 Funding 权限）
3. 更新配置文件中的密钥

```bash
# 编辑配置文件
vi config.yaml

# 重启容器
docker compose restart
```

### 问题 2: 配置文件格式错误

**症状**：
```
Error parsing config file
yaml: unmarshal errors
```

**解决方案**：
1. 检查 YAML 格式（缩进必须使用空格，不能用 Tab）
2. 确保所有必需字段都存在
3. 使用 [config.remote.yaml](config.remote.yaml) 作为模板

### 问题 3: 端口冲突

**症状**：
```
bind: address already in use
```

**解决方案**：
修改 config.yaml 中的 API_PORT：
```yaml
API_PORT: 8091  # 改为其他端口
```

### 问题 4: 权限问题

**症状**：
```
permission denied
cannot open config file
```

**解决方案**：
```bash
# 检查文件权限
docker exec bitfinex-bot-web ls -la /app/

# 修复权限
docker exec bitfinex-bot-web chmod 644 /app/config.yaml
```

### 问题 5: 多币种配置缺失字段

**症状**：
```
missing required field in currency config
```

**解决方案**：
确保每个币种配置包含所有必需字段。参考 [config.remote.yaml](config.remote.yaml)。

## 完整配置示例

参考 [config.remote.yaml](config.remote.yaml) 文件，确保包含：

1. **全局配置**：
   - BITFINEX_API_KEY
   - BITFINEX_SECRET_KEY
   - ORDER_LIMIT
   - MINUTES_RUN

2. **币种配置**（CURRENCIES）：
   - 每个币种必须包含所有字段
   - ENABLED 设置为 true 才会启用

3. **API 配置**：
   - API_ENABLED: true
   - API_PORT: 8090
   - API_HOST: "127.0.0.1"

## 验证修复

修复后，验证容器正常运行：

```bash
# 1. 检查容器状态
docker ps | grep bitfinex-bot
# 应该显示 "Up" 状态

# 2. 检查进程状态
docker exec bitfinex-bot-web supervisorctl status
# 应该显示：
# bitfinex-bot    RUNNING   pid xxx, uptime x:xx:xx
# nginx           RUNNING   pid xxx, uptime x:xx:xx

# 3. 查看日志
docker logs bitfinex-bot-web --tail 50
# 应该看到正常的启动日志，没有错误

# 4. 访问 Web 界面
curl http://localhost:8089/health
# 应该返回 {"status":"ok"}
```

## 获取帮助

如果以上步骤无法解决问题，请：

1. 运行诊断脚本并保存输出
2. 收集以下信息：
   - 完整的错误日志
   - 配置文件内容（隐藏 API 密钥）
   - Docker 版本和系统信息
3. 提交 Issue 并附上诊断信息

## 紧急回滚

如果新版本无法正常工作，可以回滚到旧版本：

```bash
# 停止当前容器
docker stop bitfinex-bot-web
docker rm bitfinex-bot-web

# 使用旧版本
docker pull apexlgf/bitfinexwebbot:v2.2.0
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.2.0
```

注意：旧版本不支持多币种，需要使用旧格式的配置文件。
