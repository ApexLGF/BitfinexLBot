package api

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/constants"
	"github.com/ApexLGF/BitfinexLBot/internal/currency"
)

// DataCache Web API 数据缓存，避免与策略执行争抢 apiMutex
type DataCache struct {
	mu sync.RWMutex

	// 缓存数据
	wallets    []*bitfinex.Wallet
	offers     map[string][]*bitfinex.FundingOffer  // key: 大写币种
	credits    map[string][]*bitfinex.FundingCredit  // key: 大写币种
	balances   map[string]float64                    // key: 大写币种, funding 钱包总余额 (Balance)
	available  map[string]float64                    // key: 大写币种, funding 钱包可用余额 (Available)
	frrRates   map[string]float64                    // key: 大写币种
	earnings   map[string]EarningsData               // key: 大写币种

	// 缓存时间戳
	walletsUpdated  time.Time
	offersUpdated   time.Time
	creditsUpdated  time.Time
	balancesUpdated time.Time
	frrUpdated      time.Time
	earningsUpdated time.Time

	// 依赖
	client          *bitfinex.Client
	currencyManager *currency.CurrencyManager
	handler         *Handler // 用于调用 earnings 处理逻辑

	// 缓存有效期
	cacheTTL time.Duration

	// 后台刷新控制
	stopCh chan struct{}
}

// NewDataCache 创建数据缓存
func NewDataCache(client *bitfinex.Client, cm *currency.CurrencyManager, handler *Handler, ttl time.Duration) *DataCache {
	return &DataCache{
		client:          client,
		currencyManager: cm,
		handler:         handler,
		cacheTTL:        ttl,
		offers:          make(map[string][]*bitfinex.FundingOffer),
		credits:         make(map[string][]*bitfinex.FundingCredit),
		balances:        make(map[string]float64),
		available:       make(map[string]float64),
		frrRates:        make(map[string]float64),
		earnings:        make(map[string]EarningsData),
		stopCh:          make(chan struct{}),
	}
}

// StartBackgroundRefresh 启动后台定期刷新
func (dc *DataCache) StartBackgroundRefresh(interval time.Duration) {
	// 首次立即刷新
	dc.RefreshAll()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dc.RefreshAll()
			case <-dc.stopCh:
				return
			}
		}
	}()
}

// Stop 停止后台刷新
func (dc *DataCache) Stop() {
	close(dc.stopCh)
}

// RefreshAll 刷新所有缓存数据
func (dc *DataCache) RefreshAll() {
	log.Println("[Cache] 开始刷新缓存数据...")
	start := time.Now()

	dc.refreshWallets()
	dc.refreshBalances()
	dc.refreshOffers()
	dc.refreshCredits()
	dc.refreshFRRRates()
	dc.refreshEarnings()

	log.Printf("[Cache] 缓存刷新完成，耗时: %v", time.Since(start))
}

func (dc *DataCache) refreshWallets() {
	wallets, err := dc.client.GetWallets()
	if err != nil {
		log.Printf("[Cache] 刷新钱包数据失败: %v", err)
		return
	}
	dc.mu.Lock()
	dc.wallets = wallets
	dc.walletsUpdated = time.Now()
	dc.mu.Unlock()
}

func (dc *DataCache) refreshBalances() {
	// 从已缓存的 wallets 中提取 funding 钱包的 Balance 和 Available
	dc.mu.RLock()
	wallets := dc.wallets
	dc.mu.RUnlock()

	currencies := dc.currencyManager.GetEnabledCurrencies()
	balances := make(map[string]float64)
	available := make(map[string]float64)
	for _, cur := range currencies {
		upper := strings.ToUpper(cur)
		for _, wallet := range wallets {
			if wallet.Currency == upper && wallet.Type == "funding" {
				balances[upper] = wallet.Balance
				available[upper] = wallet.Available
				break
			}
		}
	}
	dc.mu.Lock()
	dc.balances = balances
	dc.available = available
	dc.balancesUpdated = time.Now()
	dc.mu.Unlock()
}

func (dc *DataCache) refreshOffers() {
	currencies := dc.currencyManager.GetEnabledCurrencies()
	offers := make(map[string][]*bitfinex.FundingOffer)
	for _, cur := range currencies {
		upper := strings.ToUpper(cur)
		symbol := constants.FundingSymbolPrefix + upper
		o, err := dc.client.GetFundingOffers(symbol)
		if err != nil {
			log.Printf("[Cache] 刷新 %s offers 失败: %v", upper, err)
			continue
		}
		offers[upper] = o
	}
	dc.mu.Lock()
	dc.offers = offers
	dc.offersUpdated = time.Now()
	dc.mu.Unlock()
}

