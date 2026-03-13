package api

import "time"

// APIResponse 通用API响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// BotStatus 机器人状态
type BotStatus struct {
	IsRunning     bool      `json:"is_running"`
	LastUpdate    time.Time `json:"last_update"`
	NextRun       time.Time `json:"next_run"`
	TotalEarnings float64   `json:"total_earnings"`
	ActiveOffers  int       `json:"active_offers"`
	AvailableFunds float64  `json:"available_funds"`
	TotalFunds    float64   `json:"total_funds"`
	Currency      string    `json:"currency"`
	Errors        []string  `json:"errors,omitempty"` // API调用错误信息
}

// EarningsData 收益数据
type EarningsData struct {
	Daily    float64           `json:"daily"`
	Weekly   float64           `json:"weekly"`
	Monthly  float64           `json:"monthly"`
	Currency string            `json:"currency"` // 新增币种字段
	History  []EarningsHistory `json:"history"`
}

// EarningsHistory 收益历史记录
type EarningsHistory struct {
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Rate     float64 `json:"rate"`
	Currency string  `json:"currency"`
}

// OfferData 放贷订单数据
type OfferData struct {
	ID       int64   `json:"id"`
	Amount   float64 `json:"amount"`
	Rate     float64 `json:"rate"`
	Period   int     `json:"period"`
	Status   string  `json:"status"`
	Created  time.Time `json:"created"`
	Currency string  `json:"currency"`
}

// ConfigUpdateRequest 配置更新请求
type ConfigUpdateRequest struct {
	Config map[string]interface{} `json:"config"`
}

// ControlRequest 机器人控制请求
type ControlRequest struct {
	Action string `json:"action"` // start, stop, restart
}

// LogEntry 日志条目
type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Component string    `json:"component"`
}

// LogResponse 日志响应
type LogResponse struct {
	Logs  []LogEntry `json:"logs"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

// FundingCreditData 已贷出订单数据
type FundingCreditData struct {
	ID       int64     `json:"id"`
	Symbol   string    `json:"symbol"`
	Amount   float64   `json:"amount"`
	Rate     float64   `json:"rate"`
	Period   int64     `json:"period"`
	Status   string    `json:"status"`
	Created  time.Time `json:"created"`
	Opened   time.Time `json:"opened"`
	Currency string    `json:"currency"`
}

// FundingCreditsResponse 已贷出订单响应
type FundingCreditsResponse struct {
	Credits     []FundingCreditData `json:"credits"`
	TotalCount  int                 `json:"total_count"`
	TotalAmount float64             `json:"total_amount"`
	AvgRate     float64             `json:"avg_rate"`
}