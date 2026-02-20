// BitfinexBot Web 应用主逻辑
class BitfinexBotApp {
    constructor() {
        this.isConnected = false;
        this.lastConnectionTime = null;
        this.logUpdateTimer = null;
        this.logUpdateInterval = 3000; // 3秒更新一次日志
        this.overallInfoTimer = null;
        this.overallInfoInterval = 60000; // 60秒更新一次整体信息
        this.fullRefreshTimer = null;
        this.fullRefreshInterval = 300000; // 默认 5 分钟
        this.minutesRun = 5;
        this.enabledCurrencies = [];
        this.countdownTimer = null;
        this.nextRefreshTime = null;

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

            // 启动全量刷新轮询（基于 MINUTES_RUN）
            this.startFullRefreshPolling();

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
        const timestampElement = document.getElementById('connection-timestamp');

        if (this.isConnected) {
            statusElement.textContent = '已连接';
            iconElement.className = 'bi bi-circle-fill text-success me-1';

            // 更新时间戳
            this.lastConnectionTime = new Date();
            if (timestampElement) {
                timestampElement.textContent = this.formatConnectionTime(this.lastConnectionTime);
            }
        } else {
            statusElement.textContent = '连接失败';
            iconElement.className = 'bi bi-circle-fill text-danger me-1';
            if (timestampElement) {
                timestampElement.textContent = '';
            }
        }
    }

    // 格式化连接时间
    formatConnectionTime(date) {
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        const seconds = String(date.getSeconds()).padStart(2, '0');
        return `${hours}:${minutes}:${seconds}`;
    }

    // 显示加载指示器
    showLoadingIndicator() {
        const loadingIndicator = document.getElementById('loading-indicator');
        if (loadingIndicator) {
            loadingIndicator.style.display = 'inline-block';
        }
    }

    // 隐藏加载指示器
    hideLoadingIndicator() {
        const loadingIndicator = document.getElementById('loading-indicator');
        if (loadingIndicator) {
            loadingIndicator.style.display = 'none';
        }
    }

    // 更新倒计时显示
    updateCountdown() {
        const countdownElement = document.getElementById('refresh-countdown');
        if (!countdownElement || !this.nextRefreshTime) return;

        const now = Date.now();
        const remaining = Math.max(0, this.nextRefreshTime - now);

        if (remaining === 0) {
            countdownElement.style.display = 'none';
            return;
        }

        const minutes = Math.floor(remaining / 60000);
        const seconds = Math.floor((remaining % 60000) / 1000);

        countdownElement.textContent = `下次刷新: ${minutes}:${String(seconds).padStart(2, '0')}`;
        countdownElement.style.display = 'inline';
    }

    // 启动倒计时
    startCountdown() {
        // 停止现有倒计时
        if (this.countdownTimer) {
            clearInterval(this.countdownTimer);
        }

        // 设置下次刷新时间
        this.nextRefreshTime = Date.now() + this.fullRefreshInterval;

        // 立即更新一次
        this.updateCountdown();

        // 每秒更新倒计时
        this.countdownTimer = setInterval(() => {
            this.updateCountdown();
        }, 1000);

        console.log('[APP] 倒计时已启动');
    }

    // 停止倒计时
    stopCountdown() {
        if (this.countdownTimer) {
            clearInterval(this.countdownTimer);
            this.countdownTimer = null;
        }

        const countdownElement = document.getElementById('refresh-countdown');
        if (countdownElement) {
            countdownElement.style.display = 'none';
        }
    }

