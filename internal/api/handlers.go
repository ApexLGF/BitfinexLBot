package api

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
	"github.com/ApexLGF/BitfinexLBot/internal/constants"
	"github.com/ApexLGF/BitfinexLBot/internal/currency"
)

// Handler API处理器
type Handler struct {
	config          *config.Config
	client          *bitfinex.Client
	currencyManager *currency.CurrencyManager // 替代 strategy
	isRunning       bool
	lastUpdate      time.Time
	nextRun         time.Time
	configPath      string // 添加配置文件路径
}

// NewHandler 创建新的API处理器
func NewHandler(config *config.Config, client *bitfinex.Client, currencyManager *currency.CurrencyManager, configPath string) *Handler {
	return &Handler{
		config:          config,
		client:          client,
		currencyManager: currencyManager,
		isRunning:       false,
		lastUpdate:      time.Now(),
		nextRun:         time.Now().Add(time.Duration(config.MinutesRun) * time.Minute),
		configPath:      configPath,
	}
}

// GetStatus 获取机器人状态
func (h *Handler) GetStatus(c *gin.Context) {
	// 支持查询参数指定币种
	currency := c.DefaultQuery("currency", "")

	if currency != "" {
		// 返回单个币种状态
		h.getSingleCurrencyStatus(c, currency)
		return
	}

	// 返回所有币种的汇总状态
	h.getAllCurrenciesStatus(c)
}

