package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
	"github.com/ApexLGF/BitfinexLBot/internal/constants"
	"github.com/ApexLGF/BitfinexLBot/internal/database"
	"github.com/ApexLGF/BitfinexLBot/internal/rates"
	"github.com/ApexLGF/BitfinexLBot/internal/strategy"
)

// Application 應用程式主結構
type Application struct {
	config         *config.Config
	bfxClient      *bitfinex.Client
	lendingBot     *strategy.LendingBot
	rateConverter  *rates.Converter
	dbClient       *database.Client
	commandHandler *database.CommandHandler

	// 併發控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 配置热更新
	configUpdateChan chan *config.Config
	configPath       string
}

// NewApplication 創建新的應用程式實例
func NewApplication(configPath string) (*Application, error) {
	// 載入配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 創建 Bitfinex 客戶端
	bfxClient := bitfinex.NewClient(cfg.BitfinexApiKey, cfg.BitfinexSecretKey)

	// 創建數據庫客戶端
	dbClient, err := database.NewClient(cfg.GetDatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	// 創建貸出機器人
	lendingBot := strategy.NewLendingBot(cfg, bfxClient)

	// 創建利率轉換器
	rateConverter := rates.NewConverter()

	// 創建命令處理器
	commandHandler := database.NewCommandHandler(dbClient, cfg, lendingBot)

	// 創建 context 和 cancel 函數
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		config:           cfg,
		bfxClient:        bfxClient,
		lendingBot:       lendingBot,
		rateConverter:    rateConverter,
		dbClient:         dbClient,
		commandHandler:   commandHandler,
		ctx:              ctx,
		cancel:           cancel,
		configUpdateChan: make(chan *config.Config, 1),
		configPath:       configPath,
	}

	// 设置配置更新器，建立CommandHandler和Application的关联
	commandHandler.SetConfigUpdater(app)

	return app, nil
}

// Run 運行應用程式
func (app *Application) Run() error {
	log.Printf("Config loaded successfully: %+v", app.config)

	// 顯示運行模式
	if app.config.TestMode {
		log.Println("🧪 === 測試模式啟動 ===")
		log.Println("🧪 不會執行真實的下單操作")
		log.Println("🧪 但會執行真實的取消操作")
	} else {
		log.Println("🚀 === 正式模式啟動 ===")
		log.Println("🚀 將執行真實的交易操作")
	}

	// 設置信號處理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 啟動所有 goroutines
	app.startWorkers()

	log.Printf("Scheduler started at: %v", time.Now())
	log.Printf("⚙️ 主要任務間隔: %d 分鐘", app.config.MinutesRun)
	log.Printf("💰 借貸檢查間隔: %d 分鐘", app.config.LendingCheckMinutes)
	log.Printf("📊 利率檢查: 每小時")
	log.Printf("💵 收益更新: 每小時")
	log.Println("🔄 按 Ctrl+C 優雅關閉...")

	// 等待信號或 context 取消
	select {
	case sig := <-sigChan:
		log.Printf("收到信號 %v，開始優雅關閉...", sig)
	case <-app.ctx.Done():
		log.Println("Context 被取消，開始關閉...")
	}

	return app.shutdown()
}

// startWorkers 啟動所有工作 goroutines
func (app *Application) startWorkers() {
	// 啟動命令檢查器（替代 Telegram Bot）
	app.wg.Add(1)
	go app.runWorker("CommandChecker", func() {
		defer app.wg.Done()
		app.scheduleCommandCheck()
	})

	// 啟動狀態更新器
	app.wg.Add(1)
	go app.runWorker("StatusUpdater", func() {
		defer app.wg.Done()
		app.scheduleStatusUpdate()
	})

	// 啟動每小時利率檢查
	app.wg.Add(1)
	go app.runWorker("HourlyRateCheck", func() {
		defer app.wg.Done()
		app.scheduleHourlyRateCheck()
	})

	// 啟動每小時收益更新
	app.wg.Add(1)
	go app.runWorker("HourlyEarningsUpdate", func() {
		defer app.wg.Done()
		app.scheduleHourlyEarningsUpdate()
	})

	// 啟動借貸訂單檢查
	app.wg.Add(1)
	go app.runWorker("LendingCheck", func() {
		defer app.wg.Done()
		app.scheduleLendingCheck()
	})

	// 啟動主要業務邏輯調度
	app.wg.Add(1)
	go app.runWorker("MainTask", func() {
		defer app.wg.Done()
		app.scheduleMainTask()
	})

	// 啟動配置热更新监听器
	app.wg.Add(1)
	go app.runWorker("ConfigHotReload", func() {
		defer app.wg.Done()
		app.listenForConfigUpdates()
	})
}

