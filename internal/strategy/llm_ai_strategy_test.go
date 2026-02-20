package strategy

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
)

// TestLLMAIStrategy 測試 LLM AI 策略的三種方案
func TestLLMAIStrategy(t *testing.T) {
	// 優先從配置文件讀取，其次從環境變量
	cfg, err := loadTestConfig()
	if err != nil {
		t.Skipf("跳過測試：無法加載配置: %v", err)
	}

	if cfg.OpenAIAPIKey == "" {
		t.Skip("跳過測試：未配置 OPENAI_API_KEY")
	}

	// 創建市場分析器並填充模擬歷史數��
	analyzer := NewMarketAnalyzer()
	populateTestRateHistory(analyzer)

	// 創建模擬的 fundingBook 數據
	fundingBook := createTestFundingBook()

	// 創建 LLM 策略
	llmStrategy := NewLLMAIStrategy(cfg, analyzer)

	// 測試三種策略
	strategies := []LLMStrategyType{
		StrategySimple,
		StrategyDetailed,
		StrategyChainOfThought,
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("LLM AI 策略預測測試")
	fmt.Printf("使用模型: %s\n", cfg.OpenAIModel)
	fmt.Println(strings.Repeat("=", 60) + "\n")

	for _, strategyType := range strategies {
		t.Run(GetStrategyName(strategyType), func(t *testing.T) {
			fmt.Printf("\n--- 測試策略 %d: %s ---\n", strategyType, GetStrategyName(strategyType))

			startTime := time.Now()
			prediction, err := llmStrategy.Predict(strategyType, fundingBook)
			elapsed := time.Since(startTime)

			if err != nil {
				t.Errorf("策略 %d 預測失敗: %v", strategyType, err)
				return
			}

			// 打印預測結果
			fmt.Printf("\n預測結果:\n")
			fmt.Printf("  預測利率範圍: %.4f%% - %.4f%% (最可能: %.4f%%)\n",
				prediction.PredictedRateLow,
				prediction.PredictedRateHigh,
				prediction.PredictedRateMid)
			fmt.Printf("  置信度: %.1f%%\n", prediction.Confidence*100)
			fmt.Printf("  趨勢判斷: %s\n", prediction.Trend)
			fmt.Printf("  推理說明: %s\n", prediction.Reasoning)
			fmt.Printf("  耗時: %v\n", elapsed)
			fmt.Println()

			// 驗證預測結果的合理性
			if prediction.PredictedRateMid <= 0 {
				t.Errorf("預測利率無效: %.4f", prediction.PredictedRateMid)
			}
			if prediction.Confidence < 0 || prediction.Confidence > 1 {
				t.Errorf("置信度超出範圍: %.2f", prediction.Confidence)
			}
		})
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("測試完成")
	fmt.Println(strings.Repeat("=", 60))
}

// TestLLMAIStrategyWithRealData 使用真實 Bitfinex 數據測試
func TestLLMAIStrategyWithRealData(t *testing.T) {
	cfg, err := loadTestConfig()
	if err != nil {
		t.Skipf("跳過測試：無法加載配置: %v", err)
	}

	if cfg.OpenAIAPIKey == "" {
		t.Skip("跳過測試：未配置 OPENAI_API_KEY")
	}

	// 創建 Bitfinex 客戶端獲取真實數據
	client := bitfinex.NewClient("", "") // 公開 API 不需要認證

	// 獲取真實的 fundingBook
	fundingBook, err := client.GetFundingBook("fUSD", 50)
	if err != nil {
		t.Skipf("跳過測試：無法獲取 Bitfinex 數據: %v", err)
	}

	fmt.Printf("\n獲取到 %d 層訂單簿數據\n", len(fundingBook))
	if len(fundingBook) > 0 {
		fmt.Printf("當前最佳利率: %.4f%%/天\n", fundingBook[0].Rate*100)
	}

	// 創建市場分析器
	analyzer := NewMarketAnalyzer()

	// 使用真實利率填充歷史數據（模擬過去 12 小時）
	if len(fundingBook) > 0 {
		baseRate := fundingBook[0].Rate
		populateHistoryFromBaseRate(analyzer, baseRate)
	}

	// 創建 LLM 策略
	llmStrategy := NewLLMAIStrategy(cfg, analyzer)

	// 只測試方案一（節省 API 調用）
	fmt.Printf("\n使用真實數據測試 Simple 策略 (模型: %s)...\n", cfg.OpenAIModel)
	prediction, err := llmStrategy.Predict(StrategySimple, fundingBook)
	if err != nil {
		t.Fatalf("預測失敗: %v", err)
	}

	fmt.Printf("\n真實數據預測結果:\n")
	fmt.Printf("  當前市場利率: %.4f%%\n", fundingBook[0].Rate*100)
	fmt.Printf("  預測 4 小時後: %.4f%% - %.4f%%\n",
		prediction.PredictedRateLow,
		prediction.PredictedRateHigh)
	fmt.Printf("  趨勢: %s, 置信度: %.0f%%\n",
		prediction.Trend, prediction.Confidence*100)
}

// populateTestRateHistory 填充測試用的歷史利率數據
func populateTestRateHistory(analyzer *MarketAnalyzer) {
	// 模擬過去 12 小時的利率數據（48 個點，每 15 分鐘一個）
	baseRate := 0.0003 // 0.03%/天
	now := time.Now()

	for i := 47; i >= 0; i-- {
		// 添加一些波動
		variation := float64(i%5-2) * 0.00002 // ±0.002%
		rate := baseRate + variation

		timestamp := now.Add(-time.Duration(i*15) * time.Minute)
		volume := 10000.0 + float64(i*100)

		analyzer.AddRateSnapshotWithTime(rate, volume, timestamp)
	}
}

// populateHistoryFromBaseRate 基於基準利率填充歷史數據
func populateHistoryFromBaseRate(analyzer *MarketAnalyzer, baseRate float64) {
	now := time.Now()

	for i := 47; i >= 0; i-- {
		// 模擬利率波動（±10%）
		variation := (float64(i%10) - 5) / 50 * baseRate
		rate := baseRate + variation

		timestamp := now.Add(-time.Duration(i*15) * time.Minute)
		volume := 50000.0 + float64(i*1000)

		analyzer.AddRateSnapshotWithTime(rate, volume, timestamp)
	}
}

// createTestFundingBook 創建測試用的 fundingBook 數據
func createTestFundingBook() []*bitfinex.FundingBookEntry {
	book := make([]*bitfinex.FundingBookEntry, 30)
	baseRate := 0.0003 // 0.03%/天

	for i := 0; i < 30; i++ {
		book[i] = &bitfinex.FundingBookEntry{
			Rate:   baseRate + float64(i)*0.00001, // 每層增加 0.001%
			Amount: 100000.0 - float64(i)*2000,    // 金額遞減
			Period: 2,
			Count:  10 - i/3,
		}
	}

	return book
}

// getEnvOrDefault 獲取環境變量，如果不存在則返回默認值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadTestConfig 加載測試配置（優先從配置文件，其次從環境變量）
func loadTestConfig() (*config.Config, error) {
	// 嘗試從項目根目錄加載配置文件
	configPaths := []string{
		"../../config.yaml",
		"../../../config.yaml",
		"config.yaml",
	}

	for _, path := range configPaths {
		cfg, err := config.LoadConfig(path)
		if err == nil {
			fmt.Printf("從配置文件加載: %s\n", path)
			return cfg, nil
		}
	}

	// 如果配置文件不存在，從環境變量創建配置
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("未找到配置文件且未設置 OPENAI_API_KEY 環境變量")
	}

	return &config.Config{
		OpenAIAPIKey:       apiKey,
		OpenAIModel:        getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
		OpenAIBaseURL:      getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		LLMDefaultStrategy: 1,
		LLMTimeoutSeconds:  60,
	}, nil
}

// TestLLMPromptGeneration 測試提示詞生成（不調用 API）
func TestLLMPromptGeneration(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey:       "test-key",
		OpenAIModel:        "gpt-4o",
		OpenAIBaseURL:      "https://api.openai.com/v1",
		LLMDefaultStrategy: 1,
		LLMTimeoutSeconds:  30,
	}

	analyzer := NewMarketAnalyzer()
	populateTestRateHistory(analyzer)

	fundingBook := createTestFundingBook()
	condition := analyzer.AnalyzeMarket(fundingBook)

	llmStrategy := NewLLMAIStrategy(cfg, analyzer)

	// 測試三種提示詞生成
	t.Run("SimplePrompt", func(t *testing.T) {
		prompt := llmStrategy.buildSimplePrompt(condition)
		fmt.Printf("\n=== Simple Prompt (%d chars) ===\n%s\n", len(prompt), prompt)

		if len(prompt) == 0 {
			t.Error("Simple prompt 為空")
		}
	})

	t.Run("DetailedPrompt", func(t *testing.T) {
		prompt := llmStrategy.buildDetailedPrompt(condition, fundingBook)
		fmt.Printf("\n=== Detailed Prompt (%d chars) ===\n%s\n", len(prompt), prompt)

		if len(prompt) == 0 {
			t.Error("Detailed prompt 為空")
		}
	})

	t.Run("ChainOfThoughtPrompt", func(t *testing.T) {
		prompt := llmStrategy.buildChainOfThoughtPrompt(condition)
		fmt.Printf("\n=== ChainOfThought Prompt (%d chars) ===\n%s\n", len(prompt), prompt)

		if len(prompt) == 0 {
			t.Error("ChainOfThought prompt 為空")
		}
	})
}
