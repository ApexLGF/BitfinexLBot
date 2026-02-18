// 币种详情面板管理器
class CurrencyDashboard {
    constructor() {
        this.charts = {}; // 存储每个币种的图表实例
    }

    // 渲染指定币种的面板
    async render(currency) {
        console.log(`[Dashboard] 渲染币种面板: ${currency}`);

        try {
            // 并行获取数据
            const [earnings, offers, credits, status] = await Promise.all([
                api.getEarnings(`?currency=${currency}`),
                api.getOffers(`?currency=${currency}`),
                api.getFundingCredits(`?currency=${currency}`),
                api.getStatus(`?currency=${currency}`)
            ]);

            // 更新各个部分
            this.updateEarningsChart(currency, earnings.data);
            this.updateOffersSection(currency, offers.data);
            this.updateCreditsSection(currency, credits.data);

            // 更新 Tab 统计信息
            this.updateTabStats(currency, offers.data, credits.data, status.data, earnings.data);

            console.log(`[Dashboard] ${currency} 面板渲染完成`);
        } catch (error) {
            console.error(`[Dashboard] 渲染 ${currency} 面板失败:`, error);
        }
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
                maintainAspectRatio: true,
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

    // 更新贷出挂单部分
    updateOffersSection(currency, offers) {
        // 更新统计数据
        const count = offers.length;
        const total = offers.reduce((sum, o) => sum + o.amount, 0);
        const avgRate = count > 0 ? offers.reduce((sum, o) => sum + o.rate, 0) / count : 0;

        document.getElementById(`offers-count-${currency}`).textContent = count;
        document.getElementById(`offers-total-${currency}`).textContent = total.toFixed(2);
        document.getElementById(`offers-rate-${currency}`).textContent = (avgRate * 100).toFixed(4);

        // 更新表格
        const tbody = document.querySelector(`#offers-table-${currency} tbody`);
        if (!tbody) return;

        if (offers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">暂无数据</td></tr>';
            return;
        }

        tbody.innerHTML = offers.map(offer => `
            <tr>
                <td>${offer.id}</td>
                <td>${offer.amount.toFixed(2)}</td>
                <td>${(offer.rate * 100).toFixed(4)}</td>
                <td>${offer.period}天</td>
                <td><span class="badge bg-success">活跃</span></td>
                <td>${new Date(offer.created).toLocaleString('zh-CN')}</td>
            </tr>
        `).join('');
    }

    // 更新已贷出订单部分
    updateCreditsSection(currency, creditsData) {
        const credits = creditsData.credits || [];

        // 更新统计数据
        document.getElementById(`credits-count-${currency}`).textContent = creditsData.total_count || 0;
        document.getElementById(`credits-total-${currency}`).textContent = (creditsData.total_amount || 0).toFixed(2);
        document.getElementById(`credits-rate-${currency}`).textContent = ((creditsData.avg_rate || 0) * 100).toFixed(4);

        // 更新表格
        const tbody = document.querySelector(`#credits-table-${currency} tbody`);
        if (!tbody) return;

        if (credits.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">暂无数据</td></tr>';
            return;
        }

        tbody.innerHTML = credits.map(credit => `
            <tr>
                <td>${credit.id}</td>
                <td>${credit.amount.toFixed(2)}</td>
                <td>${(credit.rate * 100).toFixed(4)}</td>
                <td>${credit.period}天</td>
                <td><span class="badge bg-info">${credit.status}</span></td>
                <td>${new Date(credit.opened).toLocaleString('zh-CN')}</td>
            </tr>
        `).join('');
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