// getSingleCurrencyStatus 获取单个币种状态
func (h *Handler) getSingleCurrencyStatus(c *gin.Context, currency string) {
	var errors []string

	// 确保币种名称是大写
	upperCurrency := strings.ToUpper(currency)

	// 获取钱包余额
	availableFunds, err := h.client.GetFundingBalance(upperCurrency)
	if err != nil {
		errorMsg := fmt.Sprintf("获取钱包余额失败: %v", err)
		log.Printf("[API] GetStatus: %s", errorMsg)
		errors = append(errors, errorMsg)
		availableFunds = 0
	}

	// 获取活跃订单数量
	symbol := constants.FundingSymbolPrefix + upperCurrency
	offers, err := h.client.GetFundingOffers(symbol)
	activeOffers := 0
	if err != nil {
		errorMsg := fmt.Sprintf("获取活跃订单失败: %v", err)
		log.Printf("[API] GetStatus: %s", errorMsg)
		errors = append(errors, errorMsg)
	} else {
		activeOffers = len(offers)
	}

	// 获取总收益 - 使用月收益作为总收益
	totalEarnings := float64(0)
	if earnings := h.getCurrencyEarningsInternal(upperCurrency); earnings.Monthly > 0 {
		totalEarnings = earnings.Monthly
	}

	status := BotStatus{
		IsRunning:      h.isRunning,
		LastUpdate:     h.lastUpdate,
		NextRun:        h.nextRun,
		TotalEarnings:  totalEarnings,
		ActiveOffers:   activeOffers,
		AvailableFunds: availableFunds,
		Currency:       currency,
		Errors:         errors,
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}

// getAllCurrenciesStatus 获取所有币种状态
func (h *Handler) getAllCurrenciesStatus(c *gin.Context) {
	currencies := h.currencyManager.GetEnabledCurrencies()
	statuses := make(map[string]BotStatus)

	var totalEarnings, totalAvailableFunds float64
	var totalActiveOffers int

	for _, currency := range currencies {
		availableFunds, _ := h.client.GetFundingBalance(currency)
		symbol := constants.FundingSymbolPrefix + strings.ToUpper(currency)
		offers, _ := h.client.GetFundingOffers(symbol)
		earnings := h.getCurrencyEarningsInternal(currency)

		statuses[currency] = BotStatus{
			IsRunning:      h.isRunning,
			LastUpdate:     h.lastUpdate,
			NextRun:        h.nextRun,
			TotalEarnings:  earnings.Monthly,
			ActiveOffers:   len(offers),
			AvailableFunds: availableFunds,
			Currency:       currency,
		}

		totalEarnings += earnings.Monthly
		totalActiveOffers += len(offers)
		totalAvailableFunds += availableFunds
	}

	response := map[string]interface{}{
		"currencies": statuses,
		"summary": map[string]interface{}{
			"total_earnings":        totalEarnings,
			"total_active_offers":   totalActiveOffers,
			"total_available_funds": totalAvailableFunds,
			"enabled_currencies":    currencies,
		},
		"is_running":  h.isRunning,
		"last_update": h.lastUpdate,
		"next_run":    h.nextRun,
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetEarnings 获取收益数据
func (h *Handler) GetEarnings(c *gin.Context) {
	// 支持查询参数指定币种
	currency := c.DefaultQuery("currency", "")

	// 如果没有指定币种，返回所有币种的收益
	if currency == "" {
		h.getAllCurrenciesEarnings(c)
		return
	}

	// 返回单个币种的收益
	h.getSingleCurrencyEarnings(c, currency)
}

// getSingleCurrencyEarnings 获取单个币种的收益数据
func (h *Handler) getSingleCurrencyEarnings(c *gin.Context, currency string) {
	log.Printf("[API] GetEarnings 开始获取 %s 的30天收益数据", currency)

	earnings := h.getCurrencyEarningsInternal(strings.ToUpper(currency))

	log.Printf("[API] 收益统计 - 日: %.4f, 周: %.4f, 月: %.4f",
		earnings.Daily, earnings.Weekly, earnings.Monthly)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    earnings,
	})
}

// getAllCurrenciesEarnings 获取所有币种的收益数据
func (h *Handler) getAllCurrenciesEarnings(c *gin.Context) {
	log.Printf("[API] GetEarnings 开始获取所有币种的30天收益数据")

	currencies := h.currencyManager.GetEnabledCurrencies()
	allEarnings := make(map[string]EarningsData)

	var totalDaily, totalWeekly, totalMonthly float64

	for _, currency := range currencies {
		earnings := h.getCurrencyEarningsInternal(strings.ToUpper(currency))
		allEarnings[strings.ToUpper(currency)] = earnings

		totalDaily += earnings.Daily
		totalWeekly += earnings.Weekly
		totalMonthly += earnings.Monthly
	}

	response := map[string]interface{}{
		"currencies": allEarnings,
		"summary": map[string]interface{}{
			"daily":   totalDaily,
			"weekly":  totalWeekly,
			"monthly": totalMonthly,
		},
	}

	log.Printf("[API] 总收益统计 - 日: %.4f, 周: %.4f, 月: %.4f",
		totalDaily, totalWeekly, totalMonthly)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// processEarningsData 处理收益数据
func (h *Handler) processEarningsData(ledgers []*bitfinex.LedgerEntry, currency string) EarningsData {
	// 按日期分组收益
	dailyEarnings := make(map[string]float64)
	var totalEarnings float64

	now := time.Now()
	oneDayAgo := now.AddDate(0, 0, -1)
	oneWeekAgo := now.AddDate(0, 0, -7)

	var dailyTotal, weeklyTotal, monthlyTotal float64

	for _, ledger := range ledgers {
		// 只统计放贷收益相关的记录
		// 1. 金额必须为正数（收益）
		// 2. 描述包含资金放贷相关关键词
		if ledger.Amount <= 0 {
			continue
		}

		// 检查是否为资金放贷收益
		desc := strings.ToLower(ledger.Description)

		// 首先排除明确不是收益的交易类型
		isExcluded := strings.Contains(desc, "transfer") ||
			strings.Contains(desc, "deposit") ||
			strings.Contains(desc, "withdrawal") ||
			strings.Contains(desc, "exchange") ||
			strings.Contains(desc, "trading") ||
			strings.Contains(desc, "fee") ||
			strings.Contains(desc, "conversion") ||
			strings.Contains(desc, "settle") ||
			strings.Contains(desc, "liquidation") ||
			strings.Contains(desc, "referral") ||
			strings.Contains(desc, "affiliate")

		if isExcluded {
			continue
		}

		// 然后检查是否为资金放贷收益
		isFundingEarning := strings.Contains(desc, "margin funding") ||
			strings.Contains(desc, "funding payment") ||
			strings.Contains(desc, "margin funding payment") ||
			strings.Contains(desc, "lending") ||
			strings.Contains(desc, "margin lending") ||
			strings.Contains(desc, "funding credit") ||
			strings.Contains(desc, "margin interest") ||
			// 可能的其他放贷收益描述
			(strings.Contains(desc, "funding") && strings.Contains(desc, "earn"))

		if !isFundingEarning {
			continue
		}

		entryTime := time.Unix(ledger.Timestamp/1000, 0)
		dateKey := entryTime.Format("2006-01-02")

		// 累计到对应日期
		dailyEarnings[dateKey] += ledger.Amount
		totalEarnings += ledger.Amount
		monthlyTotal += ledger.Amount

		// 计算最近1天和7天的收益
		if entryTime.After(oneDayAgo) {
			dailyTotal += ledger.Amount
		}
		if entryTime.After(oneWeekAgo) {
			weeklyTotal += ledger.Amount
		}
	}


	// 构建历史记录数组（按时间排序）
	var history []EarningsHistory

	// 生成最近30天的记录（即使某天没有收益也显示0）
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateKey := date.Format("2006-01-02")
		amount := dailyEarnings[dateKey] // 如果没有记录，默认为0

		history = append(history, EarningsHistory{
			Date:     dateKey,
			Amount:   amount,
			Rate:     0, // 暂时设为0，后续可以计算平均利率
			Currency: currency,
		})
	}

	return EarningsData{
		Daily:    dailyTotal,
		Weekly:   weeklyTotal,
		Monthly:  monthlyTotal,
		Currency: currency,
		History:  history,
	}
}

// GetDailyEarnings 获取日收益
func (h *Handler) GetDailyEarnings(c *gin.Context) {
	// 获取完整收益数据
	earnings := h.getEarningsDataInternal()
	
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]interface{}{"daily_earnings": earnings.Daily},
	})
}

