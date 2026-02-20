package strategy

import (
	"testing"
	"time"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
)

// TestSmartStrategyLLMCacheValidity 測試 LLM 緩存有效性
func TestSmartStrategyLLMCacheValidity(t *testing.T) {
	cfg := &config.Config{
		EnableLLMStrategy:   true,
		OpenAIAPIKey:        "test-key",
		LLMCacheHours:       3,
		LLMMaxRetries:       3,
		VolatilityThreshold: 0.002,
		MaxRateMultiplier:   2.0,
		MinRateMultiplier:   0.8,
	}
	currencyCfg := &config.CurrencyConfig{
		MinLoan: 150.0,
	}

	strategy := NewSmartStrategy(cfg, currencyCfg)

	// 測試空緩存
	if strategy.isLLMCacheValid() {
		t.Error("空緩存應該返回 false")
	}

	// 模擬設置緩存
	strategy.llmCache.prediction = &LLMPrediction{
		PredictedRateLow:  0.02,
		PredictedRateMid:  0.025,
		PredictedRateHigh: 0.03,
		Confidence:        0.75,
		Trend:             "stable",
	}
	strategy.llmCache.timestamp = time.Now()

	// 測試有效緩存
	if !strategy.isLLMCacheValid() {
		t.Error("剛設置的緩存應該有效")
	}

	// 測試過期緩存
	strategy.llmCache.timestamp = time.Now().Add(-4 * time.Hour)
	if strategy.isLLMCacheValid() {
		t.Error("4 小時前的緩存應該過期")
	}
}

// TestSmartStrategyLLMDisabled 測試 LLM 策略禁用時的行為
func TestSmartStrategyLLMDisabled(t *testing.T) {
	cfg := &config.Config{
		EnableLLMStrategy:        false, // 禁用 LLM 策略
		VolatilityThreshold:      0.002,
		MaxRateMultiplier:        2.0,
		MinRateMultiplier:        0.8,
		RateRangeIncreasePercent: 0.1,
	}
	currencyCfg := &config.CurrencyConfig{
		MinLoan: 150.0,
	}

	strategy := NewSmartStrategy(cfg, currencyCfg)

	condition := &MarketCondition{
		Trend:      "stable",
		Volatility: 0.001,
		AvgRate:    0.0003,
	}

	// 當 LLM 禁用時，應該使用合成利率
	rate := strategy.getLLMPredictedRate(
		[]*bitfinex.FundingBookEntry{},
		0.0002,
		condition,
		0,
		3,
	)

	// 應該返回合成利率（不為 0）
	if rate <= 0 {
		t.Errorf("禁用 LLM 時應該返回合成利率，但得到: %f", rate)
	}
}

// TestSmartStrategyLLMNoAPIKey 測試無 API Key 時的行為
func TestSmartStrategyLLMNoAPIKey(t *testing.T) {
	cfg := &config.Config{
		EnableLLMStrategy:        true,
		OpenAIAPIKey:             "", // 無 API Key
		VolatilityThreshold:      0.002,
		MaxRateMultiplier:        2.0,
		MinRateMultiplier:        0.8,
		RateRangeIncreasePercent: 0.1,
	}
	currencyCfg := &config.CurrencyConfig{
		MinLoan: 150.0,
	}

	strategy := NewSmartStrategy(cfg, currencyCfg)

	condition := &MarketCondition{
		Trend:      "stable",
		Volatility: 0.001,
		AvgRate:    0.0003,
	}

	// 當無 API Key 時，應該使用合成利率
	rate := strategy.getLLMPredictedRate(
		[]*bitfinex.FundingBookEntry{},
		0.0002,
		condition,
		0,
		3,
	)

	// 應該返回合成利率（不為 0）
	if rate <= 0 {
		t.Errorf("無 API Key 時應該返回合成利率，但得到: %f", rate)
	}
}

// TestSmartStrategyLLMWithCachedPrediction 測試使用緩存預測的行為
func TestSmartStrategyLLMWithCachedPrediction(t *testing.T) {
	cfg := &config.Config{
		EnableLLMStrategy:   true,
		OpenAIAPIKey:        "test-key",
		LLMCacheHours:       3,
		LLMMaxRetries:       3,
		VolatilityThreshold: 0.002,
		MaxRateMultiplier:   2.0,
		MinRateMultiplier:   0.8,
	}
	currencyCfg := &config.CurrencyConfig{
		MinLoan: 150.0,
	}

	strategy := NewSmartStrategy(cfg, currencyCfg)

	// 手動設置緩存預測
	strategy.llmCache.prediction = &LLMPrediction{
		PredictedRateLow:  0.02,  // 0.02%
		PredictedRateMid:  0.025, // 0.025%
		PredictedRateHigh: 0.03,  // 0.03%
		Confidence:        0.75,
		Trend:             "stable",
	}
	strategy.llmCache.timestamp = time.Now()

	condition := &MarketCondition{
		Trend:      "stable",
		Volatility: 0.001,
		AvgRate:    0.0003,
	}

	// 測試第一個訂單（應該接近 PredictedRateLow）
	rate0 := strategy.getLLMPredictedRate(
		[]*bitfinex.FundingBookEntry{},
		0.0001, // 最小利率
		condition,
		0, // 第一個訂單
		3, // 總共 3 個訂單
	)

	// 預期利率應該是 0.02% / 100 = 0.0002
	expectedRate0 := 0.02 / 100.0
	if rate0 < expectedRate0*0.99 || rate0 > expectedRate0*1.01 {
		t.Errorf("第一個訂單利率應該接近 %.6f，但得到: %.6f", expectedRate0, rate0)
	}

	// 測試最後一個訂單（應該接近 PredictedRateHigh）
	rate2 := strategy.getLLMPredictedRate(
		[]*bitfinex.FundingBookEntry{},
		0.0001,
		condition,
		2, // 最後一個訂單
		3,
	)

	// 預期利率應該是 0.03% / 100 = 0.0003
	expectedRate2 := 0.03 / 100.0
	if rate2 < expectedRate2*0.99 || rate2 > expectedRate2*1.01 {
		t.Errorf("最後一個訂單利率應該接近 %.6f，但得到: %.6f", expectedRate2, rate2)
	}
}

