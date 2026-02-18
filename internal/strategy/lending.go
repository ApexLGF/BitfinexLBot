package strategy

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
	"github.com/ApexLGF/BitfinexLBot/internal/constants"
	"github.com/ApexLGF/BitfinexLBot/internal/rates"
)

// LendingBot 貸出機器人
type LendingBot struct {
	config         *config.Config
	currencyConfig *config.CurrencyConfig
	currency       string // 當前管理的幣種
	client         *bitfinex.Client
	rateConverter  *rates.Converter
	smartStrategy  *SmartStrategy
	notifyCallback func(string) error // Telegram 通知回調函數
}

// NewLendingBot 創建新的貸出機器人
func NewLendingBot(cfg *config.Config, currencyConfig *config.CurrencyConfig, currency string, client *bitfinex.Client) *LendingBot {
	return &LendingBot{
		config:         cfg,
		currencyConfig: currencyConfig,
		currency:       strings.ToUpper(currency),
		client:         client,
		rateConverter:  rates.NewConverter(),
		smartStrategy:  NewSmartStrategy(cfg, currencyConfig),
	}
}

// UpdateConfig 更新配置 - 用于热重载配置
func (lb *LendingBot) UpdateConfig(newConfig *config.Config) error {
	log.Printf("[Strategy] 更新配置...")

	// 更新主配置
	lb.config = newConfig

	// 更新币种配置
	if currencyConfig, ok := newConfig.Currencies[strings.ToLower(lb.currency)]; ok {
		lb.currencyConfig = currencyConfig
	}

	// 重新初始化智能策略（如果配置变化）
	lb.smartStrategy = NewSmartStrategy(newConfig, lb.currencyConfig)

	log.Printf("[Strategy] 配置更新完成")
	return nil
}

// GetFundingSymbol 獲取當前幣種的 funding symbol
func (lb *LendingBot) GetFundingSymbol() string {
	return constants.FundingSymbolPrefix + lb.currency
}

// GetMinDailyRateDecimal 獲取當前幣種的最小日利率（小數格式）
func (lb *LendingBot) GetMinDailyRateDecimal() float64 {
	return lb.currencyConfig.MinDailyLendRate / 100.0
}

// LoanOffer 代表一個貸出訂單
type LoanOffer struct {
	Amount float64
	Rate   float64 // 日利率（小數格式）
	Period int
}

// Execute 執行機器人主要邏輯
func (lb *LendingBot) Execute() error {
	log.Println("開始執行貸出機器人...")

	// 取消所有未完成訂單
	log.Println("取消所有未完成訂單...")
	hasPendingOrders, err := lb.cancelAllOffers()
	if err != nil {
		log.Printf("取消訂單失敗: %v", err)
		return err
	}

	// 等待訂單取消完成
	time.Sleep(constants.RetryDelay)

	// 獲取可用資金
	log.Println("取得可用額度...")
	fundsAvailable, err := lb.getAvailableFunds()
	if err != nil {
		log.Printf("取得餘額錯誤: %v", err)
		return err
	}
	log.Printf("Currency: %s  Available: %f", lb.currency, fundsAvailable)

	// 扣除保留金額
	if lb.currencyConfig.ReserveAmount > 0 {
		fundsAvailable = math.Max(0, fundsAvailable-lb.currencyConfig.ReserveAmount)
		log.Printf("扣除保留金額後可用: %f", fundsAvailable)
	}

	// 檢查可用資金
	if fundsAvailable < lb.currencyConfig.MinLoan {
		log.Println("可用資金小於最小貸出額，不進行操作")
		return nil
	}

	// 獲取市場數據
	fundingBook, err := lb.client.GetFundingBook(lb.GetFundingSymbol(), constants.MaxPriceLevels)
	if err != nil {
		log.Printf("取得 Funding Book 錯誤: %v", err)
		log.Println("使用fallback模式，僅使用最小利率策略")
		// 使用空的funding book，策略會自動使用最小利率
		fundingBook = []*bitfinex.FundingBookEntry{}
	}

	// 根據配置選擇策略
	var loanOffers []*LoanOffer
	if lb.config.EnableKlineStrategy {
		log.Println("使用K線策略計算貸出訂單...")
		loanOffers = lb.calculateKlineOffers(fundsAvailable)
	} else if lb.config.EnableSmartStrategy {
		log.Println("使用智能策略計算貸出訂單...")
		loanOffers = lb.smartStrategy.CalculateSmartOffers(fundsAvailable, fundingBook)
	} else {
		log.Println("使用傳統策略計算貸出訂單...")
		loanOffers = lb.calculateLoanOffers(fundsAvailable, fundingBook)
	}

	// 下單
	return lb.placeLoanOffers(loanOffers, hasPendingOrders)
}

