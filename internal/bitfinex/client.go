package bitfinex

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bitfinexcom/bitfinex-api-go/v2/rest"

	"github.com/ApexLGF/BitfinexLBot/internal/constants"
)

// Client Bitfinex API 客戶端封裝
type Client struct {
	restClient   *rest.Client
	apiKey       string
	secretKey    string
	nonceManager *NonceManager
	apiMutex     sync.Mutex // 全局API调用锁
}

// NewClient 創建新的 Bitfinex 客戶端
func NewClient(apiKey, secretKey string) *Client {
	client := rest.NewClient().Credentials(apiKey, secretKey)
	return &Client{
		restClient:   client,
		apiKey:       apiKey,
		secretKey:    secretKey,
		nonceManager: NewNonceManager(),
	}
}

// FundingOffer 代表一個資金貸出訂單
type FundingOffer struct {
	ID     int64
	Amount float64
	Rate   float64 // 日利率（小數格式）
	Period int
}

// Wallet 代表錢包信息
type Wallet struct {
	Currency  string
	Type      string
	Balance   float64
	Available float64
}

// FundingBookEntry 代表資金訂單簿條目
type FundingBookEntry struct {
	Rate   float64 // 日利率（小數格式）
	Amount float64
	Period int
	Count  int
}

// FundingCredit 代表活躍的借貸訂單
type FundingCredit struct {
	ID         int64
	Symbol     string
	Amount     float64
	Rate       float64 // 日利率（小數格式）
	Period     int64   // 期間（天）
	MTSCreated int64   // 創建時間戳（毫秒）
	MTSOpened  int64   // 開始時間戳（毫秒）
	Status     string  // 狀態
}

// Candle 代表 K 線數據
type Candle struct {
	MTS    int64   // 時間戳（毫秒）
	Open   float64 // 開盤價
	Close  float64 // 收盤價
	High   float64 // 最高價
	Low    float64 // 最低價
	Volume float64 // 成交量
}

// GetFundingOffers 獲取未完成的資金貸出訂單
func (c *Client) GetFundingOffers(symbol string) ([]*FundingOffer, error) {
	// 构建请求路径
	path := fmt.Sprintf("/v2/auth/r/funding/offers/%s", symbol)
	
	// 构建请求体（空请求体）
	requestBody := map[string]interface{}{}
	
	// 执行认证请求
	response, err := c.makeAuthenticatedRequest("POST", path, requestBody)
	if err != nil {
		// 处理特殊的空响应错误
		if strings.Contains(err.Error(), "data slice too short") {
			return []*FundingOffer{}, nil
		}
		return nil, fmt.Errorf("failed to get funding offers: %w", err)
	}
	
	// 解析响应
	var rawData [][]interface{}
	if err := json.Unmarshal(response, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse funding offers response: %w", err)
	}
	
	// 转换数据
	result := make([]*FundingOffer, 0, len(rawData))
	for _, raw := range rawData {
		if len(raw) < 16 { // 确保有足够的字段
			continue
		}
		
		// Funding offer格式根据实际调试输出: [ID, SYMBOL, MTSCreated, MTSUpdated, AMOUNT, AMOUNT_ORIG, TYPE, FLAGS, STATUS, ?, STATUS, ?, ?, ?, RATE, PERIOD, ...]
		
		// [0] ID
		id, ok := raw[0].(float64)
		if !ok {
			continue
		}
		
		// [4] AMOUNT
		amount, ok := raw[4].(float64)
		if !ok {
			continue
		}
		
		// [14] RATE (根据调试输出，实际的RATE在索引14)
		rate, ok := raw[14].(float64)
		if !ok {
			continue
		}
		
		// [15] PERIOD (根据调试输出，实际的PERIOD在索引15)
		period, ok := raw[15].(float64)
		if !ok {
			continue
		}
		
		offer := &FundingOffer{
			ID:     int64(id),
			Amount: amount,
			Rate:   rate, // API 已返回日利率
			Period: int(period),
		}
		
		result = append(result, offer)
	}
	
	return result, nil
}

// CancelFundingOffer 取消資金貸出訂單
func (c *Client) CancelFundingOffer(offerID int64) error {
	// 构建请求路径
	path := "/v2/auth/w/funding/offer/cancel"
	
	// 构建请求体
	requestBody := map[string]interface{}{
		"id": offerID,
	}
	
	// 执行认证请求
	_, err := c.makeAuthenticatedRequest("POST", path, requestBody)
	if err != nil {
		return fmt.Errorf("failed to cancel funding offer: %w", err)
	}

	return nil
}

