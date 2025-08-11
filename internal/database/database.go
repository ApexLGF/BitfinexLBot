package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Client MySQL 数据库客户端
type Client struct {
	db *sql.DB
}

// NewClient 创建新的数据库客户端
func NewClient(dsn string) (*Client, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{db: db}, nil
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	return c.db.Close()
}

// Command 代表一个 Web 命令
type Command struct {
	ID          int             `json:"id"`
	UserID      int             `json:"user_id"`
	CommandType string          `json:"command_type"`
	CommandData json.RawMessage `json:"command_data,omitempty"`
	Status      string          `json:"status"`
	Result      string          `json:"result,omitempty"`
	ErrorMsg    string          `json:"error_message,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// BotStatus 机器人状态
type BotStatus struct {
	ID                  int             `json:"id"`
	Status              string          `json:"status"`
	LastExecution       *time.Time      `json:"last_execution,omitempty"`
	NextExecution       *time.Time      `json:"next_execution,omitempty"`
	TotalBalance        float64         `json:"total_balance"`
	AvailableBalance    float64         `json:"available_balance"`
	ActiveOrders        int             `json:"active_orders"`
	CurrentRate         float64         `json:"current_rate"`
	AvgRate             float64         `json:"avg_rate"`
	TotalEarned         float64         `json:"total_earned"`
	WeeklyEarned        float64         `json:"weekly_earned"`
	ErrorCount          int             `json:"error_count"`
	LastError           string          `json:"last_error,omitempty"`
	PerformanceMetrics  json.RawMessage `json:"performance_metrics,omitempty"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// Notification 通知
type Notification struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// GetPendingCommands 获取待处理的命令
func (c *Client) GetPendingCommands() ([]*Command, error) {
	// 按命令优先级和创建时间排序
	// 高优先级命令：stop, start, restart, cancel_all
	// 中优先级命令：status, balance, rates, orders
	// 低优先级命令：config, lending_check, rate_check
	query := `
		SELECT id, user_id, command_type, command_data, status, result, error_message, 
		       created_at, processed_at, completed_at
		FROM web_commands 
		WHERE status = 'pending' 
		ORDER BY 
			CASE command_type
				WHEN 'stop' THEN 1
				WHEN 'start' THEN 2
				WHEN 'restart' THEN 3
				WHEN 'cancel_all' THEN 4
				WHEN 'status' THEN 5
				WHEN 'balance' THEN 6
				WHEN 'rates' THEN 7
				WHEN 'orders' THEN 8
				WHEN 'config' THEN 9
				WHEN 'lending_check' THEN 10
				WHEN 'rate_check' THEN 11
				ELSE 12
			END,
			created_at ASC 
		LIMIT 10`

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending commands: %w", err)
	}
	defer rows.Close()

	var commands []*Command
	for rows.Next() {
		cmd := &Command{}
		var commandData sql.NullString
		var result sql.NullString
		var errorMsg sql.NullString
		var processedAt sql.NullTime
		var completedAt sql.NullTime

		err := rows.Scan(
			&cmd.ID, &cmd.UserID, &cmd.CommandType, &commandData, &cmd.Status,
			&result, &errorMsg, &cmd.CreatedAt, &processedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}

		if commandData.Valid {
			cmd.CommandData = json.RawMessage(commandData.String)
		}
		if result.Valid {
			cmd.Result = result.String
		}
		if errorMsg.Valid {
			cmd.ErrorMsg = errorMsg.String
		}
		if processedAt.Valid {
			cmd.ProcessedAt = &processedAt.Time
		}
		if completedAt.Valid {
			cmd.CompletedAt = &completedAt.Time
		}

		commands = append(commands, cmd)
	}

	return commands, rows.Err()
}

// UpdateCommandStatus 更新命令状态
func (c *Client) UpdateCommandStatus(commandID int, status string, result string, errorMsg string) error {
	var query string
	var args []interface{}

	switch status {
	case "processing":
		query = "UPDATE web_commands SET status = ?, processed_at = NOW() WHERE id = ?"
		args = []interface{}{status, commandID}
	case "completed", "failed":
		query = "UPDATE web_commands SET status = ?, result = ?, error_message = ?, completed_at = NOW() WHERE id = ?"
		args = []interface{}{status, result, errorMsg, commandID}
	default:
		query = "UPDATE web_commands SET status = ? WHERE id = ?"
		args = []interface{}{status, commandID}
	}

	_, err := c.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update command status: %w", err)
	}

	return nil
}

