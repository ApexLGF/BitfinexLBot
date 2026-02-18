// 整体信息面板管理器
class OverallInfoPanel {
    constructor() {
        this.frrContainer = document.getElementById('frr-rates-container');
        this.earningsContainer = document.getElementById('yearly-earnings-container');
    }

    // 刷新所有数据
    async refreshData() {
        console.log('[OverallInfo] 刷新整体信息');

        try {
            const [frrRates, yearlyEarnings] = await Promise.all([
                api.get('/frr-rates'),
                api.get('/earnings/yearly')
            ]);

            this.updateFRRRates(frrRates.data);
            this.updateYearlyEarnings(yearlyEarnings.data);

            console.log('[OverallInfo] 整体信息刷新完成');
        } catch (error) {
            console.error('[OverallInfo] 刷新失败:', error);
            this.showError();
        }
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
    }
}

// 创建全局实例
window.overallInfo = new OverallInfoPanel();
