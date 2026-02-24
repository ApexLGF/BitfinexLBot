// 整体信息面板管理器
class OverallInfoPanel {
    constructor() {
        this.frrContainer = document.getElementById('frr-rates-container');
        this.earningsContainer = document.getElementById('yearly-earnings-container');
        this.walletTotalContainer = document.getElementById('wallet-total-container');
        // 年度收益缓存
        this.yearlyEarningsCache = null;
        this.yearlyEarningsCacheDate = null;
        // 钱包总额缓存
        this.walletTotalCache = null;
        this.walletTotalCacheDate = null;
    }

    // 刷新所有数据
    async refreshData() {
        console.log('[OverallInfo] 刷新整体信息');

        try {
            // FRR 利率每次都刷新
            await this.refreshFRROnly();

            // 钱包总额和年度收益使用缓存逻辑
            await Promise.all([
                this.refreshWalletTotal(),
                this.refreshYearlyOnly()
            ]);

            console.log('[OverallInfo] 整体信息刷新完成');
        } catch (error) {
            console.error('[OverallInfo] 刷新失败:', error);
            this.showError();
        }
    }

    // 只刷新 FRR 利率（快速）
    async refreshFRROnly() {
        console.log('[OverallInfo] 刷新 FRR 利率');

        try {
            const frrRates = await api.get('/frr-rates');
            this.updateFRRRates(frrRates.data);
            console.log('[OverallInfo] FRR 利率刷新完成');
        } catch (error) {
            console.error('[OverallInfo] FRR 刷新失败:', error);
            this.frrContainer.innerHTML = '<div class="text-center text-danger py-2"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
        }
    }

    // 刷新钱包总额（每天 UTC 1:35 后更新一次）
    async refreshWalletTotal() {
        console.log('[OverallInfo] 检查钱包总额是否需要刷新');

        if (this.shouldRefreshCache('walletTotal')) {
            console.log('[OverallInfo] 需要刷新钱包总额');
            try {
                const response = await api.getStatus();  // 无参数获取所有币种汇总
                this.walletTotalCache = response.data;
                this.walletTotalCacheDate = new Date().toISOString().split('T')[0];
                localStorage.setItem('walletTotalCache', JSON.stringify(this.walletTotalCache));
                localStorage.setItem('walletTotalCacheDate', this.walletTotalCacheDate);
                this.updateWalletTotal(this.walletTotalCache);
                console.log('[OverallInfo] 钱包总额刷新完成');
            } catch (error) {
                console.error('[OverallInfo] 钱包总额刷新失败:', error);
                if (this.walletTotalContainer) {
                    this.walletTotalContainer.innerHTML = '<div class="text-center text-danger py-2"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
                }
            }
        } else {
            console.log('[OverallInfo] 使用缓存的钱包总额');
            if (!this.walletTotalCache) {
                const cached = localStorage.getItem('walletTotalCache');
                if (cached) {
                    this.walletTotalCache = JSON.parse(cached);
                    this.walletTotalCacheDate = localStorage.getItem('walletTotalCacheDate');
                }
            }
            if (this.walletTotalCache) {
                this.updateWalletTotal(this.walletTotalCache);
            }
        }
    }

    // 只刷新年度收益（每天 UTC 1:35 后更新一次）
    async refreshYearlyOnly() {
        console.log('[OverallInfo] 检查年度收益是否需要刷新');

        if (this.shouldRefreshCache('yearlyEarnings')) {
            console.log('[OverallInfo] 需要刷新年度收益');
            try {
                const yearlyEarnings = await api.get('/earnings/yearly');
                this.yearlyEarningsCache = yearlyEarnings.data;
                this.yearlyEarningsCacheDate = new Date().toISOString().split('T')[0];
                localStorage.setItem('yearlyEarningsCache', JSON.stringify(this.yearlyEarningsCache));
                localStorage.setItem('yearlyEarningsCacheDate', this.yearlyEarningsCacheDate);
                this.updateYearlyEarnings(this.yearlyEarningsCache);
                console.log('[OverallInfo] 年度收益刷新完成');
            } catch (error) {
                console.error('[OverallInfo] 年度收益刷新失败:', error);
                this.earningsContainer.innerHTML = '<div class="text-center text-danger py-2"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
            }
        } else {
            console.log('[OverallInfo] 使用缓存的年度收益');
            if (!this.yearlyEarningsCache) {
                const cached = localStorage.getItem('yearlyEarningsCache');
                if (cached) {
                    this.yearlyEarningsCache = JSON.parse(cached);
                    this.yearlyEarningsCacheDate = localStorage.getItem('yearlyEarningsCacheDate');
                }
            }
            if (this.yearlyEarningsCache) {
                this.updateYearlyEarnings(this.yearlyEarningsCache);
            }
        }
    }