func (dc *DataCache) refreshCredits() {
	currencies := dc.currencyManager.GetEnabledCurrencies()
	credits := make(map[string][]*bitfinex.FundingCredit)
	for _, cur := range currencies {
		upper := strings.ToUpper(cur)
		symbol := constants.FundingSymbolPrefix + upper
		c, err := dc.client.GetFundingCredits(symbol)
		if err != nil {
			log.Printf("[Cache] 刷新 %s credits 失败: %v", upper, err)
			continue
		}
		credits[upper] = c
	}
	dc.mu.Lock()
	dc.credits = credits
	dc.creditsUpdated = time.Now()
	dc.mu.Unlock()
}

func (dc *DataCache) refreshFRRRates() {
	currencies := dc.currencyManager.GetEnabledCurrencies()
	rates := make(map[string]float64)
	for _, cur := range currencies {
		upper := strings.ToUpper(cur)
		symbol := constants.FundingSymbolPrefix + upper
		rate, err := dc.client.GetCurrentFundingRate(symbol)
		if err != nil {
			log.Printf("[Cache] 刷新 %s FRR 失败: %v", upper, err)
			continue
		}
		rates[upper] = rate
	}
	dc.mu.Lock()
	dc.frrRates = rates
	dc.frrUpdated = time.Now()
	dc.mu.Unlock()
}

func (dc *DataCache) refreshEarnings() {
	currencies := dc.currencyManager.GetEnabledCurrencies()
	earnings := make(map[string]EarningsData)
	for _, cur := range currencies {
		upper := strings.ToUpper(cur)
		e := dc.handler.getCurrencyEarningsInternal(upper)
		earnings[upper] = e
	}
	dc.mu.Lock()
	dc.earnings = earnings
	dc.earningsUpdated = time.Now()
	dc.mu.Unlock()
}

// --- 读取缓存的方法 ---

// GetWallets 获取缓存的钱包数据
func (dc *DataCache) GetWallets() []*bitfinex.Wallet {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.wallets
}

// GetBalance 获取缓存的总余额 (wallet.Balance)
func (dc *DataCache) GetBalance(currency string) float64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.balances[strings.ToUpper(currency)]
}

// GetAvailable 获取缓存的可用余额 (wallet.Available)
func (dc *DataCache) GetAvailable(currency string) float64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.available[strings.ToUpper(currency)]
}

// GetOffers 获取缓存的 offers
func (dc *DataCache) GetOffers(currency string) []*bitfinex.FundingOffer {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.offers[strings.ToUpper(currency)]
}

// GetAllOffers 获取所有币种的 offers
func (dc *DataCache) GetAllOffers() map[string][]*bitfinex.FundingOffer {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	result := make(map[string][]*bitfinex.FundingOffer)
	for k, v := range dc.offers {
		result[k] = v
	}
	return result
}

// GetCredits 获取缓存的 credits
func (dc *DataCache) GetCredits(currency string) []*bitfinex.FundingCredit {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.credits[strings.ToUpper(currency)]
}

// GetAllCredits 获取所有币种的 credits
func (dc *DataCache) GetAllCredits() map[string][]*bitfinex.FundingCredit {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	result := make(map[string][]*bitfinex.FundingCredit)
	for k, v := range dc.credits {
		result[k] = v
	}
	return result
}

// GetFRRRate 获取缓存的 FRR 利率
func (dc *DataCache) GetFRRRate(currency string) float64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.frrRates[strings.ToUpper(currency)]
}

// GetAllFRRRates 获取所有 FRR 利率
func (dc *DataCache) GetAllFRRRates() map[string]float64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	result := make(map[string]float64)
	for k, v := range dc.frrRates {
		result[k] = v
	}
	return result
}

// GetEarnings 获取缓存的收益数据
func (dc *DataCache) GetEarnings(currency string) (EarningsData, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	e, ok := dc.earnings[strings.ToUpper(currency)]
	return e, ok
}

// GetAllEarnings 获取所有币种的收益数据
func (dc *DataCache) GetAllEarnings() map[string]EarningsData {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	result := make(map[string]EarningsData)
	for k, v := range dc.earnings {
		result[k] = v
	}
	return result
}

// GetAllBalances 获取所有总余额
func (dc *DataCache) GetAllBalances() map[string]float64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	result := make(map[string]float64)
	for k, v := range dc.balances {
		result[k] = v
	}
	return result
}

// GetAllAvailable 获取所有可用余额
func (dc *DataCache) GetAllAvailable() map[string]float64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	result := make(map[string]float64)
	for k, v := range dc.available {
		result[k] = v
	}
	return result
}