// runWorker 安全運行工作任務
func (app *Application) runWorker(name string, worker func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("工作任務 %s 發生 panic: %v", name, r)
			// 可以在這裡添加重啟邏輯
		}
	}()

	log.Printf("啟動工作任務: %s", name)
	worker()
	log.Printf("工作任務 %s 已結束", name)
}

// shutdown 優雅關閉應用程式
func (app *Application) shutdown() error {
	log.Println("正在關閉應用程式...")

	// 取消 context
	app.cancel()

	// 等待所有 goroutines 結束，設置超時
	done := make(chan struct{})
	go func() {
		app.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("所有工作任務已優雅結束")
	case <-time.After(constants.ShutdownTimeout):
		log.Println("等待超時，強制結束")
	}

	// 關閉數據庫連接
	if app.dbClient != nil {
		if err := app.dbClient.Close(); err != nil {
			log.Printf("關閉數據庫連接失敗: %v", err)
		}
	}

	log.Println("應用程式已關閉")
	return nil
}

// scheduleMainTask 調度主要任務
func (app *Application) scheduleMainTask() {
	// 先執行第一次
	app.executeMainTask()

	ticker := time.NewTicker(time.Duration(app.config.MinutesRun) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			log.Println("主要任務調度器收到停止信號")
			return
		case <-ticker.C:
			app.executeMainTask()
		}
	}
}

// executeMainTask 執行主要任務
func (app *Application) executeMainTask() {
	if err := app.lendingBot.Execute(); err != nil {
		log.Printf("執行貸出策略失敗: %v", err)
	}
}

// scheduleHourlyRateCheck 調度每小時利率檢查
func (app *Application) scheduleHourlyRateCheck() {
	for {
		select {
		case <-app.ctx.Done():
			log.Println("利率檢查調度器收到停止信號")
			return
		default:
		}

		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), constants.HourlyCheckMinute, 0, 0, now.Location())
		if now.After(next) || now.Equal(next) {
			next = next.Add(time.Hour)
		}

		delay := next.Sub(now)
		log.Printf("下次執行時間: %s, 等待時間: %s", next.Format("2006-01-02 15:04:05"), delay)

		// 使用 context 支持的 sleep
		select {
		case <-app.ctx.Done():
			log.Println("利率檢查調度器在等待中收到停止信號")
			return
		case <-time.After(delay):
			app.checkRateThreshold()
		}
	}
}

// checkRateThreshold 檢查利率閾值
func (app *Application) checkRateThreshold() {
	log.Println("定時檢查貸出利率（基於5分鐘K線12根高點）...")

	exceeded, percentageRate, err := app.lendingBot.CheckRateThreshold()
	if err != nil {
		log.Printf("取得利率數據失敗: %v", err)
		return
	}

	log.Printf("最近1小時最高利率: %.4f%%, 閾值: %.4f%%", percentageRate, app.config.NotifyRateThreshold)

	if exceeded {
		message := fmt.Sprintf("⚠️ 定時檢查提醒: 最近1小時最高利率 %.4f%% 已超過閾值 %.4f%%\n\n📊 檢查方式: 5分鐘K線最近12根高點分析",
			percentageRate, app.config.NotifyRateThreshold)

		// 記錄通知到數據庫（替代 Telegram 通知）
		if err := app.saveNotification("rate_threshold", message); err != nil {
			log.Printf("保存利率通知失敗: %v", err)
		} else {
			log.Printf("成功記錄利率提醒到數據庫")
		}
	} else {
		log.Println("最近1小時最高利率低於閾值，無需發送通知")
	}
}

