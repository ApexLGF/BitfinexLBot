// BitfinexBot Web 应用主逻辑
class BitfinexBotApp {
    constructor() {
        this.isConnected = false;
        this.isRefreshing = false;
        this.lastLogTimestamp = null;
        this.logUpdateTimer = null;
        this.logUpdateInterval = 3000; // 3秒更新一次日志
        
        // 绑定方法的上下文
        this.updateStatus = this.updateStatus.bind(this);
        this.updateOffers = this.updateOffers.bind(this);
        this.updateCredits = this.updateCredits.bind(this);
        this.updateEarnings = this.updateEarnings.bind(this);
        this.updateLogs = this.updateLogs.bind(this);
        this.refreshAllData = this.refreshAllData.bind(this);
        this.startLogPolling = this.startLogPolling.bind(this);
        this.stopLogPolling = this.stopLogPolling.bind(this);
        
        this.init();
    }

    // 初始化应用
    async init() {
        console.log('[APP] 页面初始化开始 - ', new Date().toISOString());
        
        // 清除任何可能存在的定时器
        this.clearAllTimers();
        
        this.bindEvents();
        this.initTooltips();
        
        // 初始化图表
        chartManager.initAllCharts();
        
        // 检查连接状态
        await this.checkConnection();
        
        if (this.isConnected) {
            // 初始加载一次数据
            await this.refreshAllData();
            // 启动日志实时轮询
            this.startLogPolling();
            Utils.showNotification('连接成功', 'success');
        } else {
            Utils.showNotification('无法连接到服务器', 'danger');
        }
    }

    // 绑定事件
    bindEvents() {
        // 手动刷新按钮
        document.getElementById('refresh-btn')?.addEventListener('click', () => this.refreshAllData());
        
        // 机器人控制按钮
        document.getElementById('start-btn')?.addEventListener('click', () => this.startBot());
        document.getElementById('stop-btn')?.addEventListener('click', () => this.stopBot());
        document.getElementById('restart-btn')?.addEventListener('click', () => this.restartBot());
        
        // 配置管理
        document.getElementById('refresh-config')?.addEventListener('click', () => this.loadConfig());
        document.getElementById('save-config')?.addEventListener('click', () => this.saveConfig());
        
        // 窗口大小变化时调整图表
        window.addEventListener('resize', Utils.debounce(() => {
            chartManager.resizeCharts();
        }, 250));
    }

