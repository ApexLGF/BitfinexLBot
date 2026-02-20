// 配置管理器
class ConfigManager {
    constructor() {
        this.modal = document.getElementById('config-modal');
        this.form = document.getElementById('config-form');
        this.saveBtn = document.getElementById('save-config');
        this.minRatesContainer = document.getElementById('min-rates-container');
        this.currentConfig = null;

        this.bindEvents();
    }

    // 绑定事件
    bindEvents() {
        if (this.saveBtn) {
            this.saveBtn.addEventListener('click', () => this.saveConfig());
        }
    }

    // 加载配置
    async loadConfig() {
        console.log('[ConfigManager] 加载配置');

        try {
            const response = await api.getConfig();
            this.currentConfig = response.data;
            this.renderConfigForm(this.currentConfig);
            this.displayMinRates(this.currentConfig);
            console.log('[ConfigManager] 配置加载完成');
        } catch (error) {
            console.error('[ConfigManager] 加载配置失败:', error);
            alert('加载配置失败: ' + error.message);
        }
    }

    // 使用已有配置初始化（避免重复请求）
    initWithConfig(config) {
        console.log('[ConfigManager] 使用缓存配置初始化');
        this.currentConfig = config;
        this.renderConfigForm(config);
        this.displayMinRates(config);
        console.log('[ConfigManager] 配置初始化完成');
    }

    // 显示最小利率
    displayMinRates(config) {
        if (!this.minRatesContainer) return;
        if (!config.CURRENCIES) {
            this.minRatesContainer.innerHTML = '<div class="text-muted small">暂无配置</div>';
            return;
        }

        let html = '<div class="min-rates-list">';
        html += '<div class="small text-muted mb-2">最小贷出利率</div>';

        for (const [currency, currencyConfig] of Object.entries(config.CURRENCIES)) {
            if (currencyConfig.ENABLED) {
                const rate = currencyConfig.MIN_DAILY_LEND_RATE || 0;
                const ratePercent = (rate * 100).toFixed(4);
                html += `
                    <div class="min-rate-item">
                        <span class="currency-name">${currency.toUpperCase()}</span>
                        <span class="rate-value">${ratePercent}%</span>
                    </div>
                `;
            }
        }

        html += '</div>';
        this.minRatesContainer.innerHTML = html;
    }