// GetWeeklyEarnings 获取周收益
func (h *Handler) GetWeeklyEarnings(c *gin.Context) {
	// 获取完整收益数据
	earnings := h.getEarningsDataInternal()
	
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]interface{}{"weekly_earnings": earnings.Weekly},
	})
}

// getEarningsDataInternal 内部获取收益数据（避免重复代码）
func (h *Handler) getEarningsDataInternal() EarningsData {
	// 获取第一个启用的币种
	currencies := h.currencyManager.GetEnabledCurrencies()
	if len(currencies) == 0 {
		return EarningsData{
			Daily:   0,
			Weekly:  0,
			Monthly: 0,
			History: []EarningsHistory{},
		}
	}

	// 返回第一个币种的收益数据
	return h.getCurrencyEarningsInternal(currencies[0])
}

// getCurrencyEarningsInternal 获取特定币种的收益数据
func (h *Handler) getCurrencyEarningsInternal(currency string) EarningsData {
	now := time.Now()
	startTime := now.AddDate(0, 0, -30)
	startMs := startTime.UnixNano() / 1000000
	endMs := now.UnixNano() / 1000000

	// 确保币种是大写
	upperCurrency := strings.ToUpper(currency)

	ledgers, err := h.client.GetFundingLedgers(upperCurrency, startMs, endMs, 500)
	if err != nil {
		log.Printf("[API] 获取 %s 账本记录失败: %v", upperCurrency, err)
		return EarningsData{
			Daily:    0,
			Weekly:   0,
			Monthly:  0,
			Currency: upperCurrency,
			History:  []EarningsHistory{},
		}
	}

	earnings := h.processEarningsData(ledgers, upperCurrency)
	return earnings
}

// GetOffers 获取放贷订单
func (h *Handler) GetOffers(c *gin.Context) {
	// 支持查询参数指定币种
	currency := c.DefaultQuery("currency", "")

	// 如果没有指定币种，返回所有币种的订单
	if currency == "" {
		h.getAllCurrenciesOffers(c)
		return
	}

	// 返回单个币种的订单
	h.getSingleCurrencyOffers(c, currency)
}

