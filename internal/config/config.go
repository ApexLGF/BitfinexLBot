package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/ApexLGF/BitfinexLBot/internal/constants"
	"github.com/ApexLGF/BitfinexLBot/internal/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// CurrencyConfig 單個幣種的配置
type CurrencyConfig struct {
	Enabled                       bool    `mapstructure:"ENABLED" json:"ENABLED"`
	MinLoan                       float64 `mapstructure:"MIN_LOAN" json:"MIN_LOAN"`
	MaxLoan                       float64 `mapstructure:"MAX_LOAN" json:"MAX_LOAN"`
	MinDailyLendRate              float64 `mapstructure:"MIN_DAILY_LEND_RATE" json:"MIN_DAILY_LEND_RATE"`
	SpreadLend                    int     `mapstructure:"SPREAD_LEND" json:"SPREAD_LEND"`
	GapBottom                     float64 `mapstructure:"GAP_BOTTOM" json:"GAP_BOTTOM"`
	GapTop                        float64 `mapstructure:"GAP_TOP" json:"GAP_TOP"`
	ThirtyDayLendRateThreshold    float64 `mapstructure:"THIRTY_DAY_LEND_RATE_THRESHOLD" json:"THIRTY_DAY_LEND_RATE_THRESHOLD"`
	OneTwentyDayLendRateThreshold float64 `mapstructure:"ONE_TWENTY_DAY_LEND_RATE_THRESHOLD" json:"ONE_TWENTY_DAY_LEND_RATE_THRESHOLD"`
	RateBonus                     float64 `mapstructure:"RATE_BONUS" json:"RATE_BONUS"`
	HighHoldRate                  float64 `mapstructure:"HIGH_HOLD_RATE" json:"HIGH_HOLD_RATE"`
	HighHoldAmount                float64 `mapstructure:"HIGH_HOLD_AMOUNT" json:"HIGH_HOLD_AMOUNT"`
	HighHoldOrders                int     `mapstructure:"HIGH_HOLD_ORDERS" json:"HIGH_HOLD_ORDERS"`
	NotifyRateThreshold           float64 `mapstructure:"NOTIFY_RATE_THRESHOLD" json:"NOTIFY_RATE_THRESHOLD"`
	ReserveAmount                 float64 `mapstructure:"RESERVE_AMOUNT" json:"RESERVE_AMOUNT"`
}

