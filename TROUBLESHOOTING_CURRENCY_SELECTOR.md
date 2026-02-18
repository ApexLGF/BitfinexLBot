# Web 界面币种选择功能故障排查

## 问题描述

部署环境更新镜像后，Web 界面上没有显示币种选择功能。

## 原因分析

最常见的原因是**浏览器缓存**了旧版本的 HTML/JavaScript 文件。

## 解决方案

### 方案1：清除浏览器缓存（推荐）

#### Chrome/Edge:
1. 打开开发者工具（F12 或 Cmd+Option+I）
2. 右键点击刷新按钮
3. 选择"清空缓存并硬性重新加载"

或者：
1. 按 `Cmd+Shift+Delete` (Mac) 或 `Ctrl+Shift+Delete` (Windows)
2. 选择"缓存的图片和文件"
3. 点击"清除数据"

#### Firefox:
1. 按 `Cmd+Shift+Delete` (Mac) 或 `Ctrl+Shift+Delete` (Windows)
2. 选择"缓存"
3. 点击"立即清除"

#### Safari:
1. 按 `Cmd+Option+E` 清空缓存
2. 或者在"开发"菜单中选择"清空缓存"

### 方案2：使用隐私/无痕模式

打开一个新的隐私/无痕窗口访问：
```
http://localhost:8089
```

如果在隐私模式下能看到币种选择器，说明确实是缓存问题。

### 方案3：强制刷新页面

- **Windows**: `Ctrl + F5` 或 `Ctrl + Shift + R`
- **Mac**: `Cmd + Shift + R`

### 方案4：添加版本参数到 HTML

修改 `web/index.html`，在引用 JS 文件时添加版本参数：

```html
<!-- 修改前 -->
<script src="static/js/api.js?v=10"></script>
<script src="static/js/charts.js?v=10"></script>
<script src="static/js/app.js?v=10"></script>

<!-- 修改后 -->
<script src="static/js/api.js?v=23"></script>
<script src="static/js/charts.js?v=23"></script>
<script src="static/js/app.js?v=23"></script>
```

然后重新构建镜像：
```bash
docker build -t apexlgf/bitfinexwebbot:v2.3.1 .
docker push apexlgf/bitfinexwebbot:v2.3.1
```

### 方案5：配置 Nginx 禁用缓存（开发环境）

在 `docker/nginx.conf` 中添加：

```nginx
location /static/ {
    add_header Cache-Control "no-cache, no-store, must-revalidate";
    add_header Pragma "no-cache";
    add_header Expires "0";
}
```

## 验证步骤

### 1. 检查 HTML 源代码

在浏览器中：
1. 右键点击页面 → "查看页面源代码"
2. 搜索 `currency-select`
3. 应该能找到类似这样的代码：

```html
<div class="currency-selector">
    <select id="currency-select" class="form-select form-select-sm" style="min-width: 150px;">
        <option value="">所有币种</option>
    </select>
</div>
```

如果找不到，说明 HTML 文件没有更新。

### 2. 检查 JavaScript 文件

在浏览器开发者工具中：
1. 打开 Console 标签
2. 输入：
```javascript
document.getElementById('currency-select')
```
3. 如果返回 `null`，说明元素不存在
4. 如果返回一个 `<select>` 元素，说明元素存在但可能被隐藏

### 3. 检查 JavaScript 错误

在浏览器开发者工具的 Console 标签中查看是否有 JavaScript 错误。

### 4. 检查网络请求

在浏览器开发者工具的 Network 标签中：
1. 刷新页面
2. 查看 `app.js` 的请求
3. 检查响应头中的 `Last-Modified` 或 `ETag`
4. 确认是否是最新版本

## 验证镜像内容

在服务器上运行以下命令验证镜像内容：

```bash
# 检查 HTML 文件
docker run --rm --entrypoint cat apexlgf/bitfinexwebbot:v2.3.0 \
  /usr/share/nginx/html/index.html | grep -A 5 "currency-select"

# 检查 JavaScript 文件
docker run --rm --entrypoint cat apexlgf/bitfinexwebbot:v2.3.0 \
  /usr/share/nginx/html/static/js/app.js | grep -A 5 "selectedCurrency"

# 检查文件修改时间
docker run --rm --entrypoint ls apexlgf/bitfinexwebbot:v2.3.0 \
  -la /usr/share/nginx/html/
```

## 确认容器使用的镜像版本

```bash
# 检查容器使用的镜像
docker inspect bitfinex-bot-web | grep Image

# 检查镜像的创建时间
docker images | grep bitfinexwebbot

# 强制拉取最新镜像
docker pull apexlgf/bitfinexwebbot:v2.3.0

# 查看镜像的 digest
docker images --digests | grep bitfinexwebbot
```

## 完整重新部署流程

如果以上方法都不行，执行完整的重新部署：

```bash
# 1. 停止并删除容器
docker stop bitfinex-bot-web
docker rm bitfinex-bot-web

# 2. 删除本地镜像
docker rmi apexlgf/bitfinexwebbot:v2.3.0
docker rmi apexlgf/bitfinexwebbot:latest

# 3. 清理 Docker 缓存
docker system prune -f

# 4. 拉取最新镜像
docker pull apexlgf/bitfinexwebbot:v2.3.0

# 5. 启动新容器
docker run -d \
  --name bitfinex-bot-web \
  -p 8089:8089 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v bitfinex_logs:/app/logs \
  --restart unless-stopped \
  apexlgf/bitfinexwebbot:v2.3.0

# 6. 查看日志
docker logs -f bitfinex-bot-web

# 7. 清除浏览器缓存后访问
# http://localhost:8089
```

## 预期结果

更新后，Web 界面应该显示：

1. **币种选择器**：在"机器人状态"卡片的右上角
2. **下拉菜单**：包含"所有币种"、"USD"、"UST"等选项
3. **动态切换**：选择不同币种时，数据会相应更新
4. **币种列**：查看所有币种时，表格会显示币种列

## 截图位置

币种选择器应该出现在这个位置：

```
┌─────────────────────────────────────────┐
│ 🤖 机器人状态          [所有币种 ▼]    │  ← 这里
├─────────────────────────────────────────┤
│                                         │
│  [机器人图标]    可用资金  总资金  ...  │
│                                         │
└─────────────────────────────────────────┘
```

## 联系支持

如果问题仍然存在，请提供以下信息：

1. 浏览器类型和版本
2. 是否清除了缓存
3. 浏览器 Console 中的错误信息
4. `docker inspect bitfinex-bot-web | grep Image` 的输出
5. 页面源代码中是否包含 `currency-select`

## 快速检查清单

- [ ] 清除浏览器缓存
- [ ] 强制刷新页面（Ctrl+F5 / Cmd+Shift+R）
- [ ] 使用隐私/无痕模式访问
- [ ] 检查容器使用的镜像版本
- [ ] 验证镜像内容包含币种选择器
- [ ] 检查浏览器 Console 是否有错误
- [ ] 完整重新部署容器
