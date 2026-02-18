// BitfinexBot Web 应用主逻辑
class BitfinexBotApp {
    constructor() {
        this.isConnected = false;
        this.logUpdateTimer = null;
        this.logUpdateInterval = 3000; // 3秒更新一次日志
        this.overallInfoTimer = null;
        this.overallInfoInterval = 60000; // 60秒更新一次整体信息

        this.init();
    }

    // 初始化应用
    async init() {
        console.log('[APP] 页面初始化开始 - ', new Date().toISOString());

        // 检查连接状态
        await this.checkConnection();

        if (this.isConnected) {
            // 加载配置并初始化币种
            await this.initializeCurrencies();

            // 启动日志轮询
            this.startLogPolling();

            // 启动整体信息轮询
            this.startOverallInfoPolling();

            // 监听币种切换事件
            document.addEventListener('currencyChanged', (e) => {
                this.onCurrencyChanged(e.detail.currency);
            });

            console.log('[APP] 应用初始化完成');
        } else {
            console.error('[APP] 无法连接到服务器');
            alert('无法连接到服务器，请检查后端服务是否正常运行');
        }
    }

    // 检查连接状态
    async checkConnection() {
        try {
            this.isConnected = await api.healthCheck();
            this.updateConnectionStatus();
            return this.isConnected;
        } catch (error) {
            console.error('[APP] 连接检查失败:', error);
            this.isConnected = false;
            this.updateConnectionStatus();
            return false;
        }
    }

    // 更新连接状态显示
    updateConnectionStatus() {
        const statusElement = document.getElementById('bot-status');
        const iconElement = document.querySelector('#status-indicator i');

        if (this.isConnected) {
            statusElement.textContent = '已连接';
            iconElement.className = 'bi bi-circle-fill text-success me-1';
        } else {
            statusElement.textContent = '连接失败';
            iconElement.className = 'bi bi-circle-fill text-danger me-1';
        }
    }

    // 初始化币种
    async initializeCurrencies() {
        try {
            // 获取配置
            const response = await api.getConfig();
            const config = response.data;

            // 获取启用的币种
            const currencies = [];
            if (config.CURRENCIES) {
                for (const [currency, currencyConfig] of Object.entries(config.CURRENCIES)) {
                    if (currencyConfig.ENABLED) {
                        currencies.push(currency);
                    }
                }
            }

            if (currencies.length === 0) {
                console.warn('[APP] 没有启用的币种');
                alert('没有启用的币种，请检查配置');
                return;
            }

            console.log('[APP] 启用的币种:', currencies);

            // 初始化 TAB
            tabManager.initTabs(currencies);

            // 渲染第一个币种的面板
            await dashboard.render(currencies[0]);

            // 加载整体信息
            await overallInfo.refreshData();

            // 加载配置到配置管理器
            await configManager.loadConfig();

        } catch (error) {
            console.error('[APP] 初始化币种失败:', error);
            alert('初始化失败: ' + error.message);
        }
    }

    // 币种切换事件处理
    async onCurrencyChanged(currency) {
        console.log('[APP] 币种切换到:', currency);

        try {
            await dashboard.render(currency);
        } catch (error) {
            console.error('[APP] 渲染币种面板失败:', error);
        }
    }

    // 启动日志轮询
    startLogPolling() {
        if (this.logUpdateTimer) {
            clearInterval(this.logUpdateTimer);
        }

        this.updateLogs(); // 立即更新一次

        this.logUpdateTimer = setInterval(() => {
            this.updateLogs();
        }, this.logUpdateInterval);

        console.log('[APP] 日志轮询已启动');
    }

    // 停止日志轮询
    stopLogPolling() {
        if (this.logUpdateTimer) {
            clearInterval(this.logUpdateTimer);
            this.logUpdateTimer = null;
        }
    }

    // 更新日志
    async updateLogs() {
        try {
            const response = await api.getLogs(1, 100);
            const logs = response.data.logs || [];

            this.updateLogDisplay(logs);
        } catch (error) {
            console.error('[APP] 更新日志失败:', error);
        }
    }

    // 更新日志显示
    updateLogDisplay(logs) {
        const container = document.getElementById('log-container');
        if (!container) return;

        // 检查用户是否在底部
        const isAtBottom = container.scrollHeight - container.scrollTop <= container.clientHeight + 50;

        if (logs.length === 0) {
            container.innerHTML = '<div class="text-center text-muted py-3">暂无日志</div>';
            return;
        }

        // 渲染日志 - 直接显示消息内容，不添加额外时间戳
        const html = logs.map(log => {
            return `<div class="log-entry">${this.escapeHtml(log.message)}</div>`;
        }).join('');

        container.innerHTML = html;

        // 智能滚动：只有当用户在底部时才自动滚动
        if (isAtBottom) {
            container.scrollTop = container.scrollHeight;
        }
    }

    // 启动整体信息轮询
    startOverallInfoPolling() {
        if (this.overallInfoTimer) {
            clearInterval(this.overallInfoTimer);
        }

        this.overallInfoTimer = setInterval(() => {
            overallInfo.refreshData();
        }, this.overallInfoInterval);

        console.log('[APP] 整体信息轮询已启动');
    }

    // 停止整体信息轮询
    stopOverallInfoPolling() {
        if (this.overallInfoTimer) {
            clearInterval(this.overallInfoTimer);
            this.overallInfoTimer = null;
        }
    }

    // HTML 转义
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // 清理资源
    cleanup() {
        this.stopLogPolling();
        this.stopOverallInfoPolling();
        dashboard.destroyAllCharts();
    }
}

// 页面加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    window.app = new BitfinexBotApp();
});

// 页面卸载时清理资源
window.addEventListener('beforeunload', () => {
    if (window.app) {
        window.app.cleanup();
    }
});