// getSingleCurrencyOffers 获取单个币种的放贷订单
func (h *Handler) getSingleCurrencyOffers(c *gin.Context, currency string) {
	symbol := constants.FundingSymbolPrefix + strings.ToUpper(currency)
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	log.Printf("[API] GetOffers 开始调用, symbol: %s, client: %s, user-agent: %s", symbol, clientIP, userAgent)

	offers, err := h.client.GetFundingOffers(symbol)
	if err != nil {
		log.Printf("[API] GetOffers 失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "获取订单失败: " + err.Error(),
		})
		return
	}

	log.Printf("[API] GetOffers 成功获取 %d 个放贷订单", len(offers))

	// 初始化为空数组而不是 nil
	offerData := make([]OfferData, 0)
	for _, offer := range offers {
		offerData = append(offerData, OfferData{
			ID:       offer.ID,
			Amount:   offer.Amount,
			Rate:     offer.Rate,
			Period:   offer.Period,
			Status:   "ACTIVE",
			Created:  time.Now(),
			Currency: strings.ToUpper(currency),
		})
	}

	log.Printf("[API] GetOffers 响应: 返回 %d 个订单", len(offerData))

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    offerData,
	})
}

// getAllCurrenciesOffers 获取所有币种的放贷订单
func (h *Handler) getAllCurrenciesOffers(c *gin.Context) {
	currencies := h.currencyManager.GetEnabledCurrencies()
	allOffers := make(map[string][]OfferData)

	for _, currency := range currencies {
		symbol := constants.FundingSymbolPrefix + strings.ToUpper(currency)
		offers, err := h.client.GetFundingOffers(symbol)
		if err != nil {
			log.Printf("[API] 获取 %s 订单失败: %v", currency, err)
			continue
		}

		offerData := make([]OfferData, 0)
		for _, offer := range offers {
			offerData = append(offerData, OfferData{
				ID:       offer.ID,
				Amount:   offer.Amount,
				Rate:     offer.Rate,
				Period:   offer.Period,
				Status:   "ACTIVE",
				Created:  time.Now(),
				Currency: strings.ToUpper(currency),
			})
		}

		allOffers[strings.ToUpper(currency)] = offerData
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    allOffers,
	})
}

// GetActiveOffers 获取活跃订单
func (h *Handler) GetActiveOffers(c *gin.Context) {
	log.Printf("[API] GetActiveOffers 调用 (委托给 GetOffers)")
	h.GetOffers(c) // 简化实现，实际应筛选活跃订单
}

// GetConfig 获取配置
func (h *Handler) GetConfig(c *gin.Context) {
	// 返回安全的配置信息（隐藏敏感信息）
	safeConfig := map[string]interface{}{
		// 基本设置
		"ORDER_LIMIT":        h.config.OrderLimit,
		"MINUTES_RUN":        h.config.MinutesRun,

		// 多币种配置
		"CURRENCIES":         h.config.Currencies,

		// 智能策略设置（全局）
		"ENABLE_SMART_STRATEGY":      h.config.EnableSmartStrategy,
		"VOLATILITY_THRESHOLD":       h.config.VolatilityThreshold,
		"MAX_RATE_MULTIPLIER":        h.config.MaxRateMultiplier,
		"MIN_RATE_MULTIPLIER":        h.config.MinRateMultiplier,
		"RATE_RANGE_INCREASE_PERCENT": h.config.RateRangeIncreasePercent,

		// K线策略设置（全局）
		"ENABLE_KLINE_STRATEGY": h.config.EnableKlineStrategy,
		"KLINE_TIME_FRAME":      h.config.KlineTimeFrame,
		"KLINE_PERIOD":          h.config.KlinePeriod,
		"KLINE_SPREAD_PERCENT":  h.config.KlineSpreadPercent,
		"KLINE_SMOOTH_METHOD":   h.config.KlineSmoothMethod,

		// 系统设置
		"TEST_MODE":          h.config.TestMode,
		"LENDING_CHECK_MINUTES": h.config.LendingCheckMinutes,

		// API 服务配置
		"API_ENABLED":     h.config.APIEnabled,
		"API_PORT":        h.config.APIPort,
		"API_HOST":        h.config.APIHost,
		"API_CORS_ORIGINS": h.config.APICorsOrigins,
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    safeConfig,
	})
}

