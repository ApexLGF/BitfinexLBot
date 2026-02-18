# BitfinexBot v2.4.1 发布说明

## 发布日期
2026-02-18

## 版本信息
- **版本号**: v2.4.1
- **Docker 镜像**: `apexlgf/bitfinexwebbot:v2.4.1`

## 🐛 Bug 修复

### 修复 Web 界面"没有启用的币种"错误

**问题描述**：
v2.4.0 部署后，Web 界面加载时弹窗显示错误："没有启用的币种，请检查配置"。

**根本原因**：
`CurrencyConfig` 结构体缺少 JSON 标签，导致 API 返回的配置数据使用 Go 默认的 PascalCase 字段名（如 `Enabled`），而前端期望的是大写字段名（如 `ENABLED`）。

**修复内容**：
在 `internal/config/config.go` 的 `CurrencyConfig` 结构体中为所有字段添加 JSON 标签，确保 API 返回的字段名与前端期望一致。

**修改文件**：
- `internal/config/config.go` - 为 `CurrencyConfig` 结构体添加 JSON 标签

**修复前的 API 响应**：
```json
{
  "CURRENCIES": {
    "usd": {
      "Enabled": true,
      "MinLoan": 10,
      ...
    }
  }
}
```

**修复后的 API 响应**：
```json
{
  "CURRENCIES": {
    "usd": {
      "ENABLED": true,
      "MIN_LOAN": 10,
      ...
    }
  }
}
```

---

## 更新方法

### 方法 1：使用 Docker Compose（推荐）

```bash
# 1. 更新 docker-compose.yml 中的版本号
# image: apexlgf/bitfinexwebbot:v2.4.1

# 2. 拉取新镜像并重新创建容器
docker compose pull
docker compose up -d --force-recreate
```

### 方法 2：手动更新

```bash
# 1. 停止并删除旧容器
docker stop bitfinex-bot-web
docker rm bitfinex-bot-web

# 2. 拉取新镜像
docker pull apexlgf/bitfinexwebbot:v2.4.1

# 3. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.4.1
```

### 方法 3：使用部署脚本

```bash
./deploy.sh v2.4.1
```

---

## 验证更新

更新后，请验证以下内容：

### 1. 检查容器状态
```bash
docker ps | grep bitfinex-bot
# 应该显示容器正在运行
```

### 2. 访问 Web 界面
- 打开浏览器访问：http://localhost:8089
- **强制刷新浏览器缓存**（Ctrl+Shift+R 或 Cmd+Shift+R）
- 确认不再出现"没有启用的币种"错误
- 确认看到币种 TAB 导航（USD、UST 等）
- 确认可以正常切换币种 TAB

### 3. 测试 API 响应
```bash
# 检查配置 API 返回正确的字段名
curl http://localhost:8090/api/config | jq '.data.CURRENCIES.usd.ENABLED'
# 应该返回: true
```

---

## 技术细节

### 修改的代码

**文件**: `internal/config/config.go`

**修改前**:
```go
type CurrencyConfig struct {
	Enabled                       bool    `mapstructure:"ENABLED"`
	MinLoan                       float64 `mapstructure:"MIN_LOAN"`
	// ... 其他字段
}
```

**修改后**:
```go
type CurrencyConfig struct {
	Enabled                       bool    `mapstructure:"ENABLED" json:"ENABLED"`
	MinLoan                       float64 `mapstructure:"MIN_LOAN" json:"MIN_LOAN"`
	// ... 其他字段
}
```

### 为什么需要 JSON 标签？

- `mapstructure` 标签用于从 YAML 配置文件读取数据（使用 Viper）
- `json` 标签用于将数据序列化为 JSON 响应（使用 Gin）
- 如果没有 `json` 标签，Go 会使用默认的字段名（PascalCase）
- 添加 `json` 标签后，可以指定自定义的 JSON 字段名（UPPERCASE）

---

## 版本历史

### v2.4.1 (2026-02-18)
- 🐛 修复 Web 界面"没有启用的币种"错误
- 🔧 为 `CurrencyConfig` 结构体添加 JSON 标签

### v2.4.0 (2026-02-18)
- 🎉 完全重写多币种 Web 前端界面
- ✨ 新增 70-30 分栏布局
- ✨ 新增币种 TAB 导航
- ✨ 新增 FRR 利率显示
- ✨ 新增年度收益显示

---

## 支持

如有问题，请：
- 查看 [REMOTE_TROUBLESHOOTING.md](REMOTE_TROUBLESHOOTING.md)
- 查看 [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md)
- 提交 Issue: https://github.com/ApexLGF/BitfinexLBot/issues

---

祝使用愉快！🚀
