package strategy

import (
	"testing"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
)

func TestSmartStrategy_CalculateOptimalAllocation(t *testing.T) {
	strategy := NewSmartStrategy(&config.Config{
		VolatilityThreshold: 0.002,
	})

	tests := []struct {
		name             string
		condition        *MarketCondition
		expectedHighHold float64
		expectedSpread   float64
	}{
		{
			name: "rising trend",
			condition: &MarketCondition{
				Trend:      "rising",
				Volatility: 0.001,
				RateRatio:  1.0,
			},
			expectedHighHold: 0.3,
			expectedSpread:   0.7,
		},
		{
			name: "falling trend",
			condition: &MarketCondition{
				Trend:      "falling",
				Volatility: 0.001,
				RateRatio:  1.0,
			},
			expectedHighHold: 0.7,
			expectedSpread:   0.3,
		},
		{
			name: "stable trend",
			condition: &MarketCondition{
				Trend:      "stable",
				Volatility: 0.001,
				RateRatio:  1.0,
			},
			expectedHighHold: 0.5,
			expectedSpread:   0.5,
		},
		{
			name: "high volatility",
			condition: &MarketCondition{
				Trend:      "stable",
				Volatility: 0.003, // 高於閾值
				RateRatio:  1.0,
			},
			expectedHighHold: 0.6, // 0.5 + 0.1
			expectedSpread:   0.4,
		},
		{
			name: "high rate ratio",
			condition: &MarketCondition{
				Trend:      "stable",
				Volatility: 0.001,
				RateRatio:  1.3, // 高於 1.2
			},
			expectedHighHold: 0.6, // 0.5 + 0.1
			expectedSpread:   0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			highHold, spread := strategy.calculateOptimalAllocation(tt.condition)

			if highHold != tt.expectedHighHold {
				t.Errorf("Expected highHold %f, got %f", tt.expectedHighHold, highHold)
			}
			if spread != tt.expectedSpread {
				t.Errorf("Expected spread %f, got %f", tt.expectedSpread, spread)
			}
		})
	}
}

func TestSmartStrategy_CalculateProgressiveRate(t *testing.T) {
	strategy := NewSmartStrategy(&config.Config{
		MaxRateMultiplier: 2.0,
		MinRateMultiplier: 0.8,
	})

	condition := &MarketCondition{
		Trend:      "stable",
		Volatility: 0.001,
		AvgRate:    0.0003,
	}

	tests := []struct {
		name         string
		fundingBook  []*bitfinex.FundingBookEntry
		minDailyRate float64
		orderIndex   int
		totalOrders  int
		expectMin    float64
		expectMax    float64
	}{
		{
			name:         "empty funding book",
			fundingBook:  []*bitfinex.FundingBookEntry{},
			minDailyRate: 0.0002,
			orderIndex:   0,
			totalOrders:  3,
			expectMin:    0.0002,
			expectMax:    0.0004, // 應該使用合成利率
		},
		{
			name: "with funding book data",
			fundingBook: []*bitfinex.FundingBookEntry{
				{Rate: 0.0003, Amount: 1000},
				{Rate: 0.0005, Amount: 2000},
				{Rate: 0.0007, Amount: 1500},
			},
			minDailyRate: 0.0002,
			orderIndex:   1,
			totalOrders:  3,
			expectMin:    0.0003,
			expectMax:    0.0007,
		},
		{
			name: "rates below minimum",
			fundingBook: []*bitfinex.FundingBookEntry{
				{Rate: 0.0001, Amount: 1000}, // 低於最小利率
			},
			minDailyRate: 0.0002,
			orderIndex:   0,
			totalOrders:  2,
			expectMin:    0.0002,
			expectMax:    0.0003,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := strategy.calculateProgressiveRate(tt.fundingBook, tt.minDailyRate, condition, tt.orderIndex, tt.totalOrders)

			if rate < tt.expectMin {
				t.Errorf("Rate %f is below expected minimum %f", rate, tt.expectMin)
			}
			if rate > tt.expectMax {
				t.Errorf("Rate %f is above expected maximum %f", rate, tt.expectMax)
			}
		})
	}
}