// cancelAllOffers 取消所有未完成訂單
func (lb *LendingBot) cancelAllOffers() (bool, error) {
	offers, err := lb.client.GetFundingOffers(lb.GetFundingSymbol())
	if err != nil {
		return false, err
	}

	if len(offers) == 0 {
		log.Println("目前沒有未完成的訂單")
		return false, nil
	}

	for _, offer := range offers {
		if err := lb.client.CancelFundingOffer(offer.ID); err != nil {
			log.Printf("取消訂單失敗: %v", err)
		} else {
			log.Printf("成功取消訂單 ID: %d", offer.ID)
		}
	}

	return true, nil
}

// getAvailableFunds 獲取可用資金
func (lb *LendingBot) getAvailableFunds() (float64, error) {
	return lb.client.GetFundingBalance(lb.currency)
}

// calculateLoanOffers 計算貸出訂單
func (lb *LendingBot) calculateLoanOffers(fundsAvailable float64, fundingBook []*bitfinex.FundingBookEntry) []*LoanOffer {
	var loanOffers []*LoanOffer

	// 檢查可用資金
	if fundsAvailable < lb.currencyConfig.MinLoan {
		return loanOffers
	}

	splitFundsAvailable := fundsAvailable

	// 高額持有策略
	if lb.currencyConfig.HighHoldAmount > lb.currencyConfig.MinLoan {
		highHoldOffers := lb.calculateHighHoldOffers(&splitFundsAvailable)
		loanOffers = append(loanOffers, highHoldOffers...)
	}

	// 分散貸出策略
	if splitFundsAvailable >= lb.currencyConfig.MinLoan {
		spreadOffers := lb.calculateSpreadOffers(splitFundsAvailable, fundingBook)
		loanOffers = append(loanOffers, spreadOffers...)
	}

	return loanOffers
}

// calculateHighHoldOffers 計算高額持有訂單
func (lb *LendingBot) calculateHighHoldOffers(splitFundsAvailable *float64) []*LoanOffer {
	var offers []*LoanOffer

	ordersCount := lb.currencyConfig.HighHoldOrders
	if ordersCount <= 0 {
		ordersCount = 1
	}

	highHold := lb.currencyConfig.HighHoldAmount
	if lb.currencyConfig.MaxLoan > 0 && highHold > lb.currencyConfig.MaxLoan {
		highHold = lb.currencyConfig.MaxLoan
	}

	possibleOrders := int(*splitFundsAvailable / highHold)
	actualOrders := int(math.Min(float64(ordersCount), float64(possibleOrders)))

	for i := 0; i < actualOrders; i++ {
		if *splitFundsAvailable < highHold {
			break
		}

		offer := &LoanOffer{
			Amount: highHold,
			Rate:   lb.config.GetHighHoldRateDecimal(),
			Period: constants.Period120Days,
		}
		offers = append(offers, offer)
		*splitFundsAvailable -= highHold
	}

	return offers
}

