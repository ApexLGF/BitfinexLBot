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
	"github.com/ApexLGF/BitfinexLBot/internal/strategy"
)

// Handler API处理器
type Handler struct {
	config       *config.Config
	client       *bitfinex.Client
	strategy     *strategy.LendingBot
	isRunning    bool
	lastUpdate   time.Time
	nextRun      time.Time
	configPath   string // 添加配置文件路径
}

// NewHandler 创建新的API处理器
func NewHandler(config *config.Config, client *bitfinex.Client, strategy *strategy.LendingBot, configPath string) *Handler {
	return &Handler{
		config:     config,
		client:     client,
		strategy:   strategy,
		isRunning:  false,
		lastUpdate: time.Now(),
		nextRun:    time.Now().Add(time.Duration(config.MinutesRun) * time.Minute),
		configPath: configPath,
	}
}

// GetStatus 获取机器人状态
func (h *Handler) GetStatus(c *gin.Context) {
	var errors []string

	// 获取钱包余额
	availableFunds, err := h.client.GetFundingBalance(h.config.Currency)
	if err != nil {
		errorMsg := fmt.Sprintf("获取钱包余额失败: %v", err)
		log.Printf("[API] GetStatus: %s", errorMsg)
		errors = append(errors, errorMsg)
		availableFunds = 0
	}

	// 获取活跃订单数量
	offers, err := h.client.GetFundingOffers(h.config.GetFundingSymbol())
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
	if earnings := h.getEarningsDataInternal(); earnings.Monthly > 0 {
		totalEarnings = earnings.Monthly
	}

	status := BotStatus{
		IsRunning:      h.isRunning,
		LastUpdate:     h.lastUpdate,
		NextRun:        h.nextRun,
		TotalEarnings:  totalEarnings,
		ActiveOffers:   activeOffers,
		AvailableFunds: availableFunds,
		Currency:       h.config.Currency,
		Errors:         errors,
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}

// GetEarnings 获取收益数据
func (h *Handler) GetEarnings(c *gin.Context) {
	log.Printf("[API] GetEarnings 开始获取30天收益数据")
	
	// 计算30天时间范围
	now := time.Now()
	startTime := now.AddDate(0, 0, -30) // 30天前
	
	// 转换为毫秒时间戳
	startMs := startTime.UnixNano() / 1000000
	endMs := now.UnixNano() / 1000000
	
	// 获取账本记录
	ledgers, err := h.client.GetFundingLedgers(h.config.Currency, startMs, endMs, 500)
	if err != nil {
		log.Printf("[API] 获取账本记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "获取收益数据失败: " + err.Error(),
		})
		return
	}
	
	log.Printf("[API] 成功获取 %d 条账本记录", len(ledgers))
	
	// 处理数据并计算收益统计
	earnings := h.processEarningsData(ledgers)
	
	log.Printf("[API] 收益统计 - 日: %.4f, 周: %.4f, 月: %.4f, 历史记录: %d条", 
		earnings.Daily, earnings.Weekly, earnings.Monthly, len(earnings.History))

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    earnings,
	})
}

// processEarningsData 处理收益数据
func (h *Handler) processEarningsData(ledgers []*bitfinex.LedgerEntry) EarningsData {
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
		isFundingEarning := strings.Contains(desc, "margin funding") || 
			strings.Contains(desc, "funding payment") ||
			strings.Contains(desc, "lending") ||
			strings.Contains(desc, "margin lending")
		
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
			Currency: h.config.Currency,
		})
	}
	
	return EarningsData{
		Daily:   dailyTotal,
		Weekly:  weeklyTotal,
		Monthly: monthlyTotal,
		History: history,
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
	now := time.Now()
	startTime := now.AddDate(0, 0, -30)
	startMs := startTime.UnixNano() / 1000000
	endMs := now.UnixNano() / 1000000
	
	ledgers, err := h.client.GetFundingLedgers(h.config.Currency, startMs, endMs, 500)
	if err != nil {
		log.Printf("[API] 获取账本记录失败: %v", err)
		return EarningsData{
			Daily:   0,
			Weekly:  0,
			Monthly: 0,
			History: []EarningsHistory{},
		}
	}
	
	return h.processEarningsData(ledgers)
}

