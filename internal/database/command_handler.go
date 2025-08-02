package database

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ApexLGF/BitfinexLBot/internal/config"
	"github.com/ApexLGF/BitfinexLBot/internal/strategy"
)

// CommandHandler 命令处理器
type CommandHandler struct {
	dbClient   *Client
	config     *config.Config
	lendingBot *strategy.LendingBot
}

// NewCommandHandler 创建新的命令处理器
func NewCommandHandler(dbClient *Client, config *config.Config, lendingBot *strategy.LendingBot) *CommandHandler {
	return &CommandHandler{
		dbClient:   dbClient,
		config:     config,
		lendingBot: lendingBot,
	}
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
		if credit.Status == "ACTIVE" {
			activeCounts++
		}
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