// Config 應用程式配置結構
type Config struct {
	// API 配置
	BitfinexApiKey    string `mapstructure:"BITFINEX_API_KEY"`
	BitfinexSecretKey string `mapstructure:"BITFINEX_SECRET_KEY"`

	// 基本設定
	OrderLimit int `mapstructure:"ORDER_LIMIT"`
	MinutesRun int `mapstructure:"MINUTES_RUN"`

	// 多幣種配置 - 新增
	Currencies map[string]*CurrencyConfig `mapstructure:"CURRENCIES"`

	// 向後兼容的單幣種配置（保留但標記為廢棄）
	Currency                      string  `mapstructure:"CURRENCY"`
	MinLoan                       float64 `mapstructure:"MIN_LOAN"`
	MaxLoan                       float64 `mapstructure:"MAX_LOAN"`
	MinDailyLendRate              float64 `mapstructure:"MIN_DAILY_LEND_RATE"`
	SpreadLend                    int     `mapstructure:"SPREAD_LEND"`
	GapBottom                     float64 `mapstructure:"GAP_BOTTOM"`
	GapTop                        float64 `mapstructure:"GAP_TOP"`
	ThirtyDayLendRateThreshold    float64 `mapstructure:"THIRTY_DAY_LEND_RATE_THRESHOLD"`
	OneTwentyDayLendRateThreshold float64 `mapstructure:"ONE_TWENTY_DAY_LEND_RATE_THRESHOLD"`
	RateBonus                     float64 `mapstructure:"RATE_BONUS"`
	HighHoldRate                  float64 `mapstructure:"HIGH_HOLD_RATE"`
	HighHoldAmount                float64 `mapstructure:"HIGH_HOLD_AMOUNT"`
	HighHoldOrders                int     `mapstructure:"HIGH_HOLD_ORDERS"`
	NotifyRateThreshold           float64 `mapstructure:"NOTIFY_RATE_THRESHOLD"`
	ReserveAmount                 float64 `mapstructure:"RESERVE_AMOUNT"`

	// Telegram 設定
	TelegramBotToken  string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	TelegramAuthToken string `mapstructure:"TELEGRAM_AUTH_TOKEN"`

	// 智能策略設定（全局）
	EnableSmartStrategy      bool    `mapstructure:"ENABLE_SMART_STRATEGY"`
	VolatilityThreshold      float64 `mapstructure:"VOLATILITY_THRESHOLD"`
	MaxRateMultiplier        float64 `mapstructure:"MAX_RATE_MULTIPLIER"`
	MinRateMultiplier        float64 `mapstructure:"MIN_RATE_MULTIPLIER"`
	RateRangeIncreasePercent float64 `mapstructure:"RATE_RANGE_INCREASE_PERCENT"` // 利率範圍增加百分比

	// K線策略設定（全局）
	EnableKlineStrategy bool    `mapstructure:"ENABLE_KLINE_STRATEGY"` // 啟用K線策略
	KlineTimeFrame      string  `mapstructure:"KLINE_TIME_FRAME"`      // K線時間框架，預設15m
	KlinePeriod         int     `mapstructure:"KLINE_PERIOD"`          // K線週期數量，預設24（6小時）
	KlineSpreadPercent  float64 `mapstructure:"KLINE_SPREAD_PERCENT"`  // K線最高點加成百分比，預設0%
	KlineSmoothMethod   string  `mapstructure:"KLINE_SMOOTH_METHOD"`   // K線利率平滑方法：max, sma, ema, hla, p90

	// 測試模式設定
	TestMode bool `mapstructure:"TEST_MODE"`

	// API 服務配置
	APIEnabled     bool     `mapstructure:"API_ENABLED"`      // 啟用REST API服務
	APIPort        int      `mapstructure:"API_PORT"`         // API服務端口，預設8089
	APIHost        string   `mapstructure:"API_HOST"`         // API服務主機，預設0.0.0.0
	APICorsOrigins []string `mapstructure:"API_CORS_ORIGINS"` // CORS允許的源
	APIAuthToken   string   `mapstructure:"API_AUTH_TOKEN"`   // API認證令牌（可選）

	// 借貸通知設定
	LastLendingCheckTime int64 // 上次檢查借貸訂單的時間戳
	LendingCheckMinutes  int   `mapstructure:"LENDING_CHECK_MINUTES"` // 借貸訂單檢查間隔（分鐘）

	// LLM AI 策略配置
	OpenAIAPIKey       string `mapstructure:"OPENAI_API_KEY"`        // OpenAI API Key
	OpenAIModel        string `mapstructure:"OPENAI_MODEL"`          // 模型名稱，預設 gpt-4o
	OpenAIBaseURL      string `mapstructure:"OPENAI_BASE_URL"`       // API Base URL，預設 https://api.openai.com/v1
	LLMDefaultStrategy int    `mapstructure:"LLM_DEFAULT_STRATEGY"`  // 預設策略類型 1/2/3
	LLMTimeoutSeconds  int    `mapstructure:"LLM_TIMEOUT_SECONDS"`   // API 超時時間（秒）
	EnableLLMStrategy  bool   `mapstructure:"ENABLE_LLM_STRATEGY"`   // 是否啟用 LLM 策略替代合成利率
	LLMCacheHours      int    `mapstructure:"LLM_CACHE_HOURS"`       // LLM 預測緩存時間（小時），預設 3
	LLMMaxRetries      int    `mapstructure:"LLM_MAX_RETRIES"`       // LLM 調用最大重試次數，預設 3
}

// LoadConfig 從文件加載配置
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.NewConfigError("failed to read config file", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, errors.NewConfigError("failed to unmarshal config", err)
	}

	// 檢查是否為舊格式配置並自動遷移
	if config.Currency != "" && len(config.Currencies) == 0 {
		fmt.Println("[Config] 檢測到舊格式配置，自動遷移到多幣種格式")
		config.migrateFromLegacyConfig()
	}

	// 設置多幣種配置的預設值
	config.setMultiCurrencyDefaults()

	// 設置智能策略參數的預設值
	config.setSmartStrategyDefaults()

	// 設置K線策略參數的預設值
	config.setKlineStrategyDefaults()

	// 設置借貸檢查間隔的預設值
	config.setLendingCheckDefaults()

	// 設置API配置的預設值
	config.setAPIDefaults()

	// 設置LLM配置的預設值
	config.setLLMDefaults()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 驗證配置有效性
