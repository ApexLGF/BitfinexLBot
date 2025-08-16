// BitfinexBot 图表管理
class ChartManager {
    constructor() {
        this.charts = {};
        this.defaultOptions = {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'top',
                },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                },
            },
            scales: {
                x: {
                    display: true,
                    grid: {
                        display: false,
                    },
                },
                y: {
                    display: true,
                    grid: {
                        color: 'rgba(0, 0, 0, 0.1)',
                    },
                },
            },
            interaction: {
                mode: 'nearest',
                axis: 'x',
                intersect: false,
            },
        };
    }

    // 初始化收益图表
    initEarningsChart() {
        const ctx = document.getElementById('earnings-chart');
        if (!ctx) return;

        console.log('[CHART] 初始化收益图表为混合图表（柱状图+曲线图）');
        this.charts.earnings = new Chart(ctx, {
            type: 'bar', // 改为混合图表的基础类型
            data: {
                labels: [],
                datasets: [
                    {
                        label: '日收益',
                        data: [],
                        type: 'bar', // 柱状图
                        borderColor: 'rgb(75, 192, 192)',
                        backgroundColor: 'rgba(75, 192, 192, 0.7)',
                        borderWidth: 1,
                        borderRadius: 4,
                        borderSkipped: false,
                    },
                    {
                        label: '累计收益',
                        data: [],
                        type: 'line', // 曲线图
                        borderColor: 'rgb(54, 162, 235)',
                        backgroundColor: 'rgba(54, 162, 235, 0.1)',
                        borderWidth: 3,
                        fill: false,
                        tension: 0.4,
                        pointRadius: 5,
                        pointHoverRadius: 7,
                        pointBackgroundColor: 'rgb(54, 162, 235)',
                        pointBorderColor: '#fff',
                        pointBorderWidth: 2,
                        yAxisID: 'y1',
                    }
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false,
                },
                plugins: {
                    legend: {
                        position: 'top',
                        labels: {
                            usePointStyle: true,
                            padding: 20,
                        },
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        titleColor: '#fff',
                        bodyColor: '#fff',
                        borderColor: 'rgba(255, 255, 255, 0.1)',
                        borderWidth: 1,
                        callbacks: {
                            label: function(context) {
                                const label = context.dataset.label || '';
                                const value = Utils.formatCurrency(context.parsed.y);
                                return `${label}: ${value}`;
                            },
                        },
                    },
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            display: false,
                        },
                        title: {
                            display: true,
                            text: '日期',
                            font: {
                                size: 12,
                                weight: 'bold',
                            },
                        },
                    },
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        grid: {
                            color: 'rgba(0, 0, 0, 0.1)',
                        },
                        title: {
                            display: true,
                            text: '日收益 (USD)',
                            font: {
                                size: 12,
                                weight: 'bold',
                            },
                        },
                        ticks: {
                            callback: function(value) {
                                return '$' + value.toFixed(2);
                            },
                        },
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        grid: {
                            drawOnChartArea: false,
                        },
                        title: {
                            display: true,
                            text: '累计收益 (USD)',
                            font: {
                                size: 12,
                                weight: 'bold',
                            },
                        },
                        ticks: {
                            callback: function(value) {
                                return '$' + value.toFixed(2);
                            },
                        },
                    },
                },
            },
        });
    }

    // 更新收益图表数据
    updateEarningsChart(data) {
        const chart = this.charts.earnings;
        if (!chart || !data || !data.history) return;

        const labels = data.history.map(item => item.date);
        const dailyEarnings = data.history.map(item => item.amount);
        
        // 计算累计收益
        let cumulative = 0;
        const cumulativeEarnings = dailyEarnings.map(amount => {
            cumulative += amount;
            return cumulative;
        });

        chart.data.labels = labels;
        chart.data.datasets[0].data = dailyEarnings;
        chart.data.datasets[1].data = cumulativeEarnings;
        
        chart.update('active');
    }

    // 初始化利率分布图表
    initRateDistributionChart() {
        const ctx = document.getElementById('rate-distribution-chart');
        if (!ctx) return;

        this.charts.rateDistribution = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['2天', '30天', '120天'],
                datasets: [{
                    data: [0, 0, 0],
                    backgroundColor: [
                        'rgba(255, 99, 132, 0.8)',
                        'rgba(54, 162, 235, 0.8)',
                        'rgba(255, 205, 86, 0.8)',
                    ],
                    borderColor: [
                        'rgb(255, 99, 132)',
                        'rgb(54, 162, 235)',
                        'rgb(255, 205, 86)',
                    ],
                    borderWidth: 2,
                }],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const label = context.label || '';
                                const value = Utils.formatCurrency(context.parsed);
                                const percentage = ((context.parsed / context.dataset.data.reduce((a, b) => a + b, 0)) * 100).toFixed(1);
                                return `${label}: ${value} (${percentage}%)`;
                            },
                        },
                    },
                },
            },
        });
    }

    // 更新利率分布图表
    updateRateDistributionChart(offers) {
        const chart = this.charts.rateDistribution;
        if (!chart || !offers) return;

        const distribution = { 2: 0, 30: 0, 120: 0 };
        
        offers.forEach(offer => {
            if (offer.period <= 2) {
                distribution[2] += offer.amount;
            } else if (offer.period <= 30) {
                distribution[30] += offer.amount;
            } else {
                distribution[120] += offer.amount;
            }
        });

        chart.data.datasets[0].data = [distribution[2], distribution[30], distribution[120]];
        chart.update('active');
    }

    // 初始化利率趋势图表
    initRateTrendChart() {
        const ctx = document.getElementById('rate-trend-chart');
        if (!ctx) return;

        this.charts.rateTrend = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: '平均利率',
                        data: [],
                        borderColor: 'rgb(75, 192, 192)',
                        backgroundColor: 'rgba(75, 192, 192, 0.1)',
                        borderWidth: 2,
                        fill: true,
                        tension: 0.4,
                    },
                    {
                        label: '最高利率',
                        data: [],
                        borderColor: 'rgb(255, 99, 132)',
                        backgroundColor: 'rgba(255, 99, 132, 0.1)',
                        borderWidth: 2,
                        fill: false,
                        tension: 0.4,
                    },
                    {
                        label: '最低利率',
                        data: [],
                        borderColor: 'rgb(54, 162, 235)',
                        backgroundColor: 'rgba(54, 162, 235, 0.1)',
                        borderWidth: 2,
                        fill: false,
                        tension: 0.4,
                    }
                ],
            },
            options: {
                ...this.defaultOptions,
                scales: {
                    ...this.defaultOptions.scales,
                    y: {
                        ...this.defaultOptions.scales.y,
                        title: {
                            display: true,
                            text: '利率 (%)',
                        },
                        ticks: {
                            callback: function(value) {
                                return (value * 100).toFixed(3) + '%';
                            },
                        },
                    },
                },
                plugins: {
                    ...this.defaultOptions.plugins,
                    tooltip: {
                        ...this.defaultOptions.plugins.tooltip,
                        callbacks: {
                            label: function(context) {
                                const label = context.dataset.label || '';
                                const value = Utils.formatPercentage(context.parsed.y);
                                return `${label}: ${value}`;
                            },
                        },
                    },
                },
            },
        });
    }

    // 更新利率趋势图表
    updateRateTrendChart(data) {
        const chart = this.charts.rateTrend;
        if (!chart || !data || !data.history) return;

        const labels = data.history.map(item => item.date);
        const rates = data.history.map(item => item.rate);
        
        // 计算移动平均
        const avgRates = this.calculateMovingAverage(rates, 7);
        const maxRates = rates.map((_, index) => Math.max(...rates.slice(Math.max(0, index - 6), index + 1)));
        const minRates = rates.map((_, index) => Math.min(...rates.slice(Math.max(0, index - 6), index + 1)));

        chart.data.labels = labels;
        chart.data.datasets[0].data = avgRates;
        chart.data.datasets[1].data = maxRates;
        chart.data.datasets[2].data = minRates;
        
        chart.update('active');
    }

    // 计算移动平均
    calculateMovingAverage(data, windowSize) {
        const result = [];
        for (let i = 0; i < data.length; i++) {
            const start = Math.max(0, i - windowSize + 1);
            const end = i + 1;
            const window = data.slice(start, end);
            const average = window.reduce((sum, value) => sum + value, 0) / window.length;
            result.push(average);
        }
        return result;
    }

    // 初始化所有图表
    initAllCharts() {
        this.initEarningsChart();
        this.initRateDistributionChart();
        this.initRateTrendChart();
    }

    // 销毁图表
    destroyChart(chartName) {
        if (this.charts[chartName]) {
            this.charts[chartName].destroy();
            delete this.charts[chartName];
        }
    }

    // 销毁所有图表
    destroyAllCharts() {
        Object.keys(this.charts).forEach(chartName => {
            this.destroyChart(chartName);
        });
    }

    // 调整图表大小
    resizeCharts() {
        Object.values(this.charts).forEach(chart => {
            chart.resize();
        });
    }

    // 更新图表主题
    updateTheme(isDark = false) {
        const textColor = isDark ? '#ffffff' : '#333333';
        const gridColor = isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';
        
        Object.values(this.charts).forEach(chart => {
            if (chart.options.scales) {
                Object.values(chart.options.scales).forEach(scale => {
                    if (scale.ticks) scale.ticks.color = textColor;
                    if (scale.grid) scale.grid.color = gridColor;
                    if (scale.title) scale.title.color = textColor;
                });
            }
            
            if (chart.options.plugins && chart.options.plugins.legend) {
                chart.options.plugins.legend.labels = {
                    ...chart.options.plugins.legend.labels,
                    color: textColor,
                };
            }
            
            chart.update('none');
        });
    }
}

// 创建全局图表管理器实例
window.chartManager = new ChartManager();