// SubmitFundingOffer 提交新的資金貸出訂單
func (c *Client) SubmitFundingOffer(symbol string, amount float64, dailyRate float64, period int, hidden bool) error {
	// 构建请求路径
	path := "/v2/auth/w/funding/offer/submit"
	
	// 构建请求体
	flags := 0
	if hidden {
		flags = 64 // HIDDEN flag
	}
	
	requestBody := map[string]interface{}{
		"type":   constants.OfferTypeLIMIT,
		"symbol": symbol,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64),
		"rate":   strconv.FormatFloat(dailyRate, 'f', -1, 64), // v2 API 使用日利率
		"period": period,
		"flags":  flags,
	}
	
	// 执行认证请求
	_, err := c.makeAuthenticatedRequest("POST", path, requestBody)
	if err != nil {
		return fmt.Errorf("failed to submit funding offer: %w", err)
	}

	return nil
}

// GetWallets 獲取錢包信息
func (c *Client) GetWallets() ([]*Wallet, error) {
	// 构建请求路径
	path := "/v2/auth/r/wallets"
	
	// 构建请求体（空请求体）
	requestBody := map[string]interface{}{}
	
	// 执行认证请求
	response, err := c.makeAuthenticatedRequest("POST", path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallets: %w", err)
	}
	
	// 解析响应
	var rawData [][]interface{}
	if err := json.Unmarshal(response, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse wallets response: %w", err)
	}
	
	// 转换数据
	result := make([]*Wallet, 0, len(rawData))
	for _, raw := range rawData {
		if len(raw) < 5 { // 确保有足够的字段
			continue
		}
		
		// Wallet格式: [WALLET_TYPE, CURRENCY, BALANCE, UNSETTLED_INTEREST, BALANCE_AVAILABLE]
		
		// [0] WALLET_TYPE
		walletType, ok := raw[0].(string)
		if !ok {
			continue
		}
		
		// [1] CURRENCY
		currency, ok := raw[1].(string)
		if !ok {
			continue
		}
		
		// [2] BALANCE
		balance, ok := raw[2].(float64)
		if !ok {
			continue
		}
		
		// [4] BALANCE_AVAILABLE
		available, ok := raw[4].(float64)
		if !ok {
			continue
		}
		
		wallet := &Wallet{
			Currency:  currency,
			Type:      walletType,
			Balance:   balance,
			Available: available,
		}
		
		result = append(result, wallet)
	}
	
	return result, nil
}

// GetFundingBalance 獲取指定幣種的資金錢包餘額
func (c *Client) GetFundingBalance(currency string) (float64, error) {
	wallets, err := c.GetWallets()
	if err != nil {
		return 0, err
	}

	for _, wallet := range wallets {
		if wallet.Currency == currency && wallet.Type == constants.WalletTypeFunding {
			return wallet.Available, nil
		}
	}

	return 0, nil
}

// GetFundingBook 獲取資金訂單簿
func (c *Client) GetFundingBook(symbol string, limit int) ([]*FundingBookEntry, error) {
	if limit > constants.MaxPriceLevels {
		limit = constants.MaxPriceLevels
	}
	if limit <= 0 {
		limit = constants.DefaultPriceLevels
	}

	// 使用公共API获取资金订单簿（不需要认证）
	url := fmt.Sprintf("https://api-pub.bitfinex.com/v2/book/%s/R0", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get funding book: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("failed to decode funding book response: %w", err)
	}

	if len(rawData) == 0 {
		return []*FundingBookEntry{}, nil
	}

	result := make([]*FundingBookEntry, 0, len(rawData))
	for i, raw := range rawData {
		// 限制数量
		if i >= limit {
			break
		}
		
		if len(raw) < 4 { // 确保有足够的字段
			continue
		}
		
		// Book entry格式根据实际API响应: [ID, PERIOD, RATE, AMOUNT]
		
		// [1] PERIOD
		period, ok := raw[1].(float64)
		if !ok {
			continue
		}
		
		// [2] RATE (实际的RATE在索引2)
		rate, ok := raw[2].(float64)
		if !ok {
			continue
		}
		
		// [3] AMOUNT
		amount, ok := raw[3].(float64)
		if !ok {
			continue
		}
		
		entry := &FundingBookEntry{
			Rate:   rate, // API 已返回日利率
			Amount: amount,
			Period: int(period),
			Count:  1, // 设为固定值1，因为API没有返回COUNT字段
		}
		
		result = append(result, entry)
	}

	return result, nil
}