func (c *Config) Validate() error {
	if c.BitfinexApiKey == "" || c.BitfinexApiKey == "your_api_key_here" {
		return errors.NewValidationError("BITFINEX_API_KEY is required and must be set to your actual API key")
	}
	if c.BitfinexSecretKey == "" || c.BitfinexSecretKey == "your_secret_key_here" {
		return errors.NewValidationError("BITFINEX_SECRET_KEY is required and must be set to your actual secret key")
	}

	// 驗證多幣種配置
	if len(c.Currencies) == 0 {
		return errors.NewValidationError("至少需要配置一個幣種")
	}

	// 驗證每個幣種的配置
	for currency, currencyConfig := range c.Currencies {
		if !currencyConfig.Enabled {
			continue
		}

		if currencyConfig.MinLoan <= 0 {
			return errors.NewValidationError(fmt.Sprintf("%s: MIN_LOAN must be positive", currency))
		}
		if currencyConfig.MaxLoan > 0 && currencyConfig.MaxLoan < currencyConfig.MinLoan {
			return errors.NewValidationError(fmt.Sprintf("%s: MAX_LOAN cannot be less than MIN_LOAN", currency))
		}
		if currencyConfig.MinDailyLendRate <= 0 {
			return errors.NewValidationError(fmt.Sprintf("%s: MIN_DAILY_LEND_RATE must be positive", currency))
		}
		if currencyConfig.SpreadLend <= 0 {
			return errors.NewValidationError(fmt.Sprintf("%s: SPREAD_LEND must be positive", currency))
		}
		if currencyConfig.GapBottom < 0 || currencyConfig.GapTop < 0 || currencyConfig.GapTop <= currencyConfig.GapBottom {
			return errors.NewValidationError(fmt.Sprintf("%s: invalid GAP_BOTTOM or GAP_TOP values", currency))
		}
	}

	// 驗證智能策略參數
	if c.EnableSmartStrategy {
		if c.VolatilityThreshold <= 0 || c.VolatilityThreshold > 0.01 {
			return errors.NewValidationError("VOLATILITY_THRESHOLD must be between 0 and 0.01")
		}
		if c.MaxRateMultiplier <= 1.0 || c.MaxRateMultiplier > 5.0 {
			return errors.NewValidationError("MAX_RATE_MULTIPLIER must be between 1.0 and 5.0")
		}
		if c.MinRateMultiplier < 0.1 || c.MinRateMultiplier >= 1.0 {
			return errors.NewValidationError("MIN_RATE_MULTIPLIER must be between 0.1 and 1.0")
		}
		if c.MinRateMultiplier >= c.MaxRateMultiplier {
			return errors.NewValidationError("MIN_RATE_MULTIPLIER must be less than MAX_RATE_MULTIPLIER")
		}
		if c.RateRangeIncreasePercent <= 0 || c.RateRangeIncreasePercent > 1.0 {
			return errors.NewValidationError("RATE_RANGE_INCREASE_PERCENT must be between 0 and 1.0 (0-100%)")
		}
	}

	// 驗證K線策略參數
	if c.EnableKlineStrategy {
		if c.KlineTimeFrame == "" {
			return errors.NewValidationError("KLINE_TIME_FRAME is required when ENABLE_KLINE_STRATEGY is true")
		}
		if c.KlinePeriod <= 0 {
			return errors.NewValidationError("KLINE_PERIOD must be positive")
		}
		if c.KlineSpreadPercent < 0 || c.KlineSpreadPercent > 100 {
			return errors.NewValidationError("KLINE_SPREAD_PERCENT must be between 0 and 100")
		}
		// 驗證平滑方法
		validMethods := []string{"max", "sma", "ema", "hla", "p90"}
		isValidMethod := false
		for _, method := range validMethods {
			if c.KlineSmoothMethod == method {
				isValidMethod = true
				break
			}
		}
		if !isValidMethod {
			return errors.NewValidationError("KLINE_SMOOTH_METHOD must be one of: max, sma, ema, hla, p90")
		}
	}

	// 驗證借貸檢查間隔
	if c.LendingCheckMinutes <= 0 {
		return errors.NewValidationError("LENDING_CHECK_MINUTES must be positive")
	}

	// 驗證API配置
	if c.APIEnabled {
		if c.APIPort <= 0 || c.APIPort > 65535 {
			return errors.NewValidationError("API_PORT must be between 1 and 65535")
		}
		if c.APIHost == "" {
			return errors.NewValidationError("API_HOST is required when API is enabled")
		}
	}

	return nil
}

// GetFundingSymbol 獲取 funding symbol
func (c *Config) GetFundingSymbol() string {
	return constants.FundingSymbolPrefix + strings.ToUpper(c.Currency)
}

// GetMinDailyRateDecimal 獲取最低日利率（小數格式）
func (c *Config) GetMinDailyRateDecimal() float64 {
	return c.MinDailyLendRate / constants.PercentageToDecimal
}