// UpdateConfig 更新配置
func (h *Handler) UpdateConfig(c *gin.Context) {
	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "无效的请求格式: " + err.Error(),
		})
		return
	}

	log.Printf("[API] 收到配置更新请求: %+v", req.Config)

	// 保存配置到文件
	if err := config.SaveConfig(h.configPath, req.Config); err != nil {
		log.Printf("[API] 保存配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "保存配置文件失败: " + err.Error(),
		})
		return
	}

	// 重新加载配置
	newConfig, err := config.ReloadConfig(h.configPath)
	if err != nil {
		log.Printf("[API] 重新加载配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "重新加载配置失败: " + err.Error(),
		})
		return
	}

	// 更新处理器中的配置引用
	h.config = newConfig
	
	// 注意：这里我们只更新了Handler中的配置，实际的策略和客户端可能需要重新创建
	// 为了完全生效，可能需要重启应用或重新初始化组件
	log.Printf("[API] 配置更新成功，新配置已加载")

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "配置更新成功，部分设置可能需要重启应用才能完全生效",
	})
}

// Control 机器人控制
func (h *Handler) Control(c *gin.Context) {
	var req ControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "无效的请求格式: " + err.Error(),
		})
		return
	}

	switch req.Action {
	case "start":
		h.isRunning = true
		h.lastUpdate = time.Now()
		h.nextRun = time.Now().Add(time.Duration(h.config.MinutesRun) * time.Minute)
		// TODO: 实际启动机器人逻辑
		c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Message: "机器人已启动",
		})
	case "stop":
		h.isRunning = false
		// TODO: 实际停止机器人逻辑
		c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Message: "机器人已停止",
		})
	case "restart":
		// 实际重启机器人逻辑
		if err := h.restartBot(); err != nil {
			c.JSON(http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "重启失败: " + err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Message: "机器人重启成功，配置已重新加载",
		})
	default:
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "无效的操作: " + req.Action,
		})
	}
}

// GetLogs 获取日志
func (h *Handler) GetLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000 // 限制最大数量
	}

	// 读取日志文件
	logs, err := h.readLogFile("/app/logs/bitfinex-bot.err.log", limit)
	if err != nil {
		log.Printf("[API] 读取日志文件失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "读取日志失败: " + err.Error(),
		})
		return
	}

	response := LogResponse{
		Logs:  logs,
		Total: len(logs),
		Page:  1,
		Limit: limit,
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// readLogFile 读取日志文件内容
func (h *Handler) readLogFile(logPath string, limit int) ([]LogEntry, error) {
	// 检查文件是否存在
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		log.Printf("[API] 日志文件不存在: %s", logPath)
		return []LogEntry{}, nil
	}

	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	
	// 读取所有行
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 保持原始顺序，最新的在最后

	// 限制数量 - 获取最后的N行（最新的日志）
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	// 转换为LogEntry
	var logs []LogEntry
	for _, line := range lines {
		entry := h.parseLogLine(line)
		logs = append(logs, entry)
	}

	return logs, nil
}

// parseLogLine 解析日志行
func (h *Handler) parseLogLine(line string) LogEntry {
	// 返回原始日志内容，不进行任何处理
	entry := LogEntry{
		Level:     "INFO",
		Message:   line, // 显示完整的原始日志行
		Timestamp: time.Now(), // 使用当前时间作为接收时间
		Component: "",
	}

	return entry
}

// GetFundingCredits 获取已贷出订单
func (h *Handler) GetFundingCredits(c *gin.Context) {
	// 支持查询参数指定币种
	currency := c.DefaultQuery("currency", "")

	// 如果没有指定币种，返回所有币种的已贷出订单
	if currency == "" {
		h.getAllCurrenciesCredits(c)
		return
	}

	// 返回单个币种的已贷出订单
	h.getSingleCurrencyCredits(c, currency)
}