// calculateSpreadOffers 計算分散貸出訂單
func (lb *LendingBot) calculateSpreadOffers(splitFundsAvailable float64, fundingBook []*bitfinex.FundingBookEntry) []*LoanOffer {
	var offers []*LoanOffer

	numSplits := lb.currencyConfig.SpreadLend
	if numSplits <= 0 || splitFundsAvailable < lb.currencyConfig.MinLoan {
		return offers
	}

	// 計算每筆金額
	amtEach := splitFundsAvailable / float64(numSplits)
	amtEach = float64(int64(amtEach*100)) / 100.0

	// 調整分割數
	for amtEach <= lb.currencyConfig.MinLoan && numSplits > 1 {
		numSplits--
		amtEach = splitFundsAvailable / float64(numSplits)
		amtEach = float64(int64(amtEach*100)) / 100.0
	}
	if numSplits <= 0 {
		return offers
	}

	// 計算利率遞增量
	gapClimb := (lb.currencyConfig.GapTop - lb.currencyConfig.GapBottom) / float64(numSplits)
	nextLend := lb.currencyConfig.GapBottom

	depthIndex := 0
	minDailyRate := lb.GetMinDailyRateDecimal()

	for numSplits > 0 {
		// 累計市場量至指定利率區間（僅在有funding book數據時）
		if len(fundingBook) > 0 {
			for float64(depthIndex) < nextLend && depthIndex < len(fundingBook)-1 {
				depthIndex++
			}
		}

		// 計算金額
		allocAmount := amtEach
		if lb.currencyConfig.MaxLoan > 0 && allocAmount > lb.currencyConfig.MaxLoan {
			allocAmount = lb.currencyConfig.MaxLoan
		}

		if allocAmount < lb.currencyConfig.MinLoan {
			break
		}

		// 計算利率
		var rate float64
		if len(fundingBook) > 0 && depthIndex < len(fundingBook) {
			marketRate := fundingBook[depthIndex].Rate
			if marketRate < minDailyRate {
				rate = minDailyRate
			} else {
				rate = marketRate
			}
		} else {
			// 無funding book數據時使用最小利率
			rate = minDailyRate
		}

		// 計算期間
		period := lb.calculatePeriod(rate)

		offer := &LoanOffer{
			Amount: allocAmount,
			Rate:   rate,
			Period: period,
		}
		offers = append(offers, offer)

		nextLend += gapClimb
		numSplits--
	}

	return offers
}

// calculatePeriod 根據利率計算貸出期間
func (lb *LendingBot) calculatePeriod(dailyRate float64) int {
	oneTwentyThreshold := lb.config.GetOneTwentyDayThresholdDecimal()
	thirtyThreshold := lb.config.GetThirtyDayThresholdDecimal()

	if lb.currencyConfig.OneTwentyDayLendRateThreshold > 0 && dailyRate >= oneTwentyThreshold {
		return constants.Period120Days
	} else if lb.currencyConfig.ThirtyDayLendRateThreshold > 0 && dailyRate >= thirtyThreshold {
		return constants.Period30Days
	} else {
		return constants.DefaultPeriodDays
	}
}

// placeLoanOffers 下單
func (lb *LendingBot) placeLoanOffers(loanOffers []*LoanOffer, hasPendingOrders bool) error {
	orderCount := 0
	fundingSymbol := lb.GetFundingSymbol()

	for _, offer := range loanOffers {
		if lb.config.OrderLimit != 0 && orderCount >= lb.config.OrderLimit {
			break
		}

		rate := offer.Rate
		if !hasPendingOrders {
			// 添加利率加成
			rate += lb.rateConverter.PercentageToDecimal(lb.currencyConfig.RateBonus)
		}

		// 驗證利率
		if !lb.rateConverter.ValidateDailyRate(rate) {
			log.Printf("跳過無效利率: %.6f", rate)
			continue
		}

		if lb.config.TestMode {
			// 測試模式：只記錄不真的下單
			log.Printf("🧪 [測試模式] 模擬下單 => Rate: %.6f%%, Amount: %.4f, Period: %d",
				lb.rateConverter.DecimalToPercentage(rate), offer.Amount, offer.Period)
			orderCount++
		} else {
			// 正式模式：真的下單
			log.Printf("下單 => Rate: %.6f%%, Amount: %.4f, Period: %d",
				lb.rateConverter.DecimalToPercentage(rate), offer.Amount, offer.Period)

			err := lb.client.SubmitFundingOffer(fundingSymbol, offer.Amount, rate, offer.Period, false)
			if err != nil {
				log.Printf("下訂單失敗: %v", err)
			} else {
				orderCount++
			}
		}
	}

	return nil
}

// CheckRateThreshold 檢查利率是否超過閾值（基於5分鐘K線最近12根高點）
func (lb *LendingBot) CheckRateThreshold() (bool, float64, error) {
	// 獲取5分鐘K線數據（12根，相當於1小時）
	candles, err := lb.client.GetFundingCandles(
		lb.GetFundingSymbol(),
		"5m",
		12,
	)
	if err != nil {
		return false, 0, err
	}

	// 找到最近12根K線中的最高利率
	highestRate := lb.findMaxRate(candles)
	percentageRate := lb.rateConverter.DecimalDailyToPercentageDaily(highestRate)
	exceeded := percentageRate > lb.currencyConfig.NotifyRateThreshold

	log.Printf("K線閾值檢查 - 最近12根5分鐘K線最高利率: %.4f%%, 閾值: %.4f%%, 超過: %v",
		percentageRate, lb.currencyConfig.NotifyRateThreshold, exceeded)

	return exceeded, percentageRate, nil
}