// GetBotStatus 获取机器人状态
func (c *Client) GetBotStatus() (*BotStatus, error) {
	query := `
		SELECT id, status, last_execution, next_execution, total_balance, available_balance,
		       active_orders, current_rate, avg_rate, total_earned, weekly_earned, error_count, last_error,
		       performance_metrics, updated_at
		FROM bot_realtime_status 
		ORDER BY id DESC 
		LIMIT 1`

	status := &BotStatus{}
	var lastExecution sql.NullTime
	var nextExecution sql.NullTime
	var lastError sql.NullString
	var performanceMetrics sql.NullString

	err := c.db.QueryRow(query).Scan(
		&status.ID, &status.Status, &lastExecution, &nextExecution,
		&status.TotalBalance, &status.AvailableBalance, &status.ActiveOrders,
		&status.CurrentRate, &status.AvgRate, &status.TotalEarned, &status.WeeklyEarned,
		&status.ErrorCount, &lastError, &performanceMetrics, &status.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// 如果没有记录，创建一个默认状态
			return c.createDefaultBotStatus()
		}
		return nil, fmt.Errorf("failed to get bot status: %w", err)
	}

	if lastExecution.Valid {
		status.LastExecution = &lastExecution.Time
	}
	if nextExecution.Valid {
		status.NextExecution = &nextExecution.Time
	}
	if lastError.Valid {
		status.LastError = lastError.String
	}
	if performanceMetrics.Valid {
		status.PerformanceMetrics = json.RawMessage(performanceMetrics.String)
	}

	return status, nil
}

// UpdateBotStatus 更新机器人状态
func (c *Client) UpdateBotStatus(status *BotStatus) error {
	query := `
		UPDATE bot_realtime_status SET 
		status = ?, last_execution = ?, next_execution = ?, total_balance = ?, 
		available_balance = ?, active_orders = ?, current_rate = ?, avg_rate = ?, 
		total_earned = ?, weekly_earned = ?, error_count = ?, last_error = ?, performance_metrics = ?
		WHERE id = ?`

	// 处理可能为nil的时间字段
	var lastExec, nextExec interface{}
	if status.LastExecution != nil {
		lastExec = status.LastExecution
	}
	if status.NextExecution != nil {
		nextExec = status.NextExecution
	}

	args := []interface{}{
		status.Status, lastExec, nextExec, 
		status.TotalBalance, status.AvailableBalance, status.ActiveOrders,
		status.CurrentRate, status.AvgRate, status.TotalEarned, status.WeeklyEarned,
		status.ErrorCount, status.LastError, status.PerformanceMetrics,
		status.ID,
	}

	_, err := c.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update bot status: %w", err)
	}

	return nil
}

// SaveNotification 保存通知
func (c *Client) SaveNotification(notificationType, title, message, level string) error {
	query := `
		INSERT INTO notifications (type, title, message, level) 
		VALUES (?, ?, ?, ?)`

	_, err := c.db.Exec(query, notificationType, title, message, level)
	if err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	return nil
}

// GetRecentNotifications 获取最近的通知
func (c *Client) GetRecentNotifications(limit int) ([]*Notification, error) {
	query := `
		SELECT id, type, title, message, level, is_read, created_at
		FROM notifications 
		ORDER BY created_at DESC 
		LIMIT ?`

	rows, err := c.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		err := rows.Scan(
			&notification.ID, &notification.Type, &notification.Title,
			&notification.Message, &notification.Level, &notification.IsRead,
			&notification.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	return notifications, rows.Err()
}

// createDefaultBotStatus 创建默认的机器人状态
func (c *Client) createDefaultBotStatus() (*BotStatus, error) {
	query := `
		INSERT INTO bot_realtime_status (status) VALUES ('stopped')
		ON DUPLICATE KEY UPDATE status = status`

	_, err := c.db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create default bot status: %w", err)
	}

	return c.GetBotStatus()
}