// getSingleCurrencyCredits 获取单个币种的已贷出订单
func (h *Handler) getSingleCurrencyCredits(c *gin.Context, currency string) {
	symbol := constants.FundingSymbolPrefix + strings.ToUpper(currency)
	log.Printf("[API] GetFundingCredits 开始调用, symbol: %s", symbol)

	credits, err := h.client.GetFundingCredits(symbol)
	if err != nil {
		log.Printf("[API] GetFundingCredits 失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "获取已贷出订单失败: " + err.Error(),
		})
		return
	}

	log.Printf("[API] GetFundingCredits 成功获取 %d 个已贷出订单", len(credits))

	creditData := make([]FundingCreditData, 0)
	var totalAmount float64 = 0
	var totalRate float64 = 0
	var validCount int = 0

	if credits == nil {
		credits = []*bitfinex.FundingCredit{}
	}

	for _, credit := range credits {
		if credit == nil {
			continue
		}
		creditData = append(creditData, FundingCreditData{
			ID:       credit.ID,
			Symbol:   credit.Symbol,
			Amount:   credit.Amount,
			Rate:     credit.Rate,
			Period:   credit.Period,
			Status:   credit.Status,
			Created:  time.Unix(credit.MTSCreated/1000, (credit.MTSCreated%1000)*1000000),
			Opened:   time.Unix(credit.MTSOpened/1000, (credit.MTSOpened%1000)*1000000),
			Currency: strings.ToUpper(currency),
		})

		totalAmount += credit.Amount
		if credit.Rate > 0 {
			totalRate += credit.Rate
			validCount++
		}
	}

	avgRate := float64(0)
	if validCount > 0 {
		avgRate = totalRate / float64(validCount)
	}

	response := FundingCreditsResponse{
		Credits:     creditData,
		TotalCount:  len(creditData),
		TotalAmount: totalAmount,
		AvgRate:     avgRate,
	}

	log.Printf("[API] GetFundingCredits 响应: count=%d, total=%.2f, avg_rate=%.6f",
		response.TotalCount, response.TotalAmount, response.AvgRate)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// getAllCurrenciesCredits 获取所有币种的已贷出订单
func (h *Handler) getAllCurrenciesCredits(c *gin.Context) {
	currencies := h.currencyManager.GetEnabledCurrencies()
	allCredits := make(map[string]FundingCreditsResponse)

	for _, currency := range currencies {
		symbol := constants.FundingSymbolPrefix + strings.ToUpper(currency)
		credits, err := h.client.GetFundingCredits(symbol)
		if err != nil {
			log.Printf("[API] 获取 %s 已贷出订单失败: %v", currency, err)
			continue
		}

		creditData := make([]FundingCreditData, 0)
		var totalAmount float64 = 0
		var totalRate float64 = 0
		var validCount int = 0

		for _, credit := range credits {
			if credit == nil {
				continue
			}
			creditData = append(creditData, FundingCreditData{
				ID:       credit.ID,
				Symbol:   credit.Symbol,
				Amount:   credit.Amount,
				Rate:     credit.Rate,
				Period:   credit.Period,
				Status:   credit.Status,
				Created:  time.Unix(credit.MTSCreated/1000, (credit.MTSCreated%1000)*1000000),
				Opened:   time.Unix(credit.MTSOpened/1000, (credit.MTSOpened%1000)*1000000),
				Currency: strings.ToUpper(currency),
			})

			totalAmount += credit.Amount
			if credit.Rate > 0 {
				totalRate += credit.Rate
				validCount++
			}
		}

		avgRate := float64(0)
		if validCount > 0 {
			avgRate = totalRate / float64(validCount)
		}

		allCredits[strings.ToUpper(currency)] = FundingCreditsResponse{
			Credits:     creditData,
			TotalCount:  len(creditData),
			TotalAmount: totalAmount,
			AvgRate:     avgRate,
		}
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    allCredits,
	})
}