    // 判断是否需要刷新缓存（每天 UTC 1:35 后刷新一次）
    shouldRefreshCache(cacheType) {
        const now = new Date();
        const utcHour = now.getUTCHours();
        const utcMinute = now.getUTCMinutes();
        const todayDate = now.toISOString().split('T')[0];

        let cacheDate;
        if (cacheType === 'walletTotal') {
            cacheDate = this.walletTotalCacheDate || localStorage.getItem('walletTotalCacheDate');
        } else if (cacheType === 'yearlyEarnings') {
            cacheDate = this.yearlyEarningsCacheDate || localStorage.getItem('yearlyEarningsCacheDate');
        }

        // 如果没有缓存，需要刷新
        if (!cacheDate) {
            return true;
        }

        // 如果缓存日期不是今天，且当前时间已过 UTC 1:35，需要刷新
        if (cacheDate !== todayDate && (utcHour > 1 || (utcHour === 1 && utcMinute >= 35))) {
            return true;
        }

        return false;
    }

    // 更新 FRR 利率
    updateFRRRates(rates) {
        if (!rates || Object.keys(rates).length === 0) {
            this.frrContainer.innerHTML = '<div class="text-center text-muted py-2">暂无数据</div>';
            return;
        }

        const html = Object.entries(rates).map(([currency, rate]) => `
            <div class="frr-rate-item">
                <span class="frr-rate-currency">${currency}</span>
                <span class="frr-rate-value">${(rate * 100).toFixed(4)}%</span>
            </div>
        `).join('');

        this.frrContainer.innerHTML = html;
    }

    // 更新钱包总额显示
    updateWalletTotal(data) {
        if (!this.walletTotalContainer) return;

        if (!data || !data.currencies) {
            this.walletTotalContainer.innerHTML = '<div class="text-center text-muted py-2">暂无数据</div>';
            return;
        }

        const currencies = data.currencies;
        let grandTotal = 0;

        let html = '';
        for (const [currency, status] of Object.entries(currencies)) {
            const availableFunds = status.available_funds || 0;
            grandTotal += availableFunds;

            html += `
                <div class="wallet-total-item">
                    <span class="wallet-total-currency">${currency}</span>
                    <span class="wallet-total-value">${availableFunds.toFixed(2)}</span>
                </div>
            `;
        }

        // 添加总计
        html += `
            <hr class="my-2">
            <div class="wallet-total-item">
                <span class="wallet-total-currency fw-bold">可用总额</span>
                <span class="wallet-total-value fw-bold text-primary">${grandTotal.toFixed(2)}</span>
            </div>
        `;

        this.walletTotalContainer.innerHTML = html;
    }

    // 更新年度收益
    updateYearlyEarnings(data) {
        if (!data || !data.currencies) {
            this.earningsContainer.innerHTML = '<div class="text-center text-muted py-2">暂无数据</div>';
            return;
        }

        const currencies = data.currencies;
        const total = data.total || 0;

        let html = Object.entries(currencies).map(([currency, amount]) => `
            <div class="yearly-earnings-item">
                <span class="yearly-earnings-currency">${currency}</span>
                <span class="yearly-earnings-value">${amount.toFixed(2)}</span>
            </div>
        `).join('');

        // 添加总计
        html += `
            <hr class="my-2">
            <div class="yearly-earnings-item">
                <span class="yearly-earnings-currency fw-bold">总计</span>
                <span class="yearly-earnings-value fw-bold text-success">${total.toFixed(2)}</span>
            </div>
        `;

        this.earningsContainer.innerHTML = html;
    }

    // 显示错误
    showError() {
        this.frrContainer.innerHTML = '<div class="text-center text-danger py-2"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
        this.earningsContainer.innerHTML = '<div class="text-center text-danger py-2"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
        if (this.walletTotalContainer) {
            this.walletTotalContainer.innerHTML = '<div class="text-center text-danger py-2"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
        }
    }
}

// 创建全局实例
window.overallInfo = new OverallInfoPanel();
