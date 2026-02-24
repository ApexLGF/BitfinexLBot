// 币种详情面板管理器
class CurrencyDashboard {
    constructor() {
        this.charts = {}; // 存储每个币种的图表实例
        this.dataCache = {}; // 数据缓存
    }

    // 缓存币种数据
    cacheData(currency, data) {
        this.dataCache[currency] = {
            data: data,
            timestamp: Date.now()
        };
        console.log(`[Dashboard] 缓存 ${currency} 数据`);
    }

    // 获取缓存数据
    getCachedData(currency) {
        const cached = this.dataCache[currency];
        if (!cached) {
            return null;
        }

        // 检查缓存是否过期（5分钟）
        const age = Date.now() - cached.timestamp;
        if (age > 5 * 60 * 1000) {
            console.log(`[Dashboard] ${currency} 缓存已过期`);
            return null;
        }

        return cached.data;
    }

    // 清除所有缓存
    clearCache() {
        this.dataCache = {};
        console.log('[Dashboard] 缓存已清除');
    }

    // 渲染指定币种的面板
    async render(currency) {
        console.log(`[Dashboard] 渲染币种面板: ${currency}`);

        try {
            // 先尝试使用缓存数据
            let cachedData = this.getCachedData(currency);

            if (cachedData) {
                console.log(`[Dashboard] 使用缓存数据渲染 ${currency}`);
                this.renderWithData(currency, cachedData);
                return;
            }

            // 缓存不存在或已过期，重新获取
            console.log(`[Dashboard] 重新获取 ${currency} 数据`);
            const [earnings, offers, credits, status] = await Promise.all([
                api.getEarnings(`?currency=${currency}`),
                api.getOffers(`?currency=${currency}`),
                api.getFundingCredits(`?currency=${currency}`),
                api.getStatus(`?currency=${currency}`)
            ]);

            const data = {
                earnings: earnings.data,
                offers: offers.data,
                credits: credits.data,
                status: status.data
            };

            // 缓存数据
            this.cacheData(currency, data);

            // 渲染
            this.renderWithData(currency, data);

            console.log(`[Dashboard] ${currency} 面板渲染完成`);
        } catch (error) {
            console.error(`[Dashboard] 渲染 ${currency} 面板失败:`, error);
            throw error;
        }
    }

    // 使用数据渲染
    renderWithData(currency, data) {
        this.updateEarningsChart(currency, data.earnings);
        this.updateOffersSection(currency, data.offers);
        this.updateCreditsSection(currency, data.credits);
        this.updateTabStats(currency, data.offers, data.credits, data.status, data.earnings);
    }

    // 更新 Tab 统计信息
    updateTabStats(currency, offers, creditsData, status, earnings) {
        // 计算已挂单总额
        const offersTotal = offers.reduce((sum, o) => sum + o.amount, 0);

        // 计算已贷出总额
        const creditsTotal = creditsData.total_amount || 0;

        // 获取可用余额
        const available = status.available_funds || 0;

        // 计算总金额
        const total = offersTotal + creditsTotal + available;

        // 更新统计卡片显示
        const creditsElement = document.getElementById(`stat-credits-${currency}`);
        const offersElement = document.getElementById(`stat-offers-${currency}`);
        const availableElement = document.getElementById(`stat-available-${currency}`);
        const totalElement = document.getElementById(`stat-total-${currency}`);

        if (creditsElement) creditsElement.textContent = creditsTotal.toFixed(2);
        if (offersElement) offersElement.textContent = offersTotal.toFixed(2);
        if (availableElement) availableElement.textContent = available.toFixed(2);
        if (totalElement) totalElement.textContent = total.toFixed(2);

        // 更新收益统计显示
        if (earnings) {
            const dailyElement = document.getElementById(`stat-daily-${currency}`);
            const weeklyElement = document.getElementById(`stat-weekly-${currency}`);
            const monthlyElement = document.getElementById(`stat-monthly-${currency}`);

            if (dailyElement) dailyElement.textContent = (earnings.daily || 0).toFixed(4);
            if (weeklyElement) weeklyElement.textContent = (earnings.weekly || 0).toFixed(4);
            if (monthlyElement) monthlyElement.textContent = (earnings.monthly || 0).toFixed(4);
        }

        // 更新右侧钱包总额显示
        if (window.overallInfo) {
            overallInfo.updateWalletTotalFromCache();
        }
    }