// GetWallets 获取钱包信息
func (h *Handler) GetWallets(c *gin.Context) {
	wallets, err := h.client.GetWallets()
	if err != nil {
		log.Printf("[API] GetWallets 失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "获取钱包信息失败: " + err.Error(),
		})
		return
	}
	
	// 计算指定货币的总资金
	var totalBalance float64
	var availableBalance float64
	var walletData []map[string]interface{}
	
	for _, wallet := range wallets {
		if wallet.Currency == h.config.Currency {
			totalBalance += wallet.Balance
			availableBalance += wallet.Available
		}
		
		walletData = append(walletData, map[string]interface{}{
			"currency":  wallet.Currency,
			"type":      wallet.Type,
			"balance":   wallet.Balance,
			"available": wallet.Available,
		})
	}
	
	response := map[string]interface{}{
		"wallets":           walletData,
		"total_balance":     totalBalance,
		"available_balance": availableBalance,
		"currency":          h.config.Currency,
	}
	
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetSystemInfo 获取系统信息
func (h *Handler) GetSystemInfo(c *gin.Context) {
	info := map[string]interface{}{
		"version":     "2.0.0",
		"go_version":  "1.23",
		"start_time":  time.Now(), // TODO: 记录实际启动时间
		"api_version": "v1",
		"features": []string{
			"REST API",
			"Web Interface",
			"Real-time Status",
			"Configuration Management",
		},
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    info,
	})
}

// SetRunning 设置运行状态
func (h *Handler) SetRunning(running bool) {
	h.isRunning = running
	if running {
		h.lastUpdate = time.Now()
		h.nextRun = time.Now().Add(time.Duration(h.config.MinutesRun) * time.Minute)
	}
}

// UpdateNextRun 更新下次运行时间
func (h *Handler) UpdateNextRun() {
	h.lastUpdate = time.Now()
	h.nextRun = time.Now().Add(time.Duration(h.config.MinutesRun) * time.Minute)
}

// restartBot 重启机器人 - 重新加载配置并重新初始化所有组件
func (h *Handler) restartBot() error {
	log.Printf("[API] 开始重启机器人...")
	
	// 1. 重新加载配置文件
	log.Printf("[API] 重新加载配置文件: %s", h.configPath)
	newConfig, err := config.LoadConfig(h.configPath)
	if err != nil {
		log.Printf("[API] 重新加载配置失败: %v", err)
		return fmt.Errorf("重新加载配置失败: %w", err)
	}
	
	// 2. 检查关键配置是否发生变化
	configChanged := h.hasConfigChanged(h.config, newConfig)
	if configChanged {
		log.Printf("[API] 检测到配置变化，重新初始化客户端和策略")
	}
	
	// 3. 更新配置引用
	oldConfig := h.config
	h.config = newConfig
	
	// 4. 如果API密钥或其他关键配置变化，重新初始化客户端
	if configChanged {
		log.Printf("[API] 重新初始化Bitfinex客户端")
		h.client = bitfinex.NewClient(newConfig.BitfinexApiKey, newConfig.BitfinexSecretKey)

		// 5. 重新创建币种管理器
		log.Printf("[API] 重新创建币种管理器")
		if err := h.currencyManager.ReloadConfig(newConfig); err != nil {
			log.Printf("[API] 重新加载币种管理器失败: %v", err)
			h.config = oldConfig
			return fmt.Errorf("重新加载币种管理器失败: %w", err)
		}
	} else {
		log.Printf("[API] 配置未发生关键变化，重新加载币种管理器配置")
		// 如果只是普通配置变化，重新加载币种管理器
		if err := h.currencyManager.ReloadConfig(newConfig); err != nil {
			log.Printf("[API] 重新加载币种管理器配置失败: %v", err)
			// 配置更新失败时回滚
			h.config = oldConfig
			return fmt.Errorf("重新加载币种管理器配置失败: %w", err)
		}
	}

	// 6. 执行一次策略以验证新配置
	log.Printf("[API] 执行所有币种策略验证新配置")
	if err := h.currencyManager.ExecuteAll(); err != nil {
		log.Printf("[API] 策略执行失败: %v", err)
		// 执行失败时回滚配置
		h.config = oldConfig
		return fmt.Errorf("策略执行失败: %w", err)
	}

	// 7. 更新运行状态
	h.isRunning = true
	h.lastUpdate = time.Now()
	h.nextRun = time.Now().Add(time.Duration(newConfig.MinutesRun) * time.Minute)
	
	log.Printf("[API] 机器人重启成功")
	return nil
}

