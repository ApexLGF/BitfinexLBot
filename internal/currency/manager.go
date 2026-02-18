package currency

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
	"github.com/ApexLGF/BitfinexLBot/internal/rates"
	"github.com/ApexLGF/BitfinexLBot/internal/strategy"
)

// CurrencyManager 幣種管理器
type CurrencyManager struct {
	config        *config.Config
	bfxClient     *bitfinex.Client
	lendingBots   map[string]*strategy.LendingBot
	rateConverter *rates.Converter
	mu            sync.RWMutex
}

// NewCurrencyManager 創建幣種管理器
func NewCurrencyManager(cfg *config.Config, client *bitfinex.Client) *CurrencyManager {
	cm := &CurrencyManager{
		config:        cfg,
		bfxClient:     client,
		lendingBots:   make(map[string]*strategy.LendingBot),
		rateConverter: rates.NewConverter(),
	}

	// 為每個啟用的幣種創建 LendingBot
	for currency, currencyConfig := range cfg.Currencies {
		if currencyConfig.Enabled {
			cm.lendingBots[currency] = strategy.NewLendingBot(cfg, currencyConfig, currency, client)
			log.Printf("[CurrencyManager] 初始化幣種: %s", currency)
		}
	}

	if len(cm.lendingBots) == 0 {
		log.Println("[CurrencyManager] 警告: 沒有啟用的幣種")
	} else {
		log.Printf("[CurrencyManager] 已初始化 %d 個幣種", len(cm.lendingBots))
	}

	return cm
}

// ExecuteAll 執行所有幣種的策略
func (cm *CurrencyManager) ExecuteAll() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var lastErr error
	for currency, bot := range cm.lendingBots {
		log.Printf("[CurrencyManager] 執行幣種策略: %s", currency)
		if err := bot.Execute(); err != nil {
			log.Printf("[CurrencyManager] 幣種 %s 執行失敗: %v", currency, err)
			lastErr = err
			// 繼續執行其他幣種
		}
	}

	return lastErr
}

// GetBot 獲取特定幣種的 LendingBot
func (cm *CurrencyManager) GetBot(currency string) (*strategy.LendingBot, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	bot, ok := cm.lendingBots[currency]
	return bot, ok
}

// GetEnabledCurrencies 獲取所有啟用的幣種
func (cm *CurrencyManager) GetEnabledCurrencies() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	currencies := make([]string, 0, len(cm.lendingBots))
	for currency := range cm.lendingBots {
		// 返回大写币种
		currencies = append(currencies, strings.ToUpper(currency))
	}
	return currencies
}

// ReloadConfig 重新加載配置
func (cm *CurrencyManager) ReloadConfig(newConfig *config.Config) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config = newConfig

	// 重新初始化所有 LendingBot
	cm.lendingBots = make(map[string]*strategy.LendingBot)
	for currency, currencyConfig := range newConfig.Currencies {
		if currencyConfig.Enabled {
			cm.lendingBots[currency] = strategy.NewLendingBot(newConfig, currencyConfig, currency, cm.bfxClient)
			log.Printf("[CurrencyManager] 重新初始化幣種: %s", currency)
		}
	}

	if len(cm.lendingBots) == 0 {
		return fmt.Errorf("沒有啟用的幣種")
	}

	log.Printf("[CurrencyManager] 配置重新加載完成，共 %d 個幣種", len(cm.lendingBots))
	return nil
}
