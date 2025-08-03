package database

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ApexLGF/BitfinexLBot/internal/config"
	"github.com/ApexLGF/BitfinexLBot/internal/strategy"
)

// ConfigUpdater 配置更新接口
type ConfigUpdater interface {
	TriggerConfigUpdate(newConfig *config.Config)
}

// CommandHandler 命令处理器
type CommandHandler struct {
	dbClient      *Client
	config        *config.Config
	lendingBot    *strategy.LendingBot
	configUpdater ConfigUpdater
}

// NewCommandHandler 创建新的命令处理器
func NewCommandHandler(dbClient *Client, config *config.Config, lendingBot *strategy.LendingBot) *CommandHandler {
	return &CommandHandler{
		dbClient:   dbClient,
		config:     config,
		lendingBot: lendingBot,
	}
}

// SetConfigUpdater 设置配置更新器
func (h *CommandHandler) SetConfigUpdater(updater ConfigUpdater) {
	h.configUpdater = updater
}

// ProcessCommand 处理单个命令
func (h *CommandHandler) ProcessCommand(cmd *Command) error {
	log.Printf("处理命令: %s (ID: %d)", cmd.CommandType, cmd.ID)

	// 更新命令状态为处理中
	if err := h.dbClient.UpdateCommandStatus(cmd.ID, "processing", "", ""); err != nil {
		return fmt.Errorf("failed to update command status: %w", err)
	}

	var result string
	var err error

	// 根据命令类型执行相应操作
	switch cmd.CommandType {
	case "start":
		result, err = h.handleStart(cmd)
	case "stop":
		result, err = h.handleStop(cmd)
	case "restart":
		result, err = h.handleRestart(cmd)
	case "status":
		result, err = h.handleStatus(cmd)
	case "balance":
		result, err = h.handleBalance(cmd)
	case "rates":
		result, err = h.handleRates(cmd)
	case "orders":
		result, err = h.handleOrders(cmd)
	case "credits":
		result, err = h.handleCredits(cmd)
	case "config":
		result, err = h.handleConfig(cmd)
	case "cancel_all":
		result, err = h.handleCancelAll(cmd)
	case "lending_check":
		result, err = h.handleLendingCheck(cmd)
	case "rate_check":
		result, err = h.handleRateCheck(cmd)
	case "set_config":
		result, err = h.handleSetConfig(cmd)
	case "get_config":
		result, err = h.handleGetConfig(cmd)
	case "wallets":
		result, err = h.handleWallets(cmd)
	case "daily_earnings":
		result, err = h.handleDailyEarnings(cmd)
	case "update_all_config":
		result, err = h.handleUpdateAllConfig(cmd)
	default:
		err = fmt.Errorf("unknown command type: %s", cmd.CommandType)
	}

	// 更新命令完成状态
	if err != nil {
		log.Printf("命令执行失败: %v", err)
		return h.dbClient.UpdateCommandStatus(cmd.ID, "failed", "", err.Error())
	}

	log.Printf("命令执行成功: %s", result)
	return h.dbClient.UpdateCommandStatus(cmd.ID, "completed", result, "")
}

// handleStart 处理启动命令
func (h *CommandHandler) handleStart(cmd *Command) (string, error) {
	// 更新机器人状态为运行中
	status, err := h.dbClient.GetBotStatus()
	if err != nil {
		return "", err
	}

	status.Status = "running"
	// 不修改 LastExecution，保持原值
	if err := h.dbClient.UpdateBotStatus(status); err != nil {
		return "", err
	}

	return "机器人已启动", nil
}

// handleStop 处理停止命令
func (h *CommandHandler) handleStop(cmd *Command) (string, error) {
	// 更新机器人状态为停止
	status, err := h.dbClient.GetBotStatus()
	if err != nil {
		return "", err
	}

	status.Status = "stopped"
	if err := h.dbClient.UpdateBotStatus(status); err != nil {
		return "", err
	}

	return "机器人已停止", nil
}