    // 更新收益图表
    updateEarningsChart(currency, earnings) {
        const chartId = `earnings-chart-${currency}`;
        const canvas = document.getElementById(chartId);

        if (!canvas) {
            console.warn(`[Dashboard] 找不到图表元素: ${chartId}`);
            return;
        }

        // 如果图表已存在，先销毁
        if (this.charts[chartId]) {
            this.charts[chartId].destroy();
        }

        // 准备数据
        const labels = earnings.history.map(h => h.date.substring(5)); // 只显示月-日
        const dailyData = earnings.history.map(h => h.amount);

        // 计算累计收益
        const cumulativeData = [];
        let cumulative = 0;
        for (const amount of dailyData) {
            cumulative += amount;
            cumulativeData.push(cumulative);
        }

        // 创建混合图表（柱状图 + 折线图）
        const ctx = canvas.getContext('2d');
        this.charts[chartId] = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: labels,
                datasets: [
                    {
                        type: 'bar',
                        label: '日收益',
                        data: dailyData,
                        backgroundColor: 'rgba(13, 110, 253, 0.5)',
                        borderColor: 'rgba(13, 110, 253, 1)',
                        borderWidth: 1,
                        yAxisID: 'y'
                    },
                    {
                        type: 'line',
                        label: '累计收益',
                        data: cumulativeData,
                        borderColor: 'rgba(25, 135, 84, 1)',
                        backgroundColor: 'rgba(25, 135, 84, 0.1)',
                        borderWidth: 2,
                        fill: true,
                        tension: 0.4,
                        yAxisID: 'y1'
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false
                },
                plugins: {
                    legend: {
                        display: true,
                        position: 'top'
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                let label = context.dataset.label || '';
                                if (label) {
                                    label += ': ';
                                }
                                label += context.parsed.y.toFixed(2);
                                return label;
                            }
                        }
                    }
                },
                scales: {
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        title: {
                            display: true,
                            text: '日收益'
                        }
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: '累计收益'
                        },
                        grid: {
                            drawOnChartArea: false
                        }
                    }
                }
            }
        });
    }

    // 更新贷出挂单部分（只更新统计数据，不加载表格）
    updateOffersSection(currency, offers) {
        // 更新统计数据
        const count = offers.length;
        const total = offers.reduce((sum, o) => sum + o.amount, 0);
        const avgRate = count > 0 ? offers.reduce((sum, o) => sum + o.rate, 0) / count : 0;

        const countEl = document.getElementById(`offers-count-${currency}`);
        const totalEl = document.getElementById(`offers-total-${currency}`);
        const rateEl = document.getElementById(`offers-rate-${currency}`);

        if (countEl) countEl.textContent = count;
        if (totalEl) totalEl.textContent = total.toFixed(2);
        if (rateEl) rateEl.textContent = (avgRate * 100).toFixed(4);
    }

    // 更新已贷出订单部分（只更新统计数据，不加载表格）
    updateCreditsSection(currency, creditsData) {
        const countEl = document.getElementById(`credits-count-${currency}`);
        const totalEl = document.getElementById(`credits-total-${currency}`);
        const rateEl = document.getElementById(`credits-rate-${currency}`);

        if (countEl) countEl.textContent = creditsData.total_count || 0;
        if (totalEl) totalEl.textContent = (creditsData.total_amount || 0).toFixed(2);
        if (rateEl) rateEl.textContent = ((creditsData.avg_rate || 0) * 100).toFixed(4);
    }

    // 显示贷出挂单详情（点击按钮时才加载数据）
    async showOffersDetail(currency) {
        const content = document.getElementById('offers-detail-content');
        content.innerHTML = '<div class="text-center py-3"><span class="spinner-border spinner-border-sm me-2"></span>加载中...</div>';

        // 打开模态框
        const modal = new bootstrap.Modal(document.getElementById('offers-detail-modal'));
        modal.show();

        try {
            // 点击时才从 API 加载数据
            const response = await api.getOffers(`?currency=${currency}`);
            const offers = response.data || [];

            if (offers.length === 0) {
                content.innerHTML = '<div class="text-center text-muted py-3">暂无挂单数据</div>';
            } else {
                content.innerHTML = `
                    <div class="table-responsive">
                        <table class="table table-sm table-hover">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>金额</th>
                                    <th>利率</th>
                                    <th>期间</th>
                                    <th>状态</th>
                                    <th>创建时间</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${offers.map(offer => `
                                    <tr>
                                        <td>${offer.id}</td>
                                        <td>${offer.amount.toFixed(2)}</td>
                                        <td>${(offer.rate * 100).toFixed(4)}%</td>
                                        <td>${offer.period}天</td>
                                        <td><span class="badge bg-success">活跃</span></td>
                                        <td>${new Date(offer.created).toLocaleString('zh-CN')}</td>
                                    </tr>
                                `).join('')}
                            </tbody>
                        </table>
                    </div>
                `;
            }
        } catch (error) {
            console.error('[Dashboard] 加载挂单详情失败:', error);
            content.innerHTML = '<div class="text-center text-danger py-3"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
        }
    }

    // 显示已贷出订单详情（点击按钮时才加载数据）
    async showCreditsDetail(currency) {
        const content = document.getElementById('credits-detail-content');
        content.innerHTML = '<div class="text-center py-3"><span class="spinner-border spinner-border-sm me-2"></span>加载中...</div>';

        // 打开模态框
        const modal = new bootstrap.Modal(document.getElementById('credits-detail-modal'));
        modal.show();

        try {
            // 点击时才从 API 加载数据
            const response = await api.getFundingCredits(`?currency=${currency}`);
            const credits = response.data?.credits || [];

            if (credits.length === 0) {
                content.innerHTML = '<div class="text-center text-muted py-3">暂无已贷出订单</div>';
            } else {
                content.innerHTML = `
                    <div class="table-responsive">
                        <table class="table table-sm table-hover">
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>金额</th>
                                    <th>利率</th>
                                    <th>期间</th>
                                    <th>状态</th>
                                    <th>开始时间</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${credits.map(credit => `
                                    <tr>
                                        <td>${credit.id}</td>
                                        <td>${credit.amount.toFixed(2)}</td>
                                        <td>${(credit.rate * 100).toFixed(4)}%</td>
                                        <td>${credit.period}天</td>
                                        <td><span class="badge bg-info">${credit.status}</span></td>
                                        <td>${new Date(credit.opened).toLocaleString('zh-CN')}</td>
                                    </tr>
                                `).join('')}
                            </tbody>
                        </table>
                    </div>
                `;
            }
        } catch (error) {
            console.error('[Dashboard] 加载已贷出订单详情失败:', error);
            content.innerHTML = '<div class="text-center text-danger py-3"><i class="bi bi-exclamation-triangle me-1"></i>加载失败</div>';
        }
    }

    // 销毁所有图表
    destroyAllCharts() {
        Object.values(this.charts).forEach(chart => {
            if (chart) chart.destroy();
        });
        this.charts = {};
    }
}

// 创建全局实例
window.dashboard = new CurrencyDashboard();