// LogOperation 记录操作日志
func (c *Client) LogOperation(userID int, operationType, operationDesc, ipAddress, userAgent, result string, details json.RawMessage) error {
	query := `
		INSERT INTO operation_logs (user_id, operation_type, operation_desc, ip_address, user_agent, result, details) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := c.db.Exec(query, userID, operationType, operationDesc, ipAddress, userAgent, result, details)
	if err != nil {
		return fmt.Errorf("failed to log operation: %w", err)
	}

	return nil
}

// DailyEarningSummary 每日收益汇总数据结构
type DailyEarningSummary struct {
	ID            int     `json:"id"`
	Currency      string  `json:"currency"`
	SummaryDate   string  `json:"summary_date"` // YYYY-MM-DD格式
	TotalTrades   int     `json:"total_trades"`
	TotalAmount   float64 `json:"total_amount"`
	TotalEarnings float64 `json:"total_earnings"`
	AvgRate       float64 `json:"avg_rate"`
	MinRate       float64 `json:"min_rate"`
	MaxRate       float64 `json:"max_rate"`
	AvgPeriod     float64 `json:"avg_period"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SaveDailyEarningSummary 保存每日收益汇总数据
func (c *Client) SaveDailyEarningSummary(summary *DailyEarningSummary) error {
	query := `
		INSERT INTO daily_earnings_summary 
		(currency, summary_date, total_trades, total_amount, total_earnings, avg_rate, min_rate, max_rate, avg_period)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		total_trades = VALUES(total_trades),
		total_amount = VALUES(total_amount),
		total_earnings = VALUES(total_earnings),
		avg_rate = VALUES(avg_rate),
		min_rate = VALUES(min_rate),
		max_rate = VALUES(max_rate),
		avg_period = VALUES(avg_period),
		updated_at = CURRENT_TIMESTAMP`

	_, err := c.db.Exec(query, 
		summary.Currency, summary.SummaryDate, summary.TotalTrades, summary.TotalAmount,
		summary.TotalEarnings, summary.AvgRate, summary.MinRate, summary.MaxRate, summary.AvgPeriod)
	if err != nil {
		return fmt.Errorf("failed to save daily earning summary: %w", err)
	}

	return nil
}

// GetDailyEarningSummary 获取指定日期的每日收益汇总
func (c *Client) GetDailyEarningSummary(currency string, summaryDate string) (*DailyEarningSummary, error) {
	query := `
		SELECT id, currency, summary_date, total_trades, total_amount, total_earnings, 
			   avg_rate, min_rate, max_rate, avg_period, created_at, updated_at
		FROM daily_earnings_summary 
		WHERE currency = ? AND summary_date = ?`

	summary := &DailyEarningSummary{}
	err := c.db.QueryRow(query, currency, summaryDate).Scan(
		&summary.ID, &summary.Currency, &summary.SummaryDate,
		&summary.TotalTrades, &summary.TotalAmount, &summary.TotalEarnings,
		&summary.AvgRate, &summary.MinRate, &summary.MaxRate, &summary.AvgPeriod,
		&summary.CreatedAt, &summary.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 没有记录
		}
		return nil, fmt.Errorf("failed to get daily earning summary: %w", err)
	}

	return summary, nil
}

// GetRecentDailyEarningSummaries 获取最近的每日收益汇总列表
func (c *Client) GetRecentDailyEarningSummaries(currency string, days int) ([]*DailyEarningSummary, error) {
	query := `
		SELECT id, currency, summary_date, total_trades, total_amount, total_earnings,
			   avg_rate, min_rate, max_rate, avg_period, created_at, updated_at
		FROM daily_earnings_summary 
		WHERE currency = ?
		ORDER BY summary_date DESC 
		LIMIT ?`

	rows, err := c.db.Query(query, currency, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily earning summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*DailyEarningSummary
	for rows.Next() {
		summary := &DailyEarningSummary{}
		err := rows.Scan(
			&summary.ID, &summary.Currency, &summary.SummaryDate,
			&summary.TotalTrades, &summary.TotalAmount, &summary.TotalEarnings,
			&summary.AvgRate, &summary.MinRate, &summary.MaxRate, &summary.AvgPeriod,
			&summary.CreatedAt, &summary.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily earning summary: %w", err)
		}
		summaries = append(summaries, summary)
	}

	return summaries, rows.Err()
}

// SaveDailyEarningSummaryParams 接口适配器方法，用于兼容 bitfinex.DatabaseSaver 接口
func (c *Client) SaveDailyEarningSummaryParams(currency, summaryDate string, totalTrades int, totalAmount, totalEarnings, avgRate, minRate, maxRate, avgPeriod float64) error {
	summary := &DailyEarningSummary{
		Currency:      currency,
		SummaryDate:   summaryDate,
		TotalTrades:   totalTrades,
		TotalAmount:   totalAmount,
		TotalEarnings: totalEarnings,
		AvgRate:       avgRate,
		MinRate:       minRate,
		MaxRate:       maxRate,
		AvgPeriod:     avgPeriod,
	}
	return c.SaveDailyEarningSummary(summary)
}