// handleRestart 处理重启命令
func (h *CommandHandler) handleRestart(cmd *Command) (string, error) {
	// 执行主要任务（这会取消所有订单并重新下单）
	if err := h.lendingBot.Execute(); err != nil {
		return "", fmt.Errorf("重启执行失败: %w", err)
	}

	// 更新状态
	status, err := h.dbClient.GetBotStatus()
	if err != nil {
		return "", err
	}

	status.Status = "running"
	// 不修改 LastExecution，保持原值
	if err := h.dbClient.UpdateBotStatus(status); err != nil {
		return "", err
	}

	return "机器人重启完成", nil
}

// handleStatus 处理状态查询命令
func (h *CommandHandler) handleStatus(cmd *Command) (string, error) {
	status, err := h.dbClient.GetBotStatus()
	if err != nil {
		return "", err
	}

	statusJSON, err := json.Marshal(status)
	if err != nil {
		return "", fmt.Errorf("failed to marshal status: %w", err)
	}

	return string(statusJSON), nil
}

// handleBalance 处理余额查询命令
func (h *CommandHandler) handleBalance(cmd *Command) (string, error) {
	// 获取可用余额
	availableBalance, err := h.lendingBot.GetClient().GetFundingBalance(h.config.Currency)
	if err != nil {
		return "", fmt.Errorf("获取可用余额失败: %w", err)
	}

	// 获取总余额
	totalBalance, err := h.lendingBot.GetClient().GetTotalBalance(h.config.Currency)
	if err != nil {
		return "", fmt.Errorf("获取总余额失败: %w", err)
	}

	balanceInfo := map[string]interface{}{
		"currency":          h.config.Currency,
		"available_balance": availableBalance,
		"total_balance":     totalBalance,
		"timestamp":         time.Now().Unix(),
	}

	balanceJSON, err := json.Marshal(balanceInfo)
	if err != nil {
		return "", err
	}

	return string(balanceJSON), nil
}

// handleRates 处理利率查询命令
func (h *CommandHandler) handleRates(cmd *Command) (string, error) {
	rate, err := h.lendingBot.GetClient().GetCurrentFundingRate(h.config.GetFundingSymbol())
	if err != nil {
		return "", fmt.Errorf("获取利率失败: %w", err)
	}

	rateInfo := map[string]interface{}{
		"currency": h.config.Currency,
		"daily_rate": rate,
		"annual_rate": rate * 365 * 100, // 转换为年化百分比
		"timestamp": time.Now().Unix(),
	}

	rateJSON, err := json.Marshal(rateInfo)
	if err != nil {
		return "", err
	}

	return string(rateJSON), nil
}

// handleOrders 处理订单查询命令
func (h *CommandHandler) handleOrders(cmd *Command) (string, error) {
	orders, err := h.lendingBot.GetClient().GetFundingOffers(h.config.GetFundingSymbol())
	if err != nil {
		return "", fmt.Errorf("获取订单失败: %w", err)
	}

	ordersInfo := map[string]interface{}{
		"currency": h.config.Currency,
		"count": len(orders),
		"orders": orders,
		"timestamp": time.Now().Unix(),
	}

	ordersJSON, err := json.Marshal(ordersInfo)
	if err != nil {
		return "", err
	}

	return string(ordersJSON), nil
}

// handleCredits 处理已成交借贷订单查询命令
func (h *CommandHandler) handleCredits(cmd *Command) (string, error) {
	credits, err := h.lendingBot.GetClient().GetFundingCredits(h.config.GetFundingSymbol())
	if err != nil {
		return "", fmt.Errorf("获取已成交借贷订单失败: %w", err)
	}

	// 计算统计信息
	totalAmount := 0.0
	totalEarning := 0.0
	activeCounts := 0
	
	for _, credit := range credits {
		totalAmount += credit.Amount
		// 计算已赚取的利息（基于借贷期间）
		totalEarning += credit.Amount * credit.Rate * float64(credit.Period)
		// 从Bitfinex API返回的funding credits都是活跃状态
		// Status字段可能为空字符串，所以直接计算所有记录为活跃
		activeCounts++
	}

	avgRate := 0.0
	if len(credits) > 0 {
		weightedRate := 0.0
		for _, credit := range credits {
			weightedRate += credit.Rate * credit.Amount
		}
		if totalAmount > 0 {
			avgRate = weightedRate / totalAmount
		}
	}

	creditsInfo := map[string]interface{}{
		"currency":      h.config.Currency,
		"total_count":   len(credits),
		"active_count":  activeCounts,
		"total_amount":  totalAmount,
		"avg_rate":      avgRate,
		"annual_rate":   avgRate * 365 * 100, // 转换为年化百分比
		"total_earning": totalEarning,
		"credits":       credits,
		"timestamp":     time.Now().Unix(),
	}

	creditsJSON, err := json.Marshal(creditsInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal credits info: %w", err)
	}

	return string(creditsJSON), nil
}