// handleRestart 處理重啟請求
func (app *Application) handleRestart() error {
	log.Println("收到重啟請求，開始執行重啟邏輯...")

	// 執行主要任務（這會取消所有訂單並重新下單）
	if err := app.lendingBot.Execute(); err != nil {
		log.Printf("重啟執行失敗: %v", err)
		return fmt.Errorf("重啟執行失敗: %w", err)
	}

	log.Println("重啟完成！")
	return nil
}

// scheduleLendingCheck 調度借貸訂單檢查
func (app *Application) scheduleLendingCheck() {
	// 先執行第一次檢查
	app.executeLendingCheck()

	ticker := time.NewTicker(time.Duration(app.config.LendingCheckMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			log.Println("借貸檢查調度器收到停止信號")
			return
		case <-ticker.C:
			app.executeLendingCheck()
		}
	}
}

// executeLendingCheck 執行借貸訂單檢查
func (app *Application) executeLendingCheck() {
	if err := app.lendingBot.CheckNewLendingCredits(); err != nil {
		log.Printf("檢查借貸訂單失敗: %v", err)
	}
}

// listenForConfigUpdates 监听配置更新事件
func (app *Application) listenForConfigUpdates() {
	log.Println("启动配置热更新监听器")

	for {
		select {
		case <-app.ctx.Done():
			log.Println("配置热更新监听器收到停止信号")
			return
		case newConfig := <-app.configUpdateChan:
			log.Println("收到配置更新信号，开始热重载...")
			if err := app.hotReloadConfig(newConfig); err != nil {
				log.Printf("配置热重载失败: %v", err)
			} else {
				log.Println("配置热重载成功完成！")
			}
		}
	}
}

// hotReloadConfig 执行配置热重载
func (app *Application) hotReloadConfig(newConfig *config.Config) error {
	log.Println("🔄 开始配置热重载流程...")

	// 1. 取消所有活跃的放贷订单
	log.Println("📝 取消所有活跃订单...")
	offers, err := app.bfxClient.GetFundingOffers(app.config.GetFundingSymbol())
	if err != nil {
		log.Printf("获取活跃订单失败: %v", err)
	} else {
		cancelledCount := 0
		for _, offer := range offers {
			if err := app.bfxClient.CancelFundingOffer(offer.ID); err != nil {
				log.Printf("取消订单失败 (ID: %d): %v", offer.ID, err)
			} else {
				cancelledCount++
			}
		}
		log.Printf("✅ 已取消 %d/%d 个订单", cancelledCount, len(offers))
	}

	// 2. 更新应用程序配置
	log.Println("🔧 更新应用程序配置...")
	app.config = newConfig

	// 3. 重新创建 LendingBot 实例（使用新配置）
	log.Println("🤖 重新初始化放贷机器人...")
	app.lendingBot = strategy.NewLendingBot(newConfig, app.bfxClient)

	// 4. 更新 CommandHandler 的配置引用
	app.commandHandler = database.NewCommandHandler(app.dbClient, newConfig, app.lendingBot)

	// 5. 短暂等待，然后开始新的放贷策略
	log.Println("⏳ 等待2秒后开始新的放贷策略...")
	time.Sleep(2 * time.Second)

	// 6. 执行新的放贷策略
	log.Println("🚀 使用新配置执行放贷策略...")
	if err := app.lendingBot.Execute(); err != nil {
		return fmt.Errorf("执行新放贷策略失败: %w", err)
	}

	log.Println("✨ 配置热重载完成，机器人已使用新参数运行")
	return nil
}