// hasConfigChanged 检查配置是否发生关键变化
func (h *Handler) hasConfigChanged(oldConfig, newConfig *config.Config) bool {
	// 检查需要重新初始化客户端的关键配置
	if oldConfig.BitfinexApiKey != newConfig.BitfinexApiKey {
		return true
	}
	if oldConfig.BitfinexSecretKey != newConfig.BitfinexSecretKey {
		return true
	}
	if oldConfig.Currency != newConfig.Currency {
		return true
	}

	// 其他配置变化不需要重新初始化客户端，但需要更新策略
	return false
}

// GetFRRRates 获取所有币种的当前 FRR 利率
func (h *Handler) GetFRRRates(c *gin.Context) {
	log.Printf("[API] GetFRRRates 开始获取所有币种的 FRR 利率")

	currencies := h.currencyManager.GetEnabledCurrencies()
	frrRates := make(map[string]float64)

	for _, currency := range currencies {
		symbol := constants.FundingSymbolPrefix + strings.ToUpper(currency)
		rate, err := h.client.GetCurrentFundingRate(symbol)
		if err != nil {
			log.Printf("[API] 获取 %s FRR 利率失败: %v", currency, err)
			frrRates[strings.ToUpper(currency)] = 0
			continue
		}
		frrRates[strings.ToUpper(currency)] = rate
		log.Printf("[API] %s FRR 利率: %.6f", currency, rate)
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    frrRates,
	})
}

// GetYearlyEarnings 获取所有币种的年度累计收益（最近365天）
func (h *Handler) GetYearlyEarnings(c *gin.Context) {
	log.Printf("[API] GetYearlyEarnings 开始获取所有币种的年度收益")

	currencies := h.currencyManager.GetEnabledCurrencies()
	yearlyEarnings := make(map[string]float64)
	var totalYearly float64

	now := time.Now()
	startTime := now.AddDate(-1, 0, 0) // 365天前
	startMs := startTime.UnixNano() / 1000000
	endMs := now.UnixNano() / 1000000

	for _, currency := range currencies {
		upperCurrency := strings.ToUpper(currency)

		ledgers, err := h.client.GetFundingLedgers(upperCurrency, startMs, endMs, 1000)
		if err != nil {
			log.Printf("[API] 获取 %s 年度账本记录失败: %v", upperCurrency, err)
			yearlyEarnings[upperCurrency] = 0
			continue
		}

		// 计算年度总收益
		var yearlyTotal float64
		for _, ledger := range ledgers {
			// 只统计放贷收益相关的记录
			if ledger.Amount <= 0 {
				continue
			}

			desc := strings.ToLower(ledger.Description)

			// 排除非收益交易
			isExcluded := strings.Contains(desc, "transfer") ||
				strings.Contains(desc, "deposit") ||
				strings.Contains(desc, "withdrawal") ||
				strings.Contains(desc, "exchange") ||
				strings.Contains(desc, "trading") ||
				strings.Contains(desc, "fee") ||
				strings.Contains(desc, "conversion") ||
				strings.Contains(desc, "settle") ||
				strings.Contains(desc, "liquidation") ||
				strings.Contains(desc, "referral") ||
				strings.Contains(desc, "affiliate")

			if isExcluded {
				continue
			}

			// 检查是否为资金放贷收益
			isFundingEarning := strings.Contains(desc, "margin funding") ||
				strings.Contains(desc, "funding payment") ||
				strings.Contains(desc, "margin funding payment") ||
				strings.Contains(desc, "lending") ||
				strings.Contains(desc, "margin lending") ||
				strings.Contains(desc, "funding credit") ||
				strings.Contains(desc, "margin interest") ||
				(strings.Contains(desc, "funding") && strings.Contains(desc, "earn"))

			if isFundingEarning {
				yearlyTotal += ledger.Amount
			}
		}

		yearlyEarnings[upperCurrency] = yearlyTotal
		totalYearly += yearlyTotal
		log.Printf("[API] %s 年度收益: %.4f", upperCurrency, yearlyTotal)
	}

	response := map[string]interface{}{
		"currencies": yearlyEarnings,
		"total":      totalYearly,
	}

	log.Printf("[API] 总年度收益: %.4f", totalYearly)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}