// GetHighHoldRateDecimal 獲取高額持有利率（小數格式）
func (c *Config) GetHighHoldRateDecimal() float64 {
	return c.HighHoldRate / constants.PercentageToDecimal
}

// GetThirtyDayThresholdDecimal 獲取30天閾值（小數格式）
func (c *Config) GetThirtyDayThresholdDecimal() float64 {
	return c.ThirtyDayLendRateThreshold / constants.PercentageToDecimal
}

// GetOneTwentyDayThresholdDecimal 獲取120天閾值（小數格式）
func (c *Config) GetOneTwentyDayThresholdDecimal() float64 {
	return c.OneTwentyDayLendRateThreshold / constants.PercentageToDecimal
}

// setSmartStrategyDefaults 設置智能策略參數的預設值
func (c *Config) setSmartStrategyDefaults() {
	// 如果智能策略啟用但參數為零，設置建議的預設值
	if c.EnableSmartStrategy {
		if c.VolatilityThreshold == 0 {
			c.VolatilityThreshold = constants.DefaultVolatilityThreshold
		}
		if c.MaxRateMultiplier == 0 {
			c.MaxRateMultiplier = constants.DefaultMaxRateMultiplier
		}
		if c.MinRateMultiplier == 0 {
			c.MinRateMultiplier = constants.DefaultMinRateMultiplier
		}
		if c.RateRangeIncreasePercent == 0 {
			c.RateRangeIncreasePercent = constants.RateRangeIncreasePercent
		}
	} else {
		// 如果智能策略未啟用，確保參數有預設值以防止驗證錯誤
		if c.VolatilityThreshold == 0 {
			c.VolatilityThreshold = constants.DefaultVolatilityThreshold
		}
		if c.MaxRateMultiplier == 0 {
			c.MaxRateMultiplier = constants.DefaultMaxRateMultiplier
		}
		if c.MinRateMultiplier == 0 {
			c.MinRateMultiplier = constants.DefaultMinRateMultiplier
		}
		if c.RateRangeIncreasePercent == 0 {
			c.RateRangeIncreasePercent = constants.RateRangeIncreasePercent
		}
	}
}

// setKlineStrategyDefaults 設置K線策略參數的預設值
func (c *Config) setKlineStrategyDefaults() {
	// 如果K線策略啟用但參數為空，設置預設值
	if c.EnableKlineStrategy {
		if c.KlineTimeFrame == "" {
			c.KlineTimeFrame = "15m"
		}
		if c.KlinePeriod == 0 {
			c.KlinePeriod = 24 // 6小時的15分鐘K線
		}
		if c.KlineSpreadPercent == 0 {
			c.KlineSpreadPercent = 0.0 // 0%加成
		}
		if c.KlineSmoothMethod == "" {
			c.KlineSmoothMethod = "ema" // 預設使用指數移動平均
		}
	}
}

// setLendingCheckDefaults 設置借貸檢查間隔的預設值
func (c *Config) setLendingCheckDefaults() {
	// 如果未設置借貸檢查間隔，預設為 10 分鐘
	if c.LendingCheckMinutes == 0 {
		c.LendingCheckMinutes = 10
	}
}

// setAPIDefaults 設置API配置的預設值
func (c *Config) setAPIDefaults() {
	// 預設啟用API服務
	if !c.APIEnabled {
		c.APIEnabled = true
	}

	// 預設端口8090（內部服務端口）
	if c.APIPort == 0 {
		c.APIPort = 8090
	}

	// 預設綁定所有地址
	if c.APIHost == "" {
		c.APIHost = "0.0.0.0"
	}

	// 預設CORS設置
	if len(c.APICorsOrigins) == 0 {
		c.APICorsOrigins = []string{"*"}
	}
}

// setLLMDefaults 設置LLM配置的預設值
func (c *Config) setLLMDefaults() {
	if c.OpenAIModel == "" {
		c.OpenAIModel = "gpt-4o"
	}
	if c.OpenAIBaseURL == "" {
		c.OpenAIBaseURL = "https://api.openai.com/v1"
	}
	if c.LLMDefaultStrategy == 0 {
		c.LLMDefaultStrategy = 1
	}
	if c.LLMTimeoutSeconds == 0 {
		c.LLMTimeoutSeconds = 30
	}
	if c.LLMCacheHours == 0 {
		c.LLMCacheHours = 3 // 預設 3 小時緩存
	}
	if c.LLMMaxRetries == 0 {
		c.LLMMaxRetries = 3 // 預設 3 次重試
	}
}

