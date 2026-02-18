// BitfinexBot API 客户端
class BitfinexAPI {
    constructor(baseURL = '') {
        this.baseURL = baseURL;
        this.apiPrefix = '/api';
    }

    // 通用请求方法
    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${this.apiPrefix}${endpoint}`;
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
            },
        };

        const config = { ...defaultOptions, ...options };

        try {
            const response = await fetch(url, config);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}: ${response.statusText}`);
            }

            return data;
        } catch (error) {
            console.error('API请求失败:', error);
            throw error;
        }
    }

    // GET 请求
    async get(endpoint) {
        return this.request(endpoint, { method: 'GET' });
    }

    // POST 请求
    async post(endpoint, data) {
        return this.request(endpoint, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    // 获取机器人状态
    async getStatus(queryString = '') {
        return this.get(`/status${queryString}`);
    }

    // 获取收益数据
    async getEarnings(queryString = '') {
        return this.get(`/earnings${queryString}`);
    }

    // 获取日收益
    async getDailyEarnings() {
        return this.get('/earnings/daily');
    }

    // 获取周收益
    async getWeeklyEarnings() {
        return this.get('/earnings/weekly');
    }

    // 获取放贷订单
    async getOffers(queryString = '') {
        console.log('[API] getOffers 被调用 - ', new Date().toISOString(), '- 调用栈:', new Error().stack);
        return this.get(`/offers${queryString}`);
    }

    // 获取活跃订单
    async getActiveOffers(queryString = '') {
        return this.get(`/offers/active${queryString}`);
    }

    // 获取已贷出订单
    async getFundingCredits(queryString = '') {
        return this.get(`/credits${queryString}`);
    }

    // 获取配置
    async getConfig() {
        return this.get('/config');
    }

    // 更新配置
    async updateConfig(config) {
        return this.post('/config', { config });
    }

    // 控制机器人
    async control(action) {
        return this.post('/control', { action });
    }

    // 启动机器人
    async start() {
        return this.control('start');
    }

    // 停止机器人
    async stop() {
        return this.control('stop');
    }

    // 重启机器人
    async restart() {
        return this.control('restart');
    }

    // 获取日志
    async getLogs(page = 1, limit = 100) {
        return this.get(`/logs?page=${page}&limit=${limit}`);
    }

    // 获取系统信息
    async getSystemInfo() {
        return this.get('/info');
    }

    // 获取钱包信息
    async getWallets() {
        return this.get('/wallets');
    }

    // 健康检查
    async healthCheck() {
        try {
            const response = await fetch(`${this.baseURL}/health`);
            return response.ok;
        } catch (error) {
            return false;
        }
    }
}

// 创建全局API实例
window.api = new BitfinexAPI();

// 工具函数
class Utils {
    // 格式化货币
    static formatCurrency(amount, currency = 'USD', decimals = 2) {
        if (amount === null || amount === undefined || isNaN(amount)) {
            return '-';
        }
        
        // 处理非标准货币代码
        const currencyMap = {
            'UST': 'USD',  // 将UST映射为USD进行格式化
            'USDT': 'USD'  // 将USDT映射为USD进行格式化
        };
        
        const mappedCurrency = currencyMap[currency] || currency;
        
        try {
            const formatter = new Intl.NumberFormat('zh-CN', {
                style: 'currency',
                currency: mappedCurrency,
                minimumFractionDigits: decimals,
                maximumFractionDigits: decimals,
            });
            
            let formatted = formatter.format(amount);
            
            // 如果是非标准货币，替换显示的货币符号
            if (currencyMap[currency]) {
                formatted = formatted.replace(/USD|US\$|\$/, currency);
            }
            
            return formatted;
        } catch (error) {
            // 如果格式化失败，使用简单格式
            console.warn('货币格式化失败:', error);
            return `${amount.toFixed(decimals)} ${currency}`;
        }
    }

    // 格式化百分比
    static formatPercentage(rate, decimals = 4) {
        if (rate === null || rate === undefined || isNaN(rate)) {
            return '-';
        }
        
        return `${(rate * 100).toFixed(decimals)}%`;
    }

    // 格式化时间
    static formatTime(date) {
        if (!date) return '-';
        
        const now = new Date();
        const targetDate = new Date(date);
        const diff = Math.abs(now - targetDate);
        
        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        
        if (days > 0) {
            return `${days}天前`;
        } else if (hours > 0) {
            return `${hours}小时前`;
        } else if (minutes > 0) {
            return `${minutes}分钟前`;
        } else {
            return '刚刚';
        }
    }

    // 格式化相对时间
    static formatRelativeTime(date) {
        if (!date) return '-';
        
        const now = new Date();
        const targetDate = new Date(date);
        const diff = targetDate - now;
        
        if (diff < 0) {
            return '已过期';
        }
        
        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            return `${hours}小时${minutes % 60}分钟后`;
        } else if (minutes > 0) {
            return `${minutes}分钟后`;
        } else {
            return `${seconds}秒后`;
        }
    }

    // 显示通知
    static showNotification(message, type = 'info', duration = 3000) {
        const notification = document.createElement('div');
        notification.className = `alert alert-${type} notification`;
        notification.innerHTML = `
            <div class="d-flex align-items-center">
                <i class="bi bi-${this.getNotificationIcon(type)} me-2"></i>
                <span>${message}</span>
                <button type="button" class="btn-close ms-auto" onclick="this.parentElement.parentElement.remove()"></button>
            </div>
        `;
        
        document.body.appendChild(notification);
        
        // 显示动画
        setTimeout(() => notification.classList.add('show'), 10);
        
        // 自动隐藏
        if (duration > 0) {
            setTimeout(() => {
                notification.classList.remove('show');
                setTimeout(() => notification.remove(), 300);
            }, duration);
        }
    }

    // 获取通知图标
    static getNotificationIcon(type) {
        const icons = {
            success: 'check-circle',
            danger: 'exclamation-triangle',
            warning: 'exclamation-triangle',
            info: 'info-circle',
            primary: 'info-circle',
        };
        return icons[type] || 'info-circle';
    }

    // 防抖函数
    static debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func.apply(this, args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    // 节流函数
    static throttle(func, limit) {
        let inThrottle;
        return function executedFunction(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }

    // 数字动画
    static animateNumber(element, start, end, duration = 1000) {
        if (!element) return;
        
        const startTime = performance.now();
        const difference = end - start;
        
        const step = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // 使用缓动函数
            const easedProgress = this.easeOutQuart(progress);
            const current = start + (difference * easedProgress);
            
            element.textContent = Math.round(current);
            
            if (progress < 1) {
                requestAnimationFrame(step);
            }
        };
        
        requestAnimationFrame(step);
    }

    // 缓动函数
    static easeOutQuart(t) {
        return 1 - Math.pow(1 - t, 4);
    }

    // 复制到剪贴板
    static async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            this.showNotification('已复制到剪贴板', 'success', 1500);
        } catch (error) {
            console.error('复制失败:', error);
            this.showNotification('复制失败', 'danger', 1500);
        }
    }

    // 下载JSON文件
    static downloadJSON(data, filename = 'config.json') {
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }
}

// 将工具类设为全局可用
window.Utils = Utils;