    // 渲染配置表单
    renderConfigForm(config) {
        if (!config) return;

        let html = '';

        // 基本设置
        html += '<h6 class="mb-3">基本设置</h6>';
        html += this.createFormGroup('ORDER_LIMIT', '订单限制', config.ORDER_LIMIT, 'number');
        html += this.createFormGroup('MINUTES_RUN', '运行间隔（分钟）', config.MINUTES_RUN, 'number');
        html += '<hr>';

        // 多币种配置
        if (config.CURRENCIES) {
            html += '<h6 class="mb-3">币种配置</h6>';

            Object.entries(config.CURRENCIES).forEach(([currency, currencyConfig]) => {
                html += `<div class="card mb-3">`;
                html += `<div class="card-header bg-light">${currency}</div>`;
                html += `<div class="card-body">`;

                html += this.createFormGroup(`CURRENCIES.${currency}.ENABLED`, '启用', currencyConfig.ENABLED, 'checkbox');
                html += this.createFormGroup(`CURRENCIES.${currency}.MIN_LOAN`, '最小贷出额', currencyConfig.MIN_LOAN, 'number');
                html += this.createFormGroup(`CURRENCIES.${currency}.MAX_LOAN`, '最大贷出额', currencyConfig.MAX_LOAN, 'number');
                html += this.createFormGroup(`CURRENCIES.${currency}.MIN_DAILY_LEND_RATE`, '最小日利率', currencyConfig.MIN_DAILY_LEND_RATE, 'number', 0.000001);
                html += this.createFormGroup(`CURRENCIES.${currency}.SPREAD_LEND`, '分散订单数', currencyConfig.SPREAD_LEND, 'number');
                html += this.createFormGroup(`CURRENCIES.${currency}.GAP_BOTTOM`, '利率底部间隙', currencyConfig.GAP_BOTTOM, 'number');
                html += this.createFormGroup(`CURRENCIES.${currency}.GAP_TOP`, '利率顶部间隙', currencyConfig.GAP_TOP, 'number');
                html += this.createFormGroup(`CURRENCIES.${currency}.HIGH_HOLD_RATE`, '高额持有利率', currencyConfig.HIGH_HOLD_RATE, 'number', 0.000001);
                html += this.createFormGroup(`CURRENCIES.${currency}.HIGH_HOLD_AMOUNT`, '高额持有金额', currencyConfig.HIGH_HOLD_AMOUNT, 'number');
                html += this.createFormGroup(`CURRENCIES.${currency}.RESERVE_AMOUNT`, '保留金额', currencyConfig.RESERVE_AMOUNT, 'number');

                html += `</div></div>`;
            });

            html += '<hr>';
        }

        // 智能策略设置
        html += '<h6 class="mb-3">智能策略设置</h6>';
        html += this.createFormGroup('ENABLE_SMART_STRATEGY', '启用智能策略', config.ENABLE_SMART_STRATEGY, 'checkbox');
        html += this.createFormGroup('VOLATILITY_THRESHOLD', '波动率阈值', config.VOLATILITY_THRESHOLD, 'number', 0.000001);
        html += this.createFormGroup('MAX_RATE_MULTIPLIER', '最大利率倍数', config.MAX_RATE_MULTIPLIER, 'number', 0.01);
        html += this.createFormGroup('MIN_RATE_MULTIPLIER', '最小利率倍数', config.MIN_RATE_MULTIPLIER, 'number', 0.01);
        html += this.createFormGroup('RATE_RANGE_INCREASE_PERCENT', '利率范围增加百分比', config.RATE_RANGE_INCREASE_PERCENT, 'number', 0.01);
        html += '<hr>';

        // K线策略设置
        html += '<h6 class="mb-3">K线策略设置</h6>';
        html += this.createFormGroup('ENABLE_KLINE_STRATEGY', '启用K线策略', config.ENABLE_KLINE_STRATEGY, 'checkbox');
        html += this.createFormGroup('KLINE_TIME_FRAME', 'K线时间框架', config.KLINE_TIME_FRAME, 'text');
        html += this.createFormGroup('KLINE_PERIOD', 'K线周期数量', config.KLINE_PERIOD, 'number');
        html += '<hr>';

        // 系统设置
        html += '<h6 class="mb-3">系统设置</h6>';
        html += this.createFormGroup('TEST_MODE', '测试模式', config.TEST_MODE, 'checkbox');
        html += this.createFormGroup('LENDING_CHECK_MINUTES', '借贷检查间隔（分钟）', config.LENDING_CHECK_MINUTES, 'number');

        this.form.innerHTML = html;
    }

    // 创建表单组
    createFormGroup(name, label, value, type = 'text', step = null) {
        const id = `config-${name.replace(/\./g, '-')}`;

        if (type === 'checkbox') {
            return `
                <div class="mb-3 form-check">
                    <input type="checkbox" class="form-check-input" id="${id}" name="${name}" ${value ? 'checked' : ''}>
                    <label class="form-check-label" for="${id}">${label}</label>
                </div>
            `;
        }

        const stepAttr = step ? `step="${step}"` : '';

        return `
            <div class="mb-3">
                <label for="${id}" class="form-label">${label}</label>
                <input type="${type}" class="form-control" id="${id}" name="${name}" value="${value || ''}" ${stepAttr}>
            </div>
        `;
    }

    // 保存配置
    async saveConfig() {
        console.log('[ConfigManager] 保存配置');

        try {
            // 收集表单数据
            const formData = new FormData(this.form);
            const config = this.parseFormData(formData);

            // 发送到后端
            const response = await api.updateConfig(config);

            if (response.success) {
                alert('配置保存成功！');
                // 关闭模态框
                const modalInstance = bootstrap.Modal.getInstance(this.modal);
                if (modalInstance) {
                    modalInstance.hide();
                }
            } else {
                alert('配置保存失败: ' + (response.error || '未知错误'));
            }
        } catch (error) {
            console.error('[ConfigManager] 保存配置失败:', error);
            alert('保存配置失败: ' + error.message);
        }
    }

    // 解析表单数据
    parseFormData(formData) {
        const config = {};

        for (const [key, value] of formData.entries()) {
            const keys = key.split('.');
            let current = config;

            for (let i = 0; i < keys.length - 1; i++) {
                if (!current[keys[i]]) {
                    current[keys[i]] = {};
                }
                current = current[keys[i]];
            }

            const lastKey = keys[keys.length - 1];
            const input = this.form.querySelector(`[name="${key}"]`);

            if (input.type === 'checkbox') {
                current[lastKey] = input.checked;
            } else if (input.type === 'number') {
                current[lastKey] = parseFloat(value) || 0;
            } else {
                current[lastKey] = value;
            }
        }

        return config;
    }
}

// 创建全局实例
window.configManager = new ConfigManager();