// SetNotifyCallback 設置 Telegram 通知回調函數
func (lb *LendingBot) SetNotifyCallback(callback func(string) error) {
	lb.notifyCallback = callback
}

// CheckNewLendingCredits 檢查新的借貸訂單並發送通知
func (lb *LendingBot) CheckNewLendingCredits() error {
	log.Println("檢查新的借貸訂單...")

	// 獲取當前活躍的借貸訂單
	credits, err := lb.client.GetFundingCredits(lb.GetFundingSymbol())
	if err != nil {
		log.Printf("獲取借貸訂單失敗: %v", err)
		return err
	}

	if len(credits) == 0 {
		log.Println("目前沒有活躍的借貸訂單")
		return nil
	}

	// 獲取當前時間戳（毫秒）
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)

	// 如果這是第一次檢查（LastLendingCheckTime 為 0），檢查是否有最近60分鐘內的訂單
	if lb.config.LastLendingCheckTime == 0 {
		log.Printf("首次檢查，發現 %d 個現有的借貸訂單", len(credits))

		// 檢查是否有最近60分鐘內的訂單
		recentThreshold := currentTime - (60 * 60 * 1000) // 60分鐘前的時間戳
		var recentCredits []*bitfinex.FundingCredit
		for _, credit := range credits {
			if credit.MTSOpened > recentThreshold {
				recentCredits = append(recentCredits, credit)
			}
		}

		// 初始化時間戳
		lb.config.LastLendingCheckTime = currentTime

		// 如果有最近60分鐘內的訂單，發送通知
		if len(recentCredits) > 0 {
			log.Printf("發現 %d 個最近60分鐘內的借貸訂單，發送通知", len(recentCredits))
			return lb.sendLendingNotification(recentCredits)
		}

		log.Println("沒有最近60分鐘內的新訂單，初始化完成")
		return nil
	}

	// 檢查是否有新的借貸訂單（開始時間大於上次檢查時間）
	var newCredits []*bitfinex.FundingCredit
	for _, credit := range credits {
		if credit.MTSOpened > lb.config.LastLendingCheckTime {
			newCredits = append(newCredits, credit)
		}
	}

	// 更新最後檢查時間
	lb.config.LastLendingCheckTime = currentTime

	// 如果有新的借貸訂單，發送通知
	if len(newCredits) > 0 {
		log.Printf("發現 %d 個新的借貸訂單", len(newCredits))
		return lb.sendLendingNotification(newCredits)
	}

	log.Println("沒有新的借貸訂單")
	return nil
}

// sendLendingNotification 發送借貸訂單通知
func (lb *LendingBot) sendLendingNotification(credits []*bitfinex.FundingCredit) error {
	if lb.notifyCallback == nil {
		log.Println("Telegram 通知回調未設置，跳過通知")
		return nil
	}

	message := "💰 新的借貸訂單通知\n\n"

	// 先計算所有訂單的統計信息
	totalAmount := 0.0
	totalEarnings := 0.0

	for _, credit := range credits {
		dailyEarnings := credit.Amount * credit.Rate
		periodEarnings := dailyEarnings * float64(credit.Period)
		totalAmount += credit.Amount
		totalEarnings += periodEarnings
	}

	// 顯示詳細信息（最多顯示配置數量的訂單）
	for i, credit := range credits {
		if i >= constants.MaxDisplayOrders {
			remaining := len(credits) - constants.MaxDisplayOrders
			message += fmt.Sprintf("... 還有 %d 個訂單\n", remaining)
			break
		}

		// 計算預期收益（日利率 * 金額 * 期間）
		dailyEarnings := credit.Amount * credit.Rate
		periodEarnings := dailyEarnings * float64(credit.Period)

		// 格式化開始時間
		openTime := time.Unix(credit.MTSOpened/1000, 0)

		message += fmt.Sprintf("📊 訂單 #%d\n", i+1)
		message += fmt.Sprintf("💵 金額: %.2f %s\n", credit.Amount, lb.config.Currency)
		message += fmt.Sprintf("📈 日利率: %.4f%%\n", lb.rateConverter.DecimalToPercentage(credit.Rate))
		message += fmt.Sprintf("📈 年利率: %.4f%%\n", lb.rateConverter.DecimalToPercentage(credit.Rate)*constants.DaysPerYear)
		message += fmt.Sprintf("⏰ 期間: %d 天\n", credit.Period)
		message += fmt.Sprintf("💰 預期收益: %.4f %s\n", periodEarnings, lb.config.Currency)
		message += fmt.Sprintf("🕐 開始時間: %s\n", openTime.Format("2006-01-02 15:04:05"))
		message += "\n"
	}

	// 添加統計信息
	message += "📊 統計信息:\n"
	message += fmt.Sprintf("📦 總數量: %d 個訂單\n", len(credits))
	message += fmt.Sprintf("💵 總金額: %.2f %s\n", totalAmount, lb.config.Currency)
	message += fmt.Sprintf("💰 總預期收益: %.4f %s\n", totalEarnings, lb.config.Currency)

	// 嘗試發送通知，如果失敗（例如 Telegram 未認證）只記錄日誌但不返回錯誤
	if err := lb.notifyCallback(message); err != nil {
		log.Printf("發送借貸訂單通知失敗: %v", err)
		log.Println("新借貸訂單通知內容:")
		log.Println(message)
		return nil // 不返回錯誤，避免影響主程序執行
	}

	log.Println("借貸訂單通知發送成功")
	return nil
}