    // 初始化币种
    async initializeCurrencies() {
        try {
            this.showLoadingIndicator();

            // 获取配置
            const response = await api.getConfig();
            const config = response.data;

            // 保存 MINUTES_RUN 配置
            this.minutesRun = config.MINUTES_RUN || 5;
            this.fullRefreshInterval = this.minutesRun * 60 * 1000;
            console.log(`[APP] 全量刷新间隔设置为 ${this.minutesRun} 分钟`);

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
                this.hideLoadingIndicator();
                return;
            }

            console.log('[APP] 启用的币种:', currencies);

            // 保存币种列表和配置（供 configManager 使用，避免重复请求）
            this.enabledCurrencies = currencies;
            this.cachedConfig = config;

            // 初始化 TAB
            tabManager.initTabs(currencies);

            // 优化：只加载第一个币种的数据，其他币种延迟加载
            const firstCurrency = currencies[0];
            console.log(`[APP] 优先加载第一个币种: ${firstCurrency}`);

            // 并行执行：加载第一个币种数据 + FRR 利率（年度收益延迟加载）
            const [firstCurrencyData] = await Promise.all([
                this.loadCurrencyData(firstCurrency),
                overallInfo.refreshFRROnly() // 只加载 FRR，年度收益延迟
            ]);

            // 渲染第一个币种的面板（使用已缓存的数据）
            await dashboard.render(firstCurrency);

            // 使用已缓存的配置初始化配置管理器（避免重复请求）
            configManager.initWithConfig(config);

            this.hideLoadingIndicator();

            // 延迟加载：其他币种数据和年度收益（不阻塞页面显示）
            this.loadRemainingDataInBackground(currencies);

        } catch (error) {
            console.error('[APP] 初始化币种失败:', error);
            alert('初始化失败: ' + error.message);
            this.hideLoadingIndicator();
        }
    }

    // 后台加载剩余数据
    async loadRemainingDataInBackground(currencies) {
        console.log('[APP] 开始后台加载剩余数据...');

        // 延迟 500ms 后开始加载，让页面先渲染完成
        setTimeout(async () => {
            try {
                // 1. 加载其他币种数据（如果有多个币种）
                if (currencies.length > 1) {
                    const otherCurrencies = currencies.slice(1);
                    console.log('[APP] 后台加载其他币种:', otherCurrencies);

                    // 串行加载其他币种，避免同时发起太多请求
                    for (const currency of otherCurrencies) {
                        await this.loadCurrencyData(currency);
                    }
                }

                // 2. 加载年度收益（最慢的请求）
                console.log('[APP] 后台加载年度收益...');
                await overallInfo.refreshYearlyOnly();

                console.log('[APP] 后台数据加载完成');
            } catch (error) {
                console.error('[APP] 后台数据加载失败:', error);
            }
        }, 500);
    }

    // 币种切换事件处理
    async onCurrencyChanged(currency) {
        console.log('[APP] 币种切换到:', currency);

        // 检查连接状态
        if (!this.isConnected) {
            console.warn('[APP] 未连接，无法切换币种');
            alert('连接已断开，请刷新页面重新连接');
            return;
        }

        try {
            await dashboard.render(currency);
        } catch (error) {
            console.error('[APP] 渲染币种面板失败:', error);
            // 如果是连接错误，更新连接状态
            if (error.message && error.message.includes('Failed to fetch')) {
                this.isConnected = false;
                this.updateConnectionStatus();
                alert('连接已断开，请刷新页面重新连接');
            }
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

    // 启动全量刷新轮询
    startFullRefreshPolling() {
        if (this.fullRefreshTimer) {
            clearInterval(this.fullRefreshTimer);
        }

        // 启动定时器
        this.fullRefreshTimer = setInterval(() => {
            this.performFullRefresh();
        }, this.fullRefreshInterval);

        // 启动倒计时
        this.startCountdown();

        console.log(`[APP] 全量刷新轮询已启动，间隔 ${this.minutesRun} 分钟`);
    }

    // 停止全量刷新轮询
    stopFullRefreshPolling() {
        if (this.fullRefreshTimer) {
            clearInterval(this.fullRefreshTimer);
            this.fullRefreshTimer = null;
        }
        this.stopCountdown();
    }

    // 执行全量刷新
    async performFullRefresh() {
        console.log('[APP] 开始全量刷新...');
        this.showLoadingIndicator();
        this.stopCountdown(); // 停止倒计时显示

        try {
            // 1. 重新检查连接状态
            await this.checkConnection();

            if (!this.isConnected) {
                console.error('[APP] 连接失败，跳过数据刷新');
                this.hideLoadingIndicator();
                this.startCountdown(); // 重新启动倒计时
                return;
            }

            // 2. 预加载所有币种数据
            await this.loadAllCurrenciesData();

            // 3. 刷新当前显示的币种
            const currentCurrency = tabManager.activeCurrency;
            if (currentCurrency) {
                await dashboard.render(currentCurrency);
            }

            // 4. 刷新整体信息
            await overallInfo.refreshData();

            console.log('[APP] 全量刷新完成');
            this.hideLoadingIndicator();
            this.startCountdown(); // 重新启动倒计时
        } catch (error) {
            console.error('[APP] 全量刷新失败:', error);
            this.hideLoadingIndicator();
            this.startCountdown(); // 重新启动倒计时
        }
    }

    // 预加载所有币种数据
    async loadAllCurrenciesData() {
        // 检查连接状态
        if (!this.isConnected) {
            console.warn('[APP] 未连接，跳过数据加载');
            return;
        }

        if (!this.enabledCurrencies || this.enabledCurrencies.length === 0) {
            console.warn('[APP] 没有启用的币种，跳过预加载');
            return;
        }

        console.log('[APP] 开始预加载所有币种数据:', this.enabledCurrencies);

        try {
            // 并行获取所有币种的数据
            const promises = this.enabledCurrencies.map(currency =>
                this.loadCurrencyData(currency)
            );

            await Promise.all(promises);

            console.log('[APP] 所有币种数据预加载完成');
        } catch (error) {
            console.error('[APP] 预加载币种数据失败:', error);
            // 如果是连接错误，更新连接状态
            if (error.message && error.message.includes('Failed to fetch')) {
                this.isConnected = false;
                this.updateConnectionStatus();
            }
        }
    }

    // 加载单个币种数据
    async loadCurrencyData(currency) {
        try {
            console.log(`[APP] 预加载 ${currency} 数据`);

            const [earnings, offers, credits, status] = await Promise.all([
                api.getEarnings(`?currency=${currency}`),
                api.getOffers(`?currency=${currency}`),
                api.getFundingCredits(`?currency=${currency}`),
                api.getStatus(`?currency=${currency}`)
            ]);

            // 缓存数据到 dashboard
            dashboard.cacheData(currency, {
                earnings: earnings.data,
                offers: offers.data,
                credits: credits.data,
                status: status.data
            });

            console.log(`[APP] ${currency} 数据预加载完成`);
        } catch (error) {
            console.error(`[APP] 预加载 ${currency} 数据失败:`, error);
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
        this.stopFullRefreshPolling();
        this.stopCountdown();
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