// handleConfig 处理配置查询命令
func (h *CommandHandler) handleConfig(cmd *Command) (string, error) {
	configInfo := map[string]interface{}{
		"CURRENCY":              h.config.Currency,
		"MINUTES_RUN":           h.config.MinutesRun,
		"MIN_DAILY_LEND_RATE":   h.config.MinDailyLendRate,
		"SPREAD_LEND":           h.config.SpreadLend,
		"GAP_BOTTOM":            h.config.GapBottom,
		"GAP_TOP":               h.config.GapTop,
		"HIGH_HOLD_RATE":        h.config.HighHoldRate,
		"HIGH_HOLD_AMOUNT":      h.config.HighHoldAmount,
		"TEST_MODE":             h.config.TestMode,
		"ENABLE_SMART_STRATEGY": h.config.EnableSmartStrategy,
	}

	configJSON, err := json.Marshal(configInfo)
	if err != nil {
		return "", err
	}

	return string(configJSON), nil
}

// handleCancelAll 处理取消所有订单命令
func (h *CommandHandler) handleCancelAll(cmd *Command) (string, error) {
	// 获取所有活跃订单
	orders, err := h.lendingBot.GetClient().GetFundingOffers(h.config.GetFundingSymbol())
	if err != nil {
		return "", fmt.Errorf("获取订单失败: %w", err)
	}

	cancelledCount := 0
	// 逐个取消订单
	for _, order := range orders {
		if err := h.lendingBot.GetClient().CancelFundingOffer(order.ID); err != nil {
			log.Printf("取消订单失败 (ID: %d): %v", order.ID, err)
		} else {
			cancelledCount++
		}
	}

	result := map[string]interface{}{
		"cancelled_count": cancelledCount,
		"total_orders":    len(orders),
		"message":         fmt.Sprintf("已取消 %d/%d 个订单", cancelledCount, len(orders)),
		"timestamp":       time.Now().Unix(),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// handleLendingCheck 处理借贷检查命令
func (h *CommandHandler) handleLendingCheck(cmd *Command) (string, error) {
	if err := h.lendingBot.CheckNewLendingCredits(); err != nil {
		return "", fmt.Errorf("借贷检查失败: %w", err)
	}

	return "借贷检查完成", nil
}

// handleRateCheck 处理利率检查命令
func (h *CommandHandler) handleRateCheck(cmd *Command) (string, error) {
	exceeded, percentageRate, err := h.lendingBot.CheckRateThreshold()
	if err != nil {
		return "", fmt.Errorf("利率检查失败: %w", err)
	}

	result := map[string]interface{}{
		"threshold_exceeded": exceeded,
		"current_rate":       percentageRate,
		"threshold":          h.config.NotifyRateThreshold,
		"timestamp":          time.Now().Unix(),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// handleSetConfig 处理设置配置命令
func (h *CommandHandler) handleSetConfig(cmd *Command) (string, error) {
	// TODO: 实现动态配置更新
	// 这需要扩展配置系统以支持运行时更新
	return "配置更新功能待实现", nil
}

// handleGetConfig 处理获取配置命令
func (h *CommandHandler) handleGetConfig(cmd *Command) (string, error) {
	return h.handleConfig(cmd)
}

// handleWallets 处理钱包信息查询命令（用于调试）
func (h *CommandHandler) handleWallets(cmd *Command) (string, error) {
	wallets, err := h.lendingBot.GetClient().GetWallets()
	if err != nil {
		return "", fmt.Errorf("获取钱包信息失败: %w", err)
	}

	// 过滤指定币种的钱包
	var targetWallets []map[string]interface{}
	totalBalance := 0.0
	
	for _, wallet := range wallets {
		if wallet.Currency == h.config.Currency {
			totalBalance += wallet.Balance
			targetWallets = append(targetWallets, map[string]interface{}{
				"currency":  wallet.Currency,
				"type":      wallet.Type,
				"balance":   wallet.Balance,
				"available": wallet.Available,
			})
		}
	}

	walletsInfo := map[string]interface{}{
		"currency":       h.config.Currency,
		"wallets":        targetWallets,
		"total_balance":  totalBalance,
		"wallet_count":   len(targetWallets),
		"timestamp":      time.Now().Unix(),
	}

	walletsJSON, err := json.Marshal(walletsInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal wallets info: %w", err)
	}

	return string(walletsJSON), nil
}

// handleDailyEarnings 处理每日/每周收益查询命令
func (h *CommandHandler) handleDailyEarnings(cmd *Command) (string, error) {
	// 获取昨日收益
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	dailyEarnings, err := h.lendingBot.GetClient().GetDailyFundingEarnings(h.config.Currency, yesterday)
	if err != nil {
		return "", fmt.Errorf("获取每日收益失败: %w", err)
	}

	// 获取过去7天收益
	weeklyEarnings, err := h.lendingBot.GetClient().GetWeeklyFundingEarnings(h.config.Currency)
	if err != nil {
		return "", fmt.Errorf("获取周收益失败: %w", err)
	}

	earningsInfo := map[string]interface{}{
		"currency":        h.config.Currency,
		"date":            yesterday.Format("2006-01-02"),
		"daily_earnings":  dailyEarnings,
		"weekly_earnings": weeklyEarnings,
		"timestamp":       time.Now().Unix(),
	}

	earningsJSON, err := json.Marshal(earningsInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal earnings info: %w", err)
	}

	return string(earningsJSON), nil
}

// handleUpdateAllConfig 处理批量配置更新命令
func (h *CommandHandler) handleUpdateAllConfig(cmd *Command) (string, error) {
	// 解析命令数据中的配置参数
	var configData map[string]interface{}
	if err := json.Unmarshal(cmd.CommandData, &configData); err != nil {
		return "", fmt.Errorf("invalid config data format: %w", err)
	}

	// 创建新配置对象（基于当前配置）
	newConfig := *h.config // 复制当前配置
	
	// 更新配置字段（排除API密钥相关字段）
	if val, exists := configData["MINUTES_RUN"]; exists {
		if minutesRun, ok := val.(float64); ok {
			newConfig.MinutesRun = int(minutesRun)
		}
	}
	
	if val, exists := configData["ORDER_LIMIT"]; exists {
		if orderLimit, ok := val.(float64); ok {
			newConfig.OrderLimit = int(orderLimit)
		}
	}
	
	if val, exists := configData["MIN_LOAN"]; exists {
		if minLoan, ok := val.(float64); ok {
			newConfig.MinLoan = minLoan
		}
	}
	
	if val, exists := configData["MAX_LOAN"]; exists {
		if maxLoan, ok := val.(float64); ok {
			newConfig.MaxLoan = maxLoan
		}
	}
	
	if val, exists := configData["MIN_DAILY_LEND_RATE"]; exists {
		if minRate, ok := val.(float64); ok {
			newConfig.MinDailyLendRate = minRate
		}
	}
	
	if val, exists := configData["SPREAD_LEND"]; exists {
		if spreadLend, ok := val.(float64); ok {
			newConfig.SpreadLend = int(spreadLend)
		}
	}
	
	if val, exists := configData["GAP_BOTTOM"]; exists {
		if gapBottom, ok := val.(float64); ok {
			newConfig.GapBottom = gapBottom
		}
	}
	
	if val, exists := configData["GAP_TOP"]; exists {
		if gapTop, ok := val.(float64); ok {
			newConfig.GapTop = gapTop
		}
	}
	
	if val, exists := configData["HIGH_HOLD_RATE"]; exists {
		if highRate, ok := val.(float64); ok {
			newConfig.HighHoldRate = highRate
		}
	}
	
	if val, exists := configData["HIGH_HOLD_AMOUNT"]; exists {
		if highAmount, ok := val.(float64); ok {
			newConfig.HighHoldAmount = highAmount
		}
	}
	
	if val, exists := configData["HIGH_HOLD_ORDERS"]; exists {
		if highOrders, ok := val.(float64); ok {
			newConfig.HighHoldOrders = int(highOrders)
		}
	}
	
	if val, exists := configData["ENABLE_SMART_STRATEGY"]; exists {
		if smartStrategy, ok := val.(bool); ok {
			newConfig.EnableSmartStrategy = smartStrategy
		}
	}
	
	if val, exists := configData["VOLATILITY_THRESHOLD"]; exists {
		if threshold, ok := val.(float64); ok {
			newConfig.VolatilityThreshold = threshold
		}
	}
	
	if val, exists := configData["MAX_RATE_MULTIPLIER"]; exists {
		if maxMultiplier, ok := val.(float64); ok {
			newConfig.MaxRateMultiplier = maxMultiplier
		}
	}
	
	if val, exists := configData["MIN_RATE_MULTIPLIER"]; exists {
		if minMultiplier, ok := val.(float64); ok {
			newConfig.MinRateMultiplier = minMultiplier
		}
	}
	
	if val, exists := configData["TEST_MODE"]; exists {
		if testMode, ok := val.(bool); ok {
			newConfig.TestMode = testMode
		}
	}

	// 验证新配置的有效性
	if err := h.validateConfig(&newConfig); err != nil {
		return "", fmt.Errorf("config validation failed: %w", err)
	}

	// 写入配置文件
	if err := h.writeConfigToFile(&newConfig); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	// 触发应用程序配置热重载
	if h.configUpdater != nil {
		h.configUpdater.TriggerConfigUpdate(&newConfig)
	}

	result := map[string]interface{}{
		"message": "配置更新成功，机器人正在使用新参数重新运行",
		"updated_params": configData,
		"timestamp": time.Now().Unix(),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// validateConfig 验证配置参数的有效性
func (h *CommandHandler) validateConfig(cfg *config.Config) error {
	// 验证利率范围
	if cfg.MinDailyLendRate < 0.0001 || cfg.MinDailyLendRate > 0.1 {
		return fmt.Errorf("最低日利率必须在 0.0001 到 0.1 之间")
	}
	
	// 验证贷出金额范围
	if cfg.MinLoan < 50 || cfg.MinLoan > cfg.MaxLoan {
		return fmt.Errorf("最小贷出金额必须大于50且小于最大贷出金额")
	}
	
	if cfg.MaxLoan > 50000 {
		return fmt.Errorf("最大贷出金额不能超过50000")
	}
	
	// 验证订单限制
	if cfg.OrderLimit < 1 || cfg.OrderLimit > 100 {
		return fmt.Errorf("订单限制必须在 1 到 100 之间")
	}
	
	// 验证时间间隔
	if cfg.MinutesRun < 5 || cfg.MinutesRun > 240 {
		return fmt.Errorf("执行间隔必须在 5 到 240 分钟之间")
	}
	
	// 验证分散笔数
	if cfg.SpreadLend < 1 || cfg.SpreadLend > 50 {
		return fmt.Errorf("分散笔数必须在 1 到 50 之间")
	}
	
	return nil
}

// writeConfigToFile 将配置写入文件
func (h *CommandHandler) writeConfigToFile(cfg *config.Config) error {
	// 配置文件路径（假设在容器中的路径）
	configPath := "/app/config.yaml"
	
	// 写入配置文件
	if err := config.WriteConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}
	
	log.Printf("配置已成功写入到文件: %s", configPath)
	return nil
}