// TestSmartStrategyLLMMinRateEnforcement 測試最小利率強制執行時的遞增行為
func TestSmartStrategyLLMMinRateEnforcement(t *testing.T) {
	cfg := &config.Config{
		EnableLLMStrategy:        true,
		OpenAIAPIKey:             "test-key",
		LLMCacheHours:            3,
		LLMMaxRetries:            3,
		VolatilityThreshold:      0.002,
		MaxRateMultiplier:        2.0,
		MinRateMultiplier:        0.8,
		RateRangeIncreasePercent: 0.1, // 10% 遞增範圍
	}
	currencyCfg := &config.CurrencyConfig{
		MinLoan: 150.0,
	}

	strategy := NewSmartStrategy(cfg, currencyCfg)

	// 設置一個很低的預測利率（低於最小利率）
	strategy.llmCache.prediction = &LLMPrediction{
		PredictedRateLow:  0.001, // 0.001%
		PredictedRateMid:  0.002,
		PredictedRateHigh: 0.003, // 0.003% 仍低於最小利率 0.05%
		Confidence:        0.75,
		Trend:             "stable",
	}
	strategy.llmCache.timestamp = time.Now()

	condition := &MarketCondition{
		Trend:      "stable",
		Volatility: 0.001,
		AvgRate:    0.0003,
	}

	// 設置一個較高的最小利率
	minDailyRate := 0.0005 // 0.05%

	// 測試第一個訂單
	rate0 := strategy.getLLMPredictedRate(
		[]*bitfinex.FundingBookEntry{},
		minDailyRate,
		condition,
		0, // 第一個訂單
		3, // 總共 3 個訂單
	)

	// 第一個訂單應該等於最小利率
	if rate0 < minDailyRate*0.99 || rate0 > minDailyRate*1.01 {
		t.Errorf("第一個訂單利率應該接近最小利率 %.6f，但得到: %.6f", minDailyRate, rate0)
	}

	// 測試最後一個訂單
	rate2 := strategy.getLLMPredictedRate(
		[]*bitfinex.FundingBookEntry{},
		minDailyRate,
		condition,
		2, // 最後一個訂單
		3,
	)

	// 最後一個訂單應該高於第一個訂單（遞增）
	if rate2 <= rate0 {
		t.Errorf("最後一個訂單利率 %.6f 應該高於第一個訂單利率 %.6f", rate2, rate0)
	}

	// 最後一個訂單應該接近 minDailyRate * (1 + RateRangeIncreasePercent)
	expectedMaxRate := minDailyRate * (1.0 + cfg.RateRangeIncreasePercent)
	if rate2 < expectedMaxRate*0.99 || rate2 > expectedMaxRate*1.01 {
		t.Errorf("最後一個訂單利率應該接近 %.6f，但得到: %.6f", expectedMaxRate, rate2)
	}
}

// TestSmartStrategyCalculateProgressiveRateWithLLM 測試 calculateProgressiveRate 整合 LLM
func TestSmartStrategyCalculateProgressiveRateWithLLM(t *testing.T) {
	cfg := &config.Config{
		EnableLLMStrategy:        true,
		OpenAIAPIKey:             "test-key",
		LLMCacheHours:            3,
		LLMMaxRetries:            3,
		VolatilityThreshold:      0.002,
		MaxRateMultiplier:        2.0,
		MinRateMultiplier:        0.8,
		RateRangeIncreasePercent: 0.1,
	}
	currencyCfg := &config.CurrencyConfig{
		MinLoan: 150.0,
	}

	strategy := NewSmartStrategy(cfg, currencyCfg)

	// 設置緩存預測
	strategy.llmCache.prediction = &LLMPrediction{
		PredictedRateLow:  0.025,
		PredictedRateMid:  0.03,
		PredictedRateHigh: 0.035,
		Confidence:        0.8,
		Trend:             "rising",
	}
	strategy.llmCache.timestamp = time.Now()

	condition := &MarketCondition{
		Trend:      "stable",
		Volatility: 0.001,
		AvgRate:    0.0003,
	}

	// 測試空 fundingBook 時使用 LLM 預測
	rate := strategy.calculateProgressiveRate(
		[]*bitfinex.FundingBookEntry{}, // 空 fundingBook
		0.0002,
		condition,
		0,
		3,
	)

	// 應該返回 LLM 預測的利率
	expectedRate := 0.025 / 100.0 // PredictedRateLow
	if rate < expectedRate*0.99 || rate > expectedRate*1.01 {
		t.Errorf("空 fundingBook 時應該使用 LLM 預測利率，預期 %.6f，得到: %.6f", expectedRate, rate)
	}
}