// GetCurrentFundingRate 獲取當前資金利率（Flash Return Rate）
func (c *Client) GetCurrentFundingRate(symbol string) (float64, error) {
	// 使用 ticker API 獲取真正的當前 funding rate (FRR)
	url := fmt.Sprintf("https://api-pub.bitfinex.com/v2/ticker/%s", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to get funding ticker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var tickerData []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tickerData); err != nil {
		return 0, fmt.Errorf("failed to decode ticker response: %w", err)
	}

	// 檢查響應數據格式
	if len(tickerData) < 2 {
		return 0, fmt.Errorf("invalid ticker response format")
	}

	// 對於 funding symbols，FRR (Flash Return Rate) 在索引 1
	frr, ok := tickerData[1].(float64)
	if !ok {
		return 0, fmt.Errorf("failed to parse FRR from ticker")
	}

	// FRR 已經是日利率格式
	return frr, nil
}

// GetFundingCredits 獲取活躍的借貸訂單
func (c *Client) GetFundingCredits(symbol string) ([]*FundingCredit, error) {
	// 构建请求路径
	path := fmt.Sprintf("/v2/auth/r/funding/credits/%s", symbol)
	
	// 构建请求体（空请求体）
	requestBody := map[string]interface{}{}
	
	// 执行认证请求
	response, err := c.makeAuthenticatedRequest("POST", path, requestBody)
	if err != nil {
		// 处理特殊的空响应错误
		if strings.Contains(err.Error(), "data slice too short") {
			return []*FundingCredit{}, nil
		}
		return nil, fmt.Errorf("failed to get funding credits: %w", err)
	}
	
	// 解析响应
	var rawData [][]interface{}
	if err := json.Unmarshal(response, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse funding credits response: %w", err)
	}

	log.Printf("[GetFundingCredits] %s API 返回 %d 条原始记录", symbol, len(rawData))

	// 转换数据
	result := make([]*FundingCredit, 0, len(rawData))
	for i, raw := range rawData {
		if len(raw) < 13 { // 确保有足够的字段
			log.Printf("[GetFundingCredits] 跳过第 %d 条记录: 字段数不足 (%d < 13)", i, len(raw))
			continue
		}

		// Funding credit格式根据实际调试输出: [ID, SYMBOL, SIDE, MTSCreated, MTSUpdated, AMOUNT, FLAGS, STATUS, TYPE, ?, ?, RATE, PERIOD, MTSOpened, ...]

		// [0] ID
		id, ok := raw[0].(float64)
		if !ok {
			log.Printf("[GetFundingCredits] 跳过第 %d 条记录: ID 类型断言失败, 值: %v (%T)", i, raw[0], raw[0])
			continue
		}

		// [1] SYMBOL
		symbol, ok := raw[1].(string)
		if !ok {
			log.Printf("[GetFundingCredits] 跳过第 %d 条记录: SYMBOL 类型断言失败, 值: %v (%T)", i, raw[1], raw[1])
			continue
		}

		// [3] MTSCreated
		mtsCreated := int64(0)
		if raw[3] != nil {
			if v, ok := raw[3].(float64); ok {
				mtsCreated = int64(v)
			}
		}

		// [5] AMOUNT
		amount := float64(0)
		if raw[5] != nil {
			if v, ok := raw[5].(float64); ok {
				amount = v
			}
		}

		// [7] STATUS
		status := ""
		if raw[7] != nil {
			if v, ok := raw[7].(string); ok {
				status = v
			}
		}

		// [11] RATE
		rate := float64(0)
		if raw[11] != nil {
			if v, ok := raw[11].(float64); ok {
				rate = v
			}
		}

		// [12] PERIOD
		period := int64(0)
		if raw[12] != nil {
			if v, ok := raw[12].(float64); ok {
				period = int64(v)
			}
		}

		// [13] MTSOpened
		mtsOpened := int64(0)
		if len(raw) > 13 && raw[13] != nil {
			if opened, ok := raw[13].(float64); ok {
				mtsOpened = int64(opened)
			}
		}
		
		credit := &FundingCredit{
			ID:         int64(id),
			Symbol:     symbol,
			Amount:     amount,
			Rate:       rate, // API 已返回日利率
			Period:     period,
			MTSCreated: mtsCreated,
			MTSOpened:  mtsOpened,
			Status:     status,
		}
		
		result = append(result, credit)
	}
	
	return result, nil
}