// TriggerConfigUpdate 触发配置更新（供外部调用）
func (app *Application) TriggerConfigUpdate(newConfig *config.Config) {
	select {
	case app.configUpdateChan <- newConfig:
		log.Println("配置更新信号已发送")
	default:
		log.Println("配置更新通道繁忙，跳过此次更新")
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "bitfinex-lending-bot"
	app.Version = "v2.0.0"
	app.Usage = "Automated Bitfinex lending bot with v2 API"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  "config.yaml",
			Usage:  "Configuration file path",
			EnvVar: "CONFIG_PATH",
		},
	}

	app.Action = func(c *cli.Context) error {
		configPath := c.String("config")

		application, err := NewApplication(configPath)
		if err != nil {
			log.Fatalf("Failed to create application: %v", err)
		}

		return application.Run()
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

// scheduleCommandCheck 調度命令檢查（替代 Telegram Bot）
func (app *Application) scheduleCommandCheck() {
	// 优化的轮询策略：有待处理命令时保持高频，无命令时降低频率
	fastInterval := 500 * time.Millisecond  // 有待处理命令时的快速间隔
	normalInterval := 2 * time.Second       // 正常间隔
	slowInterval := 5 * time.Second         // 无命令时的慢间隔
	
	currentInterval := fastInterval
	consecutiveEmptyChecks := 0
	maxEmptyChecks := 10 // 连续10次无命令后降低频率

	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	log.Printf("启动优化命令检查调度器，初始间隔: %v", currentInterval)

	for {
		select {
		case <-app.ctx.Done():
			log.Println("命令檢查調度器收到停止信號")
			return
		case <-ticker.C:
			// 检查待处理命令
			commandCount := app.checkPendingCommandsWithCount()
			
			var newInterval time.Duration
			
			if commandCount > 0 {
				// 有命令处理时，使用快速间隔
				newInterval = fastInterval
				consecutiveEmptyChecks = 0
			} else {
				// 无命令时，逐步降低频率
				consecutiveEmptyChecks++
				if consecutiveEmptyChecks <= 5 {
					newInterval = fastInterval  // 前5次仍保持快速检查
				} else if consecutiveEmptyChecks <= maxEmptyChecks {
					newInterval = normalInterval // 接下来5次使用正常间隔
				} else {
					newInterval = slowInterval  // 超过10次后使用慢间隔
				}
			}
			
			// 动态调整轮询间隔
			if newInterval != currentInterval {
				log.Printf("调整命令检查间隔: %v -> %v (命令数: %d, 空检查次数: %d)",
					currentInterval, newInterval, commandCount, consecutiveEmptyChecks)
				
				ticker.Stop()
				ticker = time.NewTicker(newInterval)
				currentInterval = newInterval
			}
		}
	}
}


// scheduleStatusUpdate 調度狀態更新
func (app *Application) scheduleStatusUpdate() {
	// 动态调整状态更新间隔
	runningInterval := 30 * time.Second // 运行时更频繁更新
	stoppedInterval := 60 * time.Second // 停止时较少更新

	currentInterval := runningInterval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	log.Printf("启动状态更新调度器，初始间隔: %v", currentInterval)

	for {
		select {
		case <-app.ctx.Done():
			log.Println("狀態更新調度器收到停止信號")
			return
		case <-ticker.C:
			// 更新状态并获取当前状态
			isRunning := app.updateBotStatusWithState()

			// 根据运行状态调整更新间隔
			var targetInterval time.Duration
			if isRunning {
				targetInterval = runningInterval
			} else {
				targetInterval = stoppedInterval
			}

			// 如果间隔需要调整
			if targetInterval != currentInterval {
				log.Printf("调整状态更新间隔: %v -> %v (机器人状态: %s)",
					currentInterval, targetInterval,
					map[bool]string{true: "运行中", false: "已停止"}[isRunning])

				ticker.Stop()
				ticker = time.NewTicker(targetInterval)
				currentInterval = targetInterval
			}
		}
	}
}

// checkPendingCommands 檢查待處理的命令
func (app *Application) checkPendingCommands() {
	app.checkPendingCommandsWithCount()
}

// checkPendingCommandsWithCount 檢查待處理的命令並返回命令數量
func (app *Application) checkPendingCommandsWithCount() int {
	commands, err := app.dbClient.GetPendingCommands()
	if err != nil {
		log.Printf("獲取待處理命令失敗: %v", err)
		return 0
	}

	if len(commands) == 0 {
		return 0 // 没有待处理的命令
	}

	log.Printf("發現 %d 個待處理命令", len(commands))

	processedCount := 0
	// 處理每個命令
	for _, cmd := range commands {
		if err := app.commandHandler.ProcessCommand(cmd); err != nil {
			log.Printf("處理命令失敗 (ID: %d): %v", cmd.ID, err)
		} else {
			processedCount++
		}
	}

	return processedCount
}

// updateBotStatus 更新機器人狀態
func (app *Application) updateBotStatus() {
	app.updateBotStatusWithState()
}

// updateBotStatusWithState 更新機器人狀態並返回是否運行中
func (app *Application) updateBotStatusWithState() bool {
	// 獲取當前狀態
	status, err := app.dbClient.GetBotStatus()
	if err != nil {
		log.Printf("獲取機器人狀態失敗: %v", err)
		return false
	}

	// 更新狀態信息
	err = app.updateStatusMetrics(status)
	if err != nil {
		log.Printf("更新狀態指標失敗: %v", err)
		return false
	}

	// 保存更新的狀態
	if err := app.dbClient.UpdateBotStatus(status); err != nil {
		log.Printf("保存機器人狀態失敗: %v", err)
		return false
	}

	return status.Status == "running"
}

// saveNotification 保存通知到數據庫
func (app *Application) saveNotification(notificationType, message string) error {
	return app.dbClient.SaveNotification(notificationType, "系統通知", message, "warning")
}

// updateStatusMetrics 更新狀態指標
func (app *Application) updateStatusMetrics(status *database.BotStatus) error {
	// 獲取可用餘額
	availableBalance, err := app.lendingBot.GetClient().GetFundingBalance(app.config.Currency)
	if err != nil {
		log.Printf("獲取可用餘額失敗: %v", err)
	} else {
		status.AvailableBalance = availableBalance
	}

	// 獲取總錢包餘額
	totalBalance, err := app.lendingBot.GetClient().GetTotalBalance(app.config.Currency)
	if err != nil {
		log.Printf("獲取總餘額失敗: %v", err)
	} else {
		status.TotalBalance = totalBalance
	}

	// 獲取當前利率
	rate, err := app.lendingBot.GetClient().GetCurrentFundingRate(app.config.GetFundingSymbol())
	if err != nil {
		log.Printf("獲取利率失敗: %v", err)
	} else {
		status.CurrentRate = rate
	}

	// 收益数据由专门的每小时任务更新，此处不再获取以提高性能
	// 如果数据库中没有收益数据，保持为0
	if status.TotalEarned == 0.0 {
		status.TotalEarned = 0.0
	}
	if status.WeeklyEarned == 0.0 {
		status.WeeklyEarned = 0.0
	}

	// 設置下次執行時間
	if status.Status == "running" {
		nextExecution := time.Now().Add(time.Duration(app.config.MinutesRun) * time.Minute)
		status.NextExecution = &nextExecution
	}

	return nil
}

// scheduleHourlyEarningsUpdate 调度每小时收益更新
func (app *Application) scheduleHourlyEarningsUpdate() {
	// 先执行第一次更新
	app.updateEarnings()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	log.Printf("启动每小时收益更新调度器")

	for {
		select {
		case <-app.ctx.Done():
			log.Println("收益更新调度器收到停止信号")
			return
		case <-ticker.C:
			app.updateEarnings()
		}
	}
}

// updateEarnings 更新收益数据
func (app *Application) updateEarnings() {
	log.Println("开始更新24小时和1周收益数据...")

	// 获取当前状态
	status, err := app.dbClient.GetBotStatus()
	if err != nil {
		log.Printf("获取机器人状态失败: %v", err)
		return
	}

	// 获取过去24小时收益
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	dailyEarnings, err := app.lendingBot.GetClient().GetDailyFundingEarnings(app.config.Currency, yesterday)
	if err != nil {
		log.Printf("获取24小时收益失败: %v", err)
	} else {
		status.TotalEarned = dailyEarnings
		log.Printf("✅ 24小时收益: %.6f %s", dailyEarnings, app.config.Currency)
	}

	// 获取过去7天收益并保存每日收益数据到数据库
	weeklyEarnings, err := app.lendingBot.GetClient().GetWeeklyFundingEarningsWithDB(app.config.Currency, app.dbClient)
	if err != nil {
		log.Printf("获取1周收益失败: %v", err)
	} else {
		status.WeeklyEarned = weeklyEarnings
		log.Printf("✅ 1周收益: %.6f %s (已保存每日收益数据)", weeklyEarnings, app.config.Currency)
	}

	// 保存更新的状态
	if err := app.dbClient.UpdateBotStatus(status); err != nil {
		log.Printf("保存收益数据失败: %v", err)
	} else {
		log.Println("✅ 收益数据已更新到数据库")
	}
}