// GetActiveLendingCredits 獲取活躍借貸訂單（供 Telegram 指令使用）
func (lb *LendingBot) GetActiveLendingCredits() ([]*bitfinex.FundingCredit, error) {
	return lb.client.GetFundingCredits(lb.GetFundingSymbol())
}

// calculateKlineOffers 基於K線數據計算貸出訂單
func (lb *LendingBot) calculateKlineOffers(fundsAvailable float64) []*LoanOffer {
	var loanOffers []*LoanOffer

	// 檢查可用資金
	if fundsAvailable < lb.currencyConfig.MinLoan {
		return loanOffers
	}

	// 獲取K線數據
	candles, _ := lb.client.GetFundingCandles(
		lb.GetFundingSymbol(),
		lb.config.KlineTimeFrame,
		lb.config.KlinePeriod,
	)

	// 找到最近期間內的最高利率
	highestRate := lb.findHighestRateFromCandles(candles)
	log.Printf("K線數據分析：最高利率 %.6f%%", lb.rateConverter.DecimalToPercentage(highestRate))

	// 計算目標利率（最高利率 + 加成）
	spreadMultiplier := 1.0 + (lb.config.KlineSpreadPercent / 100.0)
	targetRate := highestRate * spreadMultiplier

	// 確保不低於最小利率
	minDailyRate := lb.GetMinDailyRateDecimal()
	if targetRate < minDailyRate {
		targetRate = minDailyRate
		log.Printf("目標利率低於最小利率，使用最小利率: %.6f%%", lb.rateConverter.DecimalToPercentage(targetRate))
	}

	log.Printf("K線策略目標利率: %.6f%% (加成: %.1f%%)",
		lb.rateConverter.DecimalToPercentage(targetRate),
		lb.config.KlineSpreadPercent)

	splitFundsAvailable := fundsAvailable

	// 高額持有策略
	if lb.currencyConfig.HighHoldAmount > lb.currencyConfig.MinLoan {
		highHoldOffers := lb.calculateHighHoldOffers(&splitFundsAvailable)
		loanOffers = append(loanOffers, highHoldOffers...)
	}

	// 使用目標利率創建分散訂單
	if splitFundsAvailable >= lb.currencyConfig.MinLoan {
		klineOffers := lb.calculateKlineSpreadOffers(splitFundsAvailable, targetRate)
		loanOffers = append(loanOffers, klineOffers...)
	}

	return loanOffers
}

// findHighestRateFromCandles 從K線數據中找到最高利率
func (lb *LendingBot) findHighestRateFromCandles(candles []*bitfinex.Candle) float64 {
	if len(candles) == 0 {
		return lb.GetMinDailyRateDecimal()
	}

	// 根據配置選擇平滑方法
	switch lb.config.KlineSmoothMethod {
	case "max":
		return lb.findMaxRate(candles)
	case "sma":
		return lb.calculateSMA(candles)
	case "ema":
		return lb.calculateEMAHigh(candles)
	case "hla":
		return lb.calculateHighLowAverage(candles)
	case "p90":
		return lb.calculate90Percentile(candles)
	default:
		log.Printf("未知的平滑方法: %s，使用預設的 EMA", lb.config.KlineSmoothMethod)
		return lb.calculateEMAHigh(candles)
	}
}

