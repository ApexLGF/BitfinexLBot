// 币种 TAB 管理器
class CurrencyTabManager {
    constructor() {
        this.tabsContainer = document.getElementById('currency-tabs');
        this.contentContainer = document.getElementById('currency-tab-content');
        this.activeCurrency = null;
        this.currencies = [];
    }

    // 初始化 TAB
    initTabs(currencies) {
        if (!currencies || currencies.length === 0) {
            console.warn('[TabManager] 没有可用的币种');
            return;
        }

        this.currencies = currencies;
        this.activeCurrency = currencies[0]; // 默认选择第一个币种

        // 清空现有内容
        this.tabsContainer.innerHTML = '';
        this.contentContainer.innerHTML = '';

        // 创建每个币种的 TAB
        currencies.forEach((currency, index) => {
            this.createTab(currency, index === 0);
            this.createTabContent(currency, index === 0);
        });

        console.log(`[TabManager] 已初始化 ${currencies.length} 个币种 TAB`);
    }

    // 创建 TAB 按钮
    createTab(currency, isActive) {
        const li = document.createElement('li');
        li.className = 'nav-item';
        li.setAttribute('role', 'presentation');

        const button = document.createElement('button');
        button.className = `nav-link ${isActive ? 'active' : ''}`;
        button.id = `tab-${currency}`;
        button.setAttribute('data-bs-toggle', 'tab');
        button.setAttribute('data-bs-target', `#content-${currency}`);
        button.setAttribute('type', 'button');
        button.setAttribute('role', 'tab');
        button.textContent = currency.toUpperCase();

        // 添加点击事件
        button.addEventListener('click', () => {
            this.switchTab(currency);
        });

        li.appendChild(button);
        this.tabsContainer.appendChild(li);
    }

    // 创建 TAB 内容区域
    createTabContent(currency, isActive) {
        const div = document.createElement('div');
        div.className = `tab-pane fade ${isActive ? 'show active' : ''}`;
        div.id = `content-${currency}`;
        div.setAttribute('role', 'tabpanel');

        div.innerHTML = `
            <!-- 币种统计卡片 -->
            <div class="card mb-3">
                <div class="card-header">
                    <i class="bi bi-wallet2 me-2"></i>${currency.toUpperCase()} 资金统计
                </div>
                <div class="card-body">
                    <div class="row g-3" id="currency-stats-${currency}">
                        <div class="col-md-3 col-6">
                            <div class="stat-item">
                                <div class="stat-label text-muted">已贷出</div>
                                <div class="stat-value text-success" id="stat-credits-${currency}">0</div>
                            </div>
                        </div>
                        <div class="col-md-3 col-6">
                            <div class="stat-item">
                                <div class="stat-label text-muted">已挂单</div>
                                <div class="stat-value text-primary" id="stat-offers-${currency}">0</div>
                            </div>
                        </div>
                        <div class="col-md-3 col-6">
                            <div class="stat-item">
                                <div class="stat-label text-muted">剩余可用</div>
                                <div class="stat-value text-warning" id="stat-available-${currency}">0</div>
                            </div>
                        </div>
                        <div class="col-md-3 col-6">
                            <div class="stat-item">
                                <div class="stat-label text-muted">总金额</div>
                                <div class="stat-value text-info" id="stat-total-${currency}">0</div>
                            </div>
                        </div>
                    </div>
                    <hr class="my-3">
                    <h6 class="card-subtitle mb-2 text-muted">收益统计</h6>
                    <div class="row g-3">
                        <div class="col-md-4 col-4">
                            <div class="stat-item">
                                <div class="stat-label text-muted">日收益</div>
                                <div class="stat-value text-success" style="font-size: 1.2rem;" id="stat-daily-${currency}">0</div>
                            </div>
                        </div>
                        <div class="col-md-4 col-4">
                            <div class="stat-item">
                                <div class="stat-label text-muted">周收益</div>
                                <div class="stat-value text-primary" style="font-size: 1.2rem;" id="stat-weekly-${currency}">0</div>
                            </div>
                        </div>
                        <div class="col-md-4 col-4">
                            <div class="stat-item">
                                <div class="stat-label text-muted">月收益</div>
                                <div class="stat-value text-info" style="font-size: 1.2rem;" id="stat-monthly-${currency}">0</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- 30日收益趋势图 -->
            <div class="card mb-3">
                <div class="card-header">
                    <i class="bi bi-graph-up me-2"></i>30日收益趋势
                </div>
                <div class="card-body">
                    <canvas id="earnings-chart-${currency}"></canvas>
                </div>
            </div>

            <!-- 贷出挂单 -->
            <div class="card mb-3">
                <div class="card-header d-flex justify-content-between align-items-center">
                    <span><i class="bi bi-list-ul me-2"></i>贷出挂单</span>
                    <div class="stats-badges">
                        <span class="badge bg-primary">数量: <span id="offers-count-${currency}">0</span></span>
                        <span class="badge bg-success">总额: <span id="offers-total-${currency}">0</span></span>
                        <span class="badge bg-info">平均利率: <span id="offers-rate-${currency}">0</span></span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="table-responsive">
                        <table class="table table-sm table-hover" id="offers-table-${currency}">
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
                                <tr>
                                    <td colspan="6" class="text-center text-muted">暂无数据</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <!-- 已贷出订单 -->
            <div class="card mb-3">
                <div class="card-header d-flex justify-content-between align-items-center">
                    <span><i class="bi bi-check-circle me-2"></i>已贷出订单</span>
                    <div class="stats-badges">
                        <span class="badge bg-primary">数量: <span id="credits-count-${currency}">0</span></span>
                        <span class="badge bg-success">总额: <span id="credits-total-${currency}">0</span></span>
                        <span class="badge bg-info">平均利率: <span id="credits-rate-${currency}">0</span></span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="table-responsive">
                        <table class="table table-sm table-hover" id="credits-table-${currency}">
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
                                <tr>
                                    <td colspan="6" class="text-center text-muted">暂无数据</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        `;

        this.contentContainer.appendChild(div);
    }

    // 切换 TAB
    switchTab(currency) {
        if (this.activeCurrency === currency) {
            return; // 已经是当前 TAB，无需切换
        }

        console.log(`[TabManager] 切换到币种: ${currency}`);
        this.activeCurrency = currency;

        // 触发自定义事件，通知其他组件
        const event = new CustomEvent('currencyChanged', {
            detail: { currency }
        });
        document.dispatchEvent(event);
    }

    // 获取当前激活的币种
    getActiveCurrency() {
        return this.activeCurrency;
    }

    // 获取所有币种
    getCurrencies() {
        return this.currencies;
    }
}

// 创建全局实例
window.tabManager = new CurrencyTabManager();