// GetFundingCandles 獲取資金 K 線數據
func (c *Client) GetFundingCandles(symbol string, timeFrame string, limit int) ([]*Candle, error) {
	// 構建 candle key，格式: trade:15m:fUSD:a30:p2:p30
	candleKey := fmt.Sprintf("trade:%s:%s:a30:p2:p30", timeFrame, symbol)

	// 構建 API URL
	url := fmt.Sprintf("https://api-pub.bitfinex.com/v2/candles/%s/hist?limit=%d", candleKey, limit)

	// 發送 HTTP 請求
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get funding candles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	// 解析響應
	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("failed to decode candles response: %w", err)
	}

	// 轉換為 Candle 結構
	candles := make([]*Candle, 0, len(rawData))
	for _, raw := range rawData {
		if len(raw) != 6 {
			continue // 跳過無效數據
		}

		// 安全地轉換每個字段
		mts, ok := raw[0].(float64)
		if !ok {
			continue
		}

		open, ok := raw[1].(float64)
		if !ok {
			continue
		}

		close, ok := raw[2].(float64)
		if !ok {
			continue
		}

		high, ok := raw[3].(float64)
		if !ok {
			continue
		}

		low, ok := raw[4].(float64)
		if !ok {
			continue
		}

		volume, ok := raw[5].(float64)
		if !ok {
			continue
		}

		candle := &Candle{
			MTS:    int64(mts),
			Open:   open,
			Close:  close,
			High:   high,
			Low:    low,
			Volume: volume,
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// LedgerEntry 账本条目
type LedgerEntry struct {
	ID          int64   `json:"id"`
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
	Balance     float64 `json:"balance"`
	Description string  `json:"description"`
	Timestamp   int64   `json:"timestamp"`
}

// GetFundingLedgers 获取资金账本记录
func (c *Client) GetFundingLedgers(currency string, start, end int64, limit int) ([]*LedgerEntry, error) {
	// 构建请求路径
	path := fmt.Sprintf("/v2/auth/r/ledgers/%s/hist", currency)
	
	// 构建请求体
	requestBody := map[string]interface{}{
		"limit": limit,
	}
	
	if start > 0 {
		requestBody["start"] = start
	}
	if end > 0 {
		requestBody["end"] = end
	}
	
	// 执行认证请求
	response, err := c.makeAuthenticatedRequest("POST", path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get funding ledgers: %w", err)
	}
	
	// 解析响应
	var rawData [][]interface{}
	if err := json.Unmarshal(response, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse ledgers response: %w", err)
	}
	
	// 转换数据
	var ledgers []*LedgerEntry
	for _, raw := range rawData {
		if len(raw) < 9 { // 确保有足够的字段
			continue
		}
		
		// [0] ID
		id, ok := raw[0].(float64)
		if !ok {
			continue
		}
		
		// [1] CURRENCY
		currency, ok := raw[1].(string)
		if !ok {
			continue
		}
		
		// [3] MTS (timestamp in milliseconds)
		timestamp, ok := raw[3].(float64)
		if !ok {
			continue
		}
		
		// [5] AMOUNT
		amount, ok := raw[5].(float64)
		if !ok {
			continue
		}
		
		// [6] BALANCE
		balance, ok := raw[6].(float64)
		if !ok {
			continue
		}
		
		// [8] DESCRIPTION
		description := ""
		if len(raw) > 8 && raw[8] != nil {
			if desc, ok := raw[8].(string); ok {
				description = desc
			}
		}
		
		ledger := &LedgerEntry{
			ID:          int64(id),
			Currency:    currency,
			Amount:      amount,
			Balance:     balance,
			Description: description,
			Timestamp:   int64(timestamp),
		}
		
		ledgers = append(ledgers, ledger)
	}
	
	return ledgers, nil
}

// makeAuthenticatedRequest 执行Bitfinex认证请求
func (c *Client) makeAuthenticatedRequest(method, path string, body map[string]interface{}) ([]byte, error) {
	// 使用全局锁确保API调用串行执行，避免nonce冲突
	c.apiMutex.Lock()
	defer c.apiMutex.Unlock()
	
	// 添加延迟以遵守Bitfinex API频率限制（10-90请求/分钟）
	time.Sleep(700 * time.Millisecond)
	// Bitfinex API base URL
	baseURL := "https://api-pub.bitfinex.com"
	
	// 序列化请求体
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	
	// 使用统一的nonce管理器生成nonce
	nonce := c.nonceManager.GetNextNonce()
	
	// 构建签名载荷
	payload := "/api" + path + nonce + string(bodyBytes)
	
	// 计算HMAC-SHA384签名
	h := hmac.New(sha512.New384, []byte(c.secretKey))
	h.Write([]byte(payload))
	signature := hex.EncodeToString(h.Sum(nil))
	
	// 创建HTTP请求
	req, err := http.NewRequest(method, baseURL+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 设置必要的头部
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("bfx-nonce", nonce)
	req.Header.Set("bfx-apikey", c.apiKey)
	req.Header.Set("bfx-signature", signature)
	
	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