    // 初始化工具提示
    initTooltips() {
        const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
        tooltipTriggerList.map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));
    }

    // 检查连接状态
    async checkConnection() {
        try {
            this.isConnected = await api.healthCheck();
            this.updateConnectionStatus();
            return this.isConnected;
        } catch (error) {
            console.error('连接检查失败:', error);
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

    // 手动刷新所有数据
    async refreshAllData() {
        console.log('[APP] refreshAllData 被调用 - ', new Date().toISOString());
        if (this.isRefreshing) {
            Utils.showNotification('正在刷新中，请稍候...', 'warning', 1500);
            return;
        }

        this.isRefreshing = true;
        const refreshBtn = document.getElementById('refresh-btn');
        const refreshIcon = refreshBtn?.querySelector('i');
        
        try {
            // 更新按钮状态
            if (refreshBtn) refreshBtn.disabled = true;
            if (refreshIcon) {
                refreshIcon.style.animation = 'spin 1s linear infinite';
            }

            // 检查连接状态
            if (!this.isConnected) {
                await this.checkConnection();
                if (!this.isConnected) {
                    Utils.showNotification('无法连接到服务器', 'danger');
                    return;
                }
            }

            // 并行更新所有数据，允许部分失败
            const updatePromises = [
                this.updateStatus().catch(e => console.error('更新状态失败:', e)),
                this.updateOffers().catch(e => console.error('更新挂单失败:', e)),
                this.updateCredits().catch(e => console.error('更新已贷出订单失败:', e)),
                this.updateEarnings().catch(e => console.error('更新收益失败:', e)),
                this.updateLogs().catch(e => console.error('更新日志失败:', e)),
            ];
            
            await Promise.allSettled(updatePromises);

            Utils.showNotification('数据刷新成功', 'success', 1500);
            
        } catch (error) {
            console.error('刷新数据失败:', error);
            this.isConnected = false;
            this.updateConnectionStatus();
            Utils.showNotification('刷新失败: ' + error.message, 'danger');
        } finally {
            // 恢复按钮状态
            this.isRefreshing = false;
            if (refreshBtn) refreshBtn.disabled = false;
            if (refreshIcon) {
                refreshIcon.style.animation = '';
            }
        }
    }

    // 更新机器人状态
    async updateStatus() {
        try {
            const response = await api.getStatus();
            const status = response.data;
            
            // 检查是否有API错误信息
            if (status.errors && status.errors.length > 0) {
                status.errors.forEach(error => {
                    Utils.showNotification(`API错误: ${error}`, 'warning', 8000);
                });
            }
            
            // 更新状态显示
            this.updateStatusDisplay(status);
            
            // 更新机器人图标
            this.updateRobotIcon(status.is_running);
            
        } catch (error) {
            console.error('获取状态失败:', error);
            throw error;
        }
    }

    // 更新状态显示
    updateStatusDisplay(status) {
        // 可用资金
        const fundsElement = document.getElementById('available-funds');
        if (fundsElement) {
            fundsElement.textContent = Utils.formatCurrency(status.available_funds, status.currency);
        }
        
        // 活跃订单数
        const offersElement = document.getElementById('active-offers');
        if (offersElement) {
            offersElement.textContent = status.active_offers;
        }
        
        // 总收益
        const earningsElement = document.getElementById('total-earnings');
        if (earningsElement) {
            earningsElement.textContent = Utils.formatCurrency(status.total_earnings, status.currency);
        }
        
        // 下次运行时间
        const nextRunElement = document.getElementById('next-run');
        if (nextRunElement) {
            nextRunElement.textContent = Utils.formatRelativeTime(status.next_run);
        }
        
        // 更新按钮状态
        this.updateButtonStates(status.is_running);
    }

    // 更新机器人图标
    updateRobotIcon(isRunning) {
        const robotIcon = document.getElementById('robot-icon');
        if (robotIcon) {
            robotIcon.className = isRunning ? 
                'bi bi-robot robot-running' : 
                'bi bi-robot robot-stopped';
        }
    }

    // 更新按钮状态
    updateButtonStates(isRunning) {
        const startBtn = document.getElementById('start-btn');
        const stopBtn = document.getElementById('stop-btn');
        const restartBtn = document.getElementById('restart-btn');
        
        if (startBtn) startBtn.disabled = isRunning;
        if (stopBtn) stopBtn.disabled = !isRunning;
        if (restartBtn) restartBtn.disabled = false;
    }

    // 更新放贷订单
    async updateOffers() {
        try {
            console.log('[APP] updateOffers 被调用 - ', new Date().toISOString(), '- 调用栈:', new Error().stack);
            const response = await api.getOffers();
            const offers = response.data;
            
            this.updateOffersTable(offers);
            chartManager.updateRateDistributionChart(offers);
            
        } catch (error) {
            console.error('获取订单失败:', error);
            Utils.showNotification(`获取放贷订单失败: ${error.message}`, 'danger', 5000);
            // 不再抛出错误，避免中断其他数据加载
        }
    }

    // 更新已贷出订单
    async updateCredits() {
        try {
            const response = await api.getFundingCredits();
            const creditsData = response.data;
            
            this.updateCreditsTable(creditsData.credits || []);
            this.updateCreditsStats(creditsData || {});
            
        } catch (error) {
            console.error('获取已贷出订单失败:', error);
            Utils.showNotification(`获取已贷出订单失败: ${error.message}`, 'warning', 5000);
            // 显示错误状态
            this.updateCreditsTable([]);
            this.updateCreditsStats({});
            // 不再抛出错误，避免中断其他数据加载
        }
    }

    // 更新订单表格
    updateOffersTable(offers) {
        const tbody = document.querySelector('#offers-table tbody');
        if (!tbody) return;
        
        if (!offers || offers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">暂无活跃订单</td></tr>';
            return;
        }
        
        const rows = offers.map(offer => `
            <tr>
                <td><code>${offer.id}</code></td>
                <td>${Utils.formatCurrency(offer.amount, offer.currency)}</td>
                <td>${Utils.formatPercentage(offer.rate)}</td>
                <td>${offer.period}天</td>
                <td>
                    <span class="badge ${this.getStatusBadgeClass(offer.status)}">
                        ${offer.status}
                    </span>
                </td>
                <td>${Utils.formatTime(offer.created)}</td>
            </tr>
        `).join('');
        
        tbody.innerHTML = rows;
    }

    // 更新已贷出订单表格
    updateCreditsTable(credits) {
        const tbody = document.querySelector('#credits-table tbody');
        if (!tbody) return;
        
        // 处理null、undefined或空数组的情况
        if (!credits || !Array.isArray(credits) || credits.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted">暂无已贷出订单</td></tr>';
            return;
        }
        
        const rows = credits.map(credit => `
            <tr>
                <td><code>${credit.id}</code></td>
                <td>${Utils.formatCurrency(credit.amount, credit.currency)}</td>
                <td>${Utils.formatPercentage(credit.rate)}</td>
                <td>${credit.period}天</td>
                <td>
                    <span class="badge ${this.getStatusBadgeClass(credit.status)}">
                        ${credit.status}
                    </span>
                </td>
                <td>${Utils.formatTime(credit.created)}</td>
                <td>${Utils.formatTime(credit.opened)}</td>
            </tr>
        `).join('');
        
        tbody.innerHTML = rows;
    }

    // 更新已贷出订单统计
    updateCreditsStats(creditsData) {
        const countElement = document.getElementById('credits-count');
        const totalElement = document.getElementById('credits-total');
        const avgRateElement = document.getElementById('credits-avg-rate');
        
        if (countElement) {
            countElement.textContent = creditsData.total_count || 0;
        }
        
        if (totalElement) {
            totalElement.textContent = Utils.formatCurrency(creditsData.total_amount || 0);
        }
        
        if (avgRateElement) {
            avgRateElement.textContent = creditsData.avg_rate ? 
                (creditsData.avg_rate * 100).toFixed(4) : '0.0000';
        }
    }

    // 获取状态徽章样式
    getStatusBadgeClass(status) {
        const classes = {
            'ACTIVE': 'bg-success',
            'PENDING': 'bg-warning',
            'EXECUTED': 'bg-info',
            'CANCELLED': 'bg-secondary',
        };
        return classes[status] || 'bg-secondary';
    }

    // 更新收益数据
    async updateEarnings() {
        try {
            const earningsResponse = await api.getEarnings();
            const earnings = earningsResponse.data;
            
            console.log('[APP] 收益数据:', earnings); // 添加调试日志
            
            // 更新收益显示
            this.updateEarningsDisplay(earnings);
            
            // 更新图表
            chartManager.updateEarningsChart(earnings);
            chartManager.updateRateTrendChart(earnings);
            
        } catch (error) {
            console.error('获取收益数据失败:', error);
            Utils.showNotification(`获取收益数据失败: ${error.message}`, 'danger', 5000);
            throw error;
        }
    }

    // 更新收益显示
    updateEarningsDisplay(earnings) {
        const dailyElement = document.getElementById('daily-earnings');
        const weeklyElement = document.getElementById('weekly-earnings');
        const monthlyElement = document.getElementById('monthly-earnings');
        
        console.log('[APP] 更新收益显示:', {
            daily: earnings.daily,
            weekly: earnings.weekly, 
            monthly: earnings.monthly
        });
        
        if (dailyElement) {
            dailyElement.textContent = Utils.formatCurrency(earnings.daily);
        }
        
        if (weeklyElement) {
            weeklyElement.textContent = Utils.formatCurrency(earnings.weekly);
        }
        
        if (monthlyElement) {
            monthlyElement.textContent = Utils.formatCurrency(earnings.monthly);
        }
    }

    // 更新日志
    async updateLogs() {
        try {
            const response = await api.getLogs(1, 50);
            const logs = response.data.logs;
            
            this.updateLogDisplay(logs);
            
        } catch (error) {
            console.error('获取日志失败:', error);
        }
    }

    // 更新日志显示
    updateLogDisplay(logs) {
        const container = document.getElementById('log-container');
        if (!container || !logs) return;
        
        // 检查用户是否在底部附近
        const isNearBottom = container.scrollTop + container.clientHeight >= container.scrollHeight - 50;
        
        const logEntries = logs.map(log => `
            <div class="log-entry">
                <span class="log-message">${this.escapeHtml(log.message)}</span>
            </div>
        `).join('');
        
        container.innerHTML = logEntries || '<div class="p-3 text-muted">暂无日志</div>';
        
        // 如果用户在底部附近或者是首次加载，自动滚动到底部
        if (isNearBottom || container.scrollHeight === container.clientHeight) {
            container.scrollTop = container.scrollHeight;
        }
    }

    // HTML转义函数
    escapeHtml(text) {
        const map = {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#039;'
        };
        return text.replace(/[&<>"']/g, function(m) { return map[m]; });
    }

    // 启动机器人
    async startBot() {
        try {
            const response = await api.start();
            Utils.showNotification(response.message + ' - 请手动刷新查看最新状态', 'success');
        } catch (error) {
            Utils.showNotification('启动失败: ' + error.message, 'danger');
        }
    }

    // 停止机器人
    async stopBot() {
        try {
            const response = await api.stop();
            Utils.showNotification(response.message + ' - 请手动刷新查看最新状态', 'warning');
        } catch (error) {
            Utils.showNotification('停止失败: ' + error.message, 'danger');
        }
    }

    // 重启机器人
    async restartBot() {
        try {
            const response = await api.restart();
            Utils.showNotification(response.message + ' - 请手动刷新查看最新状态', 'info');
        } catch (error) {
            Utils.showNotification('重启失败: ' + error.message, 'danger');
        }
    }

    // 加载配置
    async loadConfig() {
        try {
            const response = await api.getConfig();
            const config = response.data;
            
            this.populateConfigForm(config);
            Utils.showNotification('配置加载成功', 'success');
            
        } catch (error) {
            Utils.showNotification('加载配置失败: ' + error.message, 'danger');
        }
    }

    // 填充配置表单
    populateConfigForm(config) {
        const form = document.getElementById('config-form');
        if (!form) return;
        
        Object.keys(config).forEach(key => {
            const element = form.querySelector(`[name="${key}"]`);
            if (element) {
                if (element.type === 'checkbox') {
                    element.checked = config[key];
                } else {
                    element.value = config[key];
                }
            }
        });
    }

    // 保存配置
    async saveConfig() {
        try {
            const form = document.getElementById('config-form');
            if (!form) return;
            
            const formData = new FormData(form);
            const config = {};
            
            for (const [key, value] of formData.entries()) {
                const element = form.querySelector(`[name="${key}"]`);
                if (element.type === 'checkbox') {
                    config[key] = element.checked;
                } else if (element.type === 'number') {
                    config[key] = parseFloat(value) || 0;
                } else {
                    config[key] = value;
                }
            }
            
            const response = await api.updateConfig(config);
            Utils.showNotification(response.message, 'success');
            
            // 关闭模态框
            const modal = bootstrap.Modal.getInstance(document.getElementById('config-modal'));
            modal?.hide();
            
        } catch (error) {
            Utils.showNotification('保存配置失败: ' + error.message, 'danger');
        }
    }

    // 启动日志轮询
    startLogPolling() {
        console.log('[APP] 启动日志实时轮询...');
        this.stopLogPolling(); // 先停止可能存在的轮询
        
        this.logUpdateTimer = setInterval(async () => {
            try {
                await this.updateLogs();
            } catch (error) {
                console.error('[APP] 日志轮询更新失败:', error);
            }
        }, this.logUpdateInterval);
        
        console.log('[APP] 日志轮询已启动，间隔:', this.logUpdateInterval, 'ms');
    }

    // 停止日志轮询
    stopLogPolling() {
        if (this.logUpdateTimer) {
            clearInterval(this.logUpdateTimer);
            this.logUpdateTimer = null;
            console.log('[APP] 日志轮询已停止');
        }
    }

    // 清除所有可能的定时器
    clearAllTimers() {
        console.log('[APP] 正在清除所有定时器...');
        
        // 停止日志轮询
        this.stopLogPolling();
        
        // 清除可能的全局定时器
        for (let i = 1; i < 10000; i++) {
            clearInterval(i);
            clearTimeout(i);
        }
        
        // 检查window对象上是否有定时器相关的属性
        Object.keys(window).forEach(key => {
            if (key.toLowerCase().includes('timer') || key.toLowerCase().includes('interval')) {
                console.log('[APP] 发现可疑的定时器属性:', key, window[key]);
                if (typeof window[key] === 'number') {
                    clearInterval(window[key]);
                    clearTimeout(window[key]);
                    window[key] = null;
                }
            }
        });
        
        console.log('[APP] 定时器清除完成');
    }

    // 检查是否有活跃的定时器
    checkActiveTimers() {
        console.log('[APP] 检查活跃定时器...');
        // 这个方法会在控制台显示当前的定时器信息
        console.log('[APP] setTimeout 数量:', window.setTimeout.toString());
        console.log('[APP] setInterval 数量:', window.setInterval.toString());
    }
}

// 页面加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    window.app = new BitfinexBotApp();
});