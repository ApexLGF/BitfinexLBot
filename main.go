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

	"github.com/ApexLGF/BitfinexLBot/internal/api"
	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
	"github.com/ApexLGF/BitfinexLBot/internal/constants"
	"github.com/ApexLGF/BitfinexLBot/internal/currency"
	"github.com/ApexLGF/BitfinexLBot/internal/rates"
)

// Application 應用程式主結構
type Application struct {
	config          *config.Config
	bfxClient       *bitfinex.Client
	apiServer       *api.Server
	apiHandler      *api.Handler
	currencyManager *currency.CurrencyManager // 替代 lendingBot
	rateConverter   *rates.Converter

	// 併發控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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

	// 創建幣種管理器（替代單一 LendingBot）
	currencyManager := currency.NewCurrencyManager(cfg, bfxClient)

	// 創建利率轉換器
	rateConverter := rates.NewConverter()

	// 創建 context 和 cancel 函數
	ctx, cancel := context.WithCancel(context.Background())

	// 創建API處理器和服務器
	var apiHandler *api.Handler
	var apiServer *api.Server

	if cfg.APIEnabled {
		apiHandler = api.NewHandler(cfg, bfxClient, currencyManager, configPath)
		apiServer = api.NewServer(cfg.APIPort, cfg.APIHost, apiHandler)
	}

	app := &Application{
		config:          cfg,
		bfxClient:       bfxClient,
		apiServer:       apiServer,
		apiHandler:      apiHandler,
		currencyManager: currencyManager,
		rateConverter:   rateConverter,
		ctx:             ctx,
		cancel:          cancel,
	}

	// 設置借貸機器人的通知回調（如果有API處理器）
	if apiHandler != nil {
		// 設置初始狀態
		apiHandler.SetRunning(true)
	}

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
	// 啟動 API 服務器（如果啟用）
	if app.apiServer != nil {
		app.wg.Add(1)
		go app.runWorker("APIServer", func() {
			defer app.wg.Done()
			if err := app.apiServer.Start(); err != nil {
				log.Printf("API服務器啟動失敗: %v", err)
			}
		})
	}

	// 啟動每小時利率檢查
	app.wg.Add(1)
	go app.runWorker("HourlyRateCheck", func() {
		defer app.wg.Done()
		app.scheduleHourlyRateCheck()
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

	// 先關閉API服務器
	if app.apiServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		
		if err := app.apiServer.Stop(shutdownCtx); err != nil {
			log.Printf("API服務器關閉失敗: %v", err)
		} else {
			log.Println("API服務器已關閉")
		}
	}

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

	log.Println("應用程式已關閉")
	return nil
}

// scheduleMainTask 調度主要任務
func (app *Application) scheduleMainTask() {
	// 先執行第一次
	app.executeMainTask()

	log.Printf("[MAIN] 設置定時器間隔: %d 分鐘", app.config.MinutesRun)
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
	// 執行所有啟用幣種的策略
	if err := app.currencyManager.ExecuteAll(); err != nil {
		log.Printf("執行貸出策略失敗: %v", err)
	}

	// 更新API處理器的狀態
	if app.apiHandler != nil {
		app.apiHandler.UpdateNextRun()
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

	// 檢查所有幣種的利率閾值
	for _, currency := range app.currencyManager.GetEnabledCurrencies() {
		bot, ok := app.currencyManager.GetBot(currency)
		if !ok {
			continue
		}

		exceeded, percentageRate, err := bot.CheckRateThreshold()
		if err != nil {
			log.Printf("[%s] 取得利率數據失敗: %v", currency, err)
			continue
		}

		// 獲取該幣種的通知閾值
		currencyConfig := app.config.Currencies[currency]
		threshold := currencyConfig.NotifyRateThreshold

		log.Printf("[%s] 最近1小時最高利率: %.4f%%, 閾值: %.4f%%", currency, percentageRate, threshold)

		if exceeded {
			message := fmt.Sprintf("⚠️ [%s] 定時檢查提醒: 最近1小時最高利率 %.4f%% 已超過閾值 %.4f%%\n\n📊 檢查方式: 5分鐘K線最近12根高點分析",
				currency, percentageRate, threshold)

			log.Printf("利率提醒: %s", message)
			// 通過API處理器記錄日志或發送通知
			if app.apiHandler != nil {
				// TODO: 實現通知功能
			}
		} else {
			log.Printf("[%s] 最近1小時最高利率低於閾值，無需發送通知", currency)
		}
	}
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
	// 檢查所有幣種的借貸訂單
	for _, currency := range app.currencyManager.GetEnabledCurrencies() {
		bot, ok := app.currencyManager.GetBot(currency)
		if !ok {
			continue
		}

		if err := bot.CheckNewLendingCredits(); err != nil {
			log.Printf("[%s] 檢查借貸訂單失敗: %v", currency, err)
		}
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