// GetOffers 获取放贷订单
func (h *Handler) GetOffers(c *gin.Context) {
	symbol := h.config.GetFundingSymbol()
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

	var offerData []OfferData
	for _, offer := range offers {
		offerData = append(offerData, OfferData{
			ID:       offer.ID,
			Amount:   offer.Amount,
			Rate:     offer.Rate,
			Period:   offer.Period,
			Status:   "ACTIVE", // FundingOffer 结构体没有Status字段，默认为ACTIVE
			Created:  time.Now(), // FundingOffer 结构体没有创建时间，使用当前时间
			Currency: h.config.Currency,
		})
	}

	log.Printf("[API] GetOffers 响应: 返回 %d 个订单", len(offerData))
	
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    offerData,
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
		"CURRENCY":           h.config.Currency,
		"ORDER_LIMIT":        h.config.OrderLimit,
		"MINUTES_RUN":        h.config.MinutesRun,
		
		// 贷出限制
		"MIN_LOAN":           h.config.MinLoan,
		"MAX_LOAN":           h.config.MaxLoan,
		
		// 利率策略
		"MIN_DAILY_LEND_RATE":              h.config.MinDailyLendRate,
		"SPREAD_LEND":                      h.config.SpreadLend,
		"GAP_BOTTOM":                       h.config.GapBottom,
		"GAP_TOP":                          h.config.GapTop,
		"THIRTY_DAY_LEND_RATE_THRESHOLD":   h.config.ThirtyDayLendRateThreshold,
		"ONE_TWENTY_DAY_LEND_RATE_THRESHOLD": h.config.OneTwentyDayLendRateThreshold,
		"RATE_BONUS":                       h.config.RateBonus,
		
		// 高额持有策略
		"HIGH_HOLD_RATE":     h.config.HighHoldRate,
		"HIGH_HOLD_AMOUNT":   h.config.HighHoldAmount,
		"HIGH_HOLD_ORDERS":   h.config.HighHoldOrders,
		
		// 通知设置
		"NOTIFY_RATE_THRESHOLD": h.config.NotifyRateThreshold,
		"RESERVE_AMOUNT":        h.config.ReserveAmount,
		
		// 智能策略设置
		"ENABLE_SMART_STRATEGY":      h.config.EnableSmartStrategy,
		"VOLATILITY_THRESHOLD":       h.config.VolatilityThreshold,
		"MAX_RATE_MULTIPLIER":        h.config.MaxRateMultiplier,
		"MIN_RATE_MULTIPLIER":        h.config.MinRateMultiplier,
		"RATE_RANGE_INCREASE_PERCENT": h.config.RateRangeIncreasePercent,
		
		// 系统设置
		"TEST_MODE":          h.config.TestMode,
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
		h.isRunning = true
		h.lastUpdate = time.Now()
		h.nextRun = time.Now().Add(time.Duration(h.config.MinutesRun) * time.Minute)
		// TODO: 实际重启机器人逻辑（执行一次策略）
		if err := h.strategy.Execute(); err != nil {
			c.JSON(http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "重启执行失败: " + err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Message: "机器人已重启",
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
	symbol := h.config.GetFundingSymbol()
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

	var creditData []FundingCreditData
	var totalAmount float64 = 0
	var totalRate float64 = 0
	var validCount int = 0

	// 处理空结果的情况
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
			Currency: h.config.Currency,
		})

		totalAmount += credit.Amount
		if credit.Rate > 0 {
			totalRate += credit.Rate
			validCount++
		}
	}

	// 计算平均利率
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