func TestSmartStrategy_CalculateSmartPeriod(t *testing.T) {
	cfg := &config.Config{
		ThirtyDayLendRateThreshold:    0.04,
		OneTwentyDayLendRateThreshold: 0.045,
		VolatilityThreshold:           0.002,
	}
	strategy := NewSmartStrategy(cfg)

	tests := []struct {
		name      string
		dailyRate float64
		condition *MarketCondition
		expected  int
	}{
		{
			name:      "low rate stable market",
			dailyRate: 0.0003, // 0.03% 日利率
			condition: &MarketCondition{
				Trend:      "stable",
				Volatility: 0.001,
				AvgRate:    0.0003,
			},
			expected: 2, // 默認期間
		},
		{
			name:      "medium rate triggers 30 day",
			dailyRate: 0.0004, // 0.04% 日利率
			condition: &MarketCondition{
				Trend:      "stable",
				Volatility: 0.001,
				AvgRate:    0.0003,
			},
			expected: 30,
		},
		{
			name:      "high rate triggers 120 day",
			dailyRate: 0.00045, // 0.045% 日利率
			condition: &MarketCondition{
				Trend:      "stable",
				Volatility: 0.001,
				AvgRate:    0.0003,
			},
			expected: 120,
		},
		{
			name:      "rising trend prefers shorter period",
			dailyRate: 0.00045, // 0.045% 日利率
			condition: &MarketCondition{
				Trend:      "rising",
				Volatility: 0.001,
				AvgRate:    0.0003,
			},
			expected: 30, // 120 -> 30 因為上升趨勢
		},
		{
			name:      "high volatility prefers shorter period",
			dailyRate: 0.00045, // 0.045% 日利率
			condition: &MarketCondition{
				Trend:      "stable",
				Volatility: 0.004, // 高波動
				AvgRate:    0.0003,
			},
			expected: 30, // 120 -> 30 因為高波動
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period := strategy.calculateSmartPeriod(tt.dailyRate, tt.condition)
			if period != tt.expected {
				t.Errorf("Expected period %d, got %d", tt.expected, period)
			}
		})
	}
}

func TestSmartStrategy_CalculateSmartOffers(t *testing.T) {
	cfg := &config.Config{
		MinLoan:                       150.0,
		MaxLoan:                       1000.0,
		SpreadLend:                    3,
		HighHoldAmount:                500.0,
		HighHoldOrders:                1,
		HighHoldRate:                  0.1,
		MinDailyLendRate:              0.02,
		ThirtyDayLendRateThreshold:    0.04,
		OneTwentyDayLendRateThreshold: 0.045,
		EnableSmartStrategy:           true,
		VolatilityThreshold:           0.002,
		MaxRateMultiplier:             2.0,
		MinRateMultiplier:             0.8,
	}
	strategy := NewSmartStrategy(cfg)

	fundingBook := []*bitfinex.FundingBookEntry{
		{Rate: 0.0003, Amount: 1000},
		{Rate: 0.0004, Amount: 2000},
		{Rate: 0.0005, Amount: 1500},
	}

	tests := []struct {
		name           string
		fundsAvailable float64
		expectedOffers int
	}{
		{
			name:           "insufficient funds",
			fundsAvailable: 100.0, // 低於 MinLoan
			expectedOffers: 0,
		},
		{
			name:           "sufficient funds for both strategies",
			fundsAvailable: 2000.0,
			expectedOffers: 4, // 1 high hold + 3 spread offers
		},
		{
			name:           "sufficient funds for spread only",
			fundsAvailable: 400.0, // 不足以做高額持有
			expectedOffers: 2,     // 只有 spread offers (400/3 約等於每筆133，少於150所以只能分2筆)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offers := strategy.CalculateSmartOffers(tt.fundsAvailable, fundingBook)
			if len(offers) != tt.expectedOffers {
				t.Errorf("Expected %d offers, got %d", tt.expectedOffers, len(offers))
			}

			// 驗證所有 offers 都有有效值
			for i, offer := range offers {
				if offer.Amount <= 0 {
					t.Errorf("Offer %d has invalid amount: %f", i, offer.Amount)
				}
				if offer.Rate <= 0 {
					t.Errorf("Offer %d has invalid rate: %f", i, offer.Rate)
				}
				if offer.Period <= 0 {
					t.Errorf("Offer %d has invalid period: %d", i, offer.Period)
				}
			}
		})
	}
}