// SaveConfig 保存配置到文件
func SaveConfig(configPath string, config map[string]interface{}) error {
	// 读取现有配置文件以保持格式和注释
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	// 解析现有YAML文档以保留注释
	var existingYAML yaml.Node
	if err := yaml.Unmarshal(existingData, &existingYAML); err != nil {
		return fmt.Errorf("failed to parse existing config: %w", err)
	}

	// 更新配置值
	if err := updateYAMLNode(&existingYAML, config); err != nil {
		return fmt.Errorf("failed to update config values: %w", err)
	}

	// 写回文件
	updatedData, err := yaml.Marshal(&existingYAML)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// updateYAMLNode 递归更新YAML节点的值，保留结构和注释
func updateYAMLNode(node *yaml.Node, config map[string]interface{}) error {
	if node.Kind != yaml.DocumentNode {
		return nil
	}

	for _, docNode := range node.Content {
		if docNode.Kind == yaml.MappingNode {
			updateMappingNode(docNode, config)
		}
	}

	return nil
}

// updateMappingNode 更新映射节点（递归处理嵌套 map，大小写不敏感匹配）
func updateMappingNode(node *yaml.Node, config map[string]interface{}) {
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode {
			key := keyNode.Value
			// 尝试匹配大小写不敏感的键
			var newValue interface{}
			var exists bool
			for k, v := range config {
				if strings.EqualFold(k, key) {
					newValue = v
					exists = true
					break
				}
			}

			if exists {
				// 检查是否为嵌套 map
				if nestedMap, ok := newValue.(map[string]interface{}); ok {
					// 如果值节点是映射类型，递归更新
					if valueNode.Kind == yaml.MappingNode {
						updateMappingNode(valueNode, nestedMap)
					}
				} else {
					// 更新标量值但保留注释
					if valueNode.Kind == yaml.ScalarNode {
						valueNode.Value = formatConfigValue(newValue)
					}
				}
			}
		}
	}
}

// formatConfigValue 格式化配置值为字符串
func formatConfigValue(value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ReloadConfig 重新加载配置文件
func ReloadConfig(configPath string) (*Config, error) {
	return LoadConfig(configPath)
}

// migrateFromLegacyConfig 從舊配置遷移到多幣種格式
func (c *Config) migrateFromLegacyConfig() {
	c.Currencies = make(map[string]*CurrencyConfig)
	c.Currencies[c.Currency] = &CurrencyConfig{
		Enabled:                       true,
		MinLoan:                       c.MinLoan,
		MaxLoan:                       c.MaxLoan,
		MinDailyLendRate:              c.MinDailyLendRate,
		SpreadLend:                    c.SpreadLend,
		GapBottom:                     c.GapBottom,
		GapTop:                        c.GapTop,
		ThirtyDayLendRateThreshold:    c.ThirtyDayLendRateThreshold,
		OneTwentyDayLendRateThreshold: c.OneTwentyDayLendRateThreshold,
		RateBonus:                     c.RateBonus,
		HighHoldRate:                  c.HighHoldRate,
		HighHoldAmount:                c.HighHoldAmount,
		HighHoldOrders:                c.HighHoldOrders,
		NotifyRateThreshold:           c.NotifyRateThreshold,
		ReserveAmount:                 c.ReserveAmount,
	}
	fmt.Printf("[Config] 已將 %s 遷移到多幣種配置\n", c.Currency)
}

// setMultiCurrencyDefaults 設置多幣種配置的預設值
func (c *Config) setMultiCurrencyDefaults() {
	if len(c.Currencies) == 0 {
		return
	}

	// 為每個幣種設置預設值
	for currency, currencyConfig := range c.Currencies {
		if currencyConfig.SpreadLend == 0 {
			currencyConfig.SpreadLend = 30
		}
		if currencyConfig.GapBottom == 0 {
			currencyConfig.GapBottom = 10
		}
		if currencyConfig.GapTop == 0 {
			currencyConfig.GapTop = 5000
		}
		if currencyConfig.ThirtyDayLendRateThreshold == 0 {
			currencyConfig.ThirtyDayLendRateThreshold = 0.04
		}
		if currencyConfig.OneTwentyDayLendRateThreshold == 0 {
			currencyConfig.OneTwentyDayLendRateThreshold = 0.045
		}
		if currencyConfig.RateBonus == 0 {
			currencyConfig.RateBonus = 0.002
		}
		fmt.Printf("[Config] 已為幣種 %s 設置預設值\n", currency)
	}
}