// findMaxRate 找到最高利率（原始方法）
func (lb *LendingBot) findMaxRate(candles []*bitfinex.Candle) float64 {
	highestRate := candles[0].High
	for _, candle := range candles {
		if candle.High > highestRate {
			highestRate = candle.High
		}
	}
	return highestRate
}

// calculateSMA 計算收盤價的簡單移動平均
func (lb *LendingBot) calculateSMA(candles []*bitfinex.Candle) float64 {
	if len(candles) == 0 {
		return lb.GetMinDailyRateDecimal()
	}

	sum := 0.0
	for _, candle := range candles {
		sum += candle.Close
	}
	return sum / float64(len(candles))
}

// calculateEMAHigh 計算高點的指數移動平均
func (lb *LendingBot) calculateEMAHigh(candles []*bitfinex.Candle) float64 {
	if len(candles) == 0 {
		return lb.GetMinDailyRateDecimal()
	}

	// EMA 係數，期間越長係數越小
	alpha := 2.0 / (float64(len(candles)) + 1.0)
	ema := candles[0].High

	for i := 1; i < len(candles); i++ {
		ema = alpha*candles[i].High + (1-alpha)*ema
	}

	return ema
}

// calculateHighLowAverage 計算高低點平均
func (lb *LendingBot) calculateHighLowAverage(candles []*bitfinex.Candle) float64 {
	if len(candles) == 0 {
		return lb.GetMinDailyRateDecimal()
	}

	sumHigh := 0.0
	sumLow := 0.0
	for _, candle := range candles {
		sumHigh += candle.High
		sumLow += candle.Low
	}

	avgHigh := sumHigh / float64(len(candles))
	avgLow := sumLow / float64(len(candles))

	// 取高低點平均的平均（偏向高點一些）
	return (avgHigh + avgLow) / 2.0
}

// calculate90Percentile 計算90百分位數
func (lb *LendingBot) calculate90Percentile(candles []*bitfinex.Candle) float64 {
	if len(candles) == 0 {
		return lb.GetMinDailyRateDecimal()
	}

	// 收集所有高點
	highs := make([]float64, len(candles))
	for i, candle := range candles {
		highs[i] = candle.High
	}

	// 簡單排序
	for i := 0; i < len(highs); i++ {
		for j := i + 1; j < len(highs); j++ {
			if highs[i] > highs[j] {
				highs[i], highs[j] = highs[j], highs[i]
			}
		}
	}

	// 計算90百分位數的索引
	index := int(float64(len(highs)) * 0.9)
	if index >= len(highs) {
		index = len(highs) - 1
	}

	return highs[index]
}

// calculateKlineSpreadOffers 基於K線目標利率計算分散訂單
func (lb *LendingBot) calculateKlineSpreadOffers(fundsAvailable float64, targetRate float64) []*LoanOffer {
	var offers []*LoanOffer

	numSplits := lb.currencyConfig.SpreadLend
	if numSplits <= 0 || fundsAvailable < lb.currencyConfig.MinLoan {
		return offers
	}

	// 計算每筆金額
	amtEach := fundsAvailable / float64(numSplits)
	amtEach = float64(int64(amtEach*100)) / 100.0

	// 調整分割數
	for amtEach <= lb.currencyConfig.MinLoan && numSplits > 1 {
		numSplits--
		amtEach = fundsAvailable / float64(numSplits)
		amtEach = float64(int64(amtEach*100)) / 100.0
	}
	if numSplits <= 0 {
		return offers
	}

	// 創建訂單，使用目標利率為基準，微調以分散風險
	for i := 0; i < numSplits; i++ {
		// 計算金額
		allocAmount := amtEach
		if lb.currencyConfig.MaxLoan > 0 && allocAmount > lb.currencyConfig.MaxLoan {
			allocAmount = lb.currencyConfig.MaxLoan
		}

		if allocAmount < lb.currencyConfig.MinLoan {
			break
		}

		rate := targetRate * (1 + (float64(i) * lb.config.RateRangeIncreasePercent))

		// 確保利率不低於最小利率
		minDailyRate := lb.GetMinDailyRateDecimal()
		if rate < minDailyRate {
			rate = minDailyRate
		}

		// 計算期間
		period := lb.calculatePeriod(rate)

		offer := &LoanOffer{
			Amount: allocAmount,
			Rate:   rate,
			Period: period,
		}
		offers = append(offers, offer)
	}

	return offers
}
