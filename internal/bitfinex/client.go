package bitfinex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bitfinexcom/bitfinex-api-go/pkg/models/common"
	"github.com/bitfinexcom/bitfinex-api-go/pkg/models/fundingoffer"
	"github.com/bitfinexcom/bitfinex-api-go/v2/rest"

	"github.com/ApexLGF/BitfinexLBot/internal/constants"
	"github.com/ApexLGF/BitfinexLBot/internal/errors"
)

// Client Bitfinex API 客戶端封裝
type Client struct {
	restClient *rest.Client
	key        string
	secret     string
	nonceGen   *CustomNonceGenerator // 保存nonce生成器用于直接HTTP调用
}

// NewClient 創建新的 Bitfinex 客戶端，使用自定义的线程安全nonce生成器
func NewClient(apiKey, secretKey string) *Client {
	// 创建自定义nonce生成器，使用较大的步长以避免高频API调用时的nonce冲突
	// 步长100适合高频交易场景，进一步减少nonce冲突
	nonceGen := NewCustomNonceGeneratorWithStep(100)
	
	// 使用自定义nonce生成器创建客户端  
	// 使用认证端点URL: https://api.bitfinex.com/v2/ (用于认证API调用)
	client := rest.NewClientWithURLNonce("https://api.bitfinex.com/v2/", nonceGen)
	client = client.Credentials(apiKey, secretKey)
	
	return &Client{
		restClient: client,
		key:        apiKey,
		secret:     secretKey,
		nonceGen:   nonceGen, // 保存nonce生成器用于直接HTTP调用
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
	// 获取资金订单（日志已移除）
	offers, err := c.restClient.Funding.Offers(symbol)
	
	if err != nil {
		log.Printf("❌ GetFundingOffers 错误: %v", err)
		// 處理特殊的空響應錯誤
		if strings.Contains(err.Error(), "data slice too short for funding offer") {
			return []*FundingOffer{}, nil
		}
		return nil, errors.NewAPIError("failed to get funding offers", err)
	}

	// 處理空響應或無數據的情況
	if offers == nil || offers.Snapshot == nil || len(offers.Snapshot) == 0 {
		return []*FundingOffer{}, nil
	}

	result := make([]*FundingOffer, 0, len(offers.Snapshot))
	for _, offer := range offers.Snapshot {
		// 添加安全檢查，防止空數據導致panic
		if offer == nil {
			continue
		}
		result = append(result, &FundingOffer{
			ID:     offer.ID,
			Amount: offer.Amount,
			Rate:   offer.Rate, // API 已返回日利率
			Period: int(offer.Period),
		})
	}

	return result, nil
}

// CancelFundingOffer 取消資金貸出訂單
func (c *Client) CancelFundingOffer(offerID int64) error {
	cancelReq := &fundingoffer.CancelRequest{
		ID: offerID,
	}

	_, err := c.restClient.Funding.CancelOffer(cancelReq)
	if err != nil {
		return errors.NewOrderError("failed to cancel funding offer", err)
	}

	return nil
}

// SubmitFundingOffer 提交新的資金貸出訂單
func (c *Client) SubmitFundingOffer(symbol string, amount float64, dailyRate float64, period int, hidden bool) error {
	offerReq := &fundingoffer.SubmitRequest{
		Type:   constants.OfferTypeLIMIT,
		Symbol: symbol,
		Amount: amount,
		Rate:   dailyRate, // v2 API 使用日利率
		Period: int64(period),
		Hidden: hidden,
	}

	_, err := c.restClient.Funding.SubmitOffer(offerReq)
	if err != nil {
		return errors.NewOrderError("failed to submit funding offer", err)
	}

	return nil
}

// GetWallets 獲取錢包信息
func (c *Client) GetWallets() ([]*Wallet, error) {
	// 获取钱包信息
	wallets, err := c.restClient.Wallet.Wallet()
	if err != nil {
		return nil, errors.NewAPIError("failed to get wallets", err)
	}

	result := make([]*Wallet, 0, len(wallets.Snapshot))
	for _, w := range wallets.Snapshot {
		result = append(result, &Wallet{
			Currency:  w.Currency,
			Type:      w.Type,
			Balance:   w.Balance,
			Available: w.BalanceAvailable,
		})
	}

	return result, nil
}

// GetTotalBalance 獲取指定幣種的總錢包餘額（所有錢包類型的總和）
func (c *Client) GetTotalBalance(currency string) (float64, error) {
	wallets, err := c.GetWallets()
	if err != nil {
		return 0, err
	}

	totalBalance := 0.0
	for _, wallet := range wallets {
		if wallet.Currency == currency {
			totalBalance += wallet.Balance
		}
	}

	return totalBalance, nil
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

	book, err := c.restClient.Book.All(symbol, common.PrecisionRawBook, limit)
	if err != nil {
		return nil, errors.NewAPIError("failed to get funding book", err)
	}

	if len(book.Snapshot) == 0 {
		return []*FundingBookEntry{}, nil
	}

	result := make([]*FundingBookEntry, 0, len(book.Snapshot))
	for _, entry := range book.Snapshot {
		result = append(result, &FundingBookEntry{
			Rate:   entry.Rate, // API 已返回日利率
			Amount: entry.Amount,
			Period: int(entry.Period),
			Count:  int(entry.Count),
		})
	}

	return result, nil
}

// GetCurrentFundingRate 獲取當前資金利率（Flash Return Rate）
func (c *Client) GetCurrentFundingRate(symbol string) (float64, error) {
	// 使用 ticker API 獲取真正的當前 funding rate (FRR)
	url := fmt.Sprintf("https://api-pub.bitfinex.com/v2/ticker/%s", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, errors.NewAPIError("failed to get funding ticker", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.NewAPIError(fmt.Sprintf("API returned status code %d", resp.StatusCode), nil)
	}

	var tickerData []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tickerData); err != nil {
		return 0, errors.NewAPIError("failed to decode ticker response", err)
	}

	// 檢查響應數據格式
	if len(tickerData) < 2 {
		return 0, errors.NewAPIError("invalid ticker response format", nil)
	}

	// 對於 funding symbols，FRR (Flash Return Rate) 在索引 1
	frr, ok := tickerData[1].(float64)
	if !ok {
		return 0, errors.NewAPIError("failed to parse FRR from ticker", nil)
	}

	// FRR 已經是日利率格式
	return frr, nil
}

// GetFundingCredits 獲取活躍的借貸訂單
func (c *Client) GetFundingCredits(symbol string) ([]*FundingCredit, error) {
	// 首先嘗試使用直接API調用獲取更多記錄
	allCredits, err := c.getFundingCreditsWithLimit(symbol, 100) // 嘗試獲取100條記錄
	if err == nil && len(allCredits) > 0 {
		return allCredits, nil
	}

	// 如果直接API調用失敗，回退到使用Go庫的方法
	credits, err := c.restClient.Funding.Credits(symbol)
	if err != nil {
		// 處理特殊的空響應錯誤
		if strings.Contains(err.Error(), "data slice too short") {
			return []*FundingCredit{}, nil
		}
		return nil, errors.NewAPIError("failed to get funding credits", err)
	}

	// 處理空響應或無數據的情況
	if credits == nil || credits.Snapshot == nil || len(credits.Snapshot) == 0 {
		return []*FundingCredit{}, nil
	}

	result := make([]*FundingCredit, 0, len(credits.Snapshot))
	for _, credit := range credits.Snapshot {
		// 添加安全檢查，防止空數據導致panic
		if credit == nil {
			continue
		}
		result = append(result, &FundingCredit{
			ID:         credit.ID,
			Symbol:     credit.Symbol,
			Amount:     credit.Amount,
			Rate:       credit.Rate, // API 已返回日利率
			Period:     credit.Period,
			MTSCreated: credit.MTSCreated,
			MTSOpened:  credit.MTSOpened,
			Status:     credit.Status,
		})
	}

	return result, nil
}

// getFundingCreditsWithLimit 使用直接HTTP請求獲取funding credits，支持limit參數
func (c *Client) getFundingCreditsWithLimit(symbol string, limit int) ([]*FundingCredit, error) {
	// 構建API URL - 使用Bitfinex API v2直接端點
	url := fmt.Sprintf("https://api.bitfinex.com/v2/auth/r/funding/credits/%s?limit=%d", symbol, limit)

	// 創建HTTP請求
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加認證頭 - 使用客户端的nonce生成器（与SDK统一）
	timestamp := c.nonceGen.GetNonceUint64() // 使用与SDK相同的nonce生成器
	body := ""
	payload := fmt.Sprintf("/api/v2/auth/r/funding/credits/%s%d%s", symbol, timestamp, body)
	
	// 計算簽名
	h := hmac.New(sha512.New384, []byte(c.secret))
	h.Write([]byte(payload))
	signature := hex.EncodeToString(h.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("bfx-nonce", fmt.Sprintf("%d", timestamp))
	req.Header.Set("bfx-apikey", c.key)
	req.Header.Set("bfx-signature", signature)

	// 發送請求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 讀取響應
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析JSON響應
	var rawCredits [][]interface{}
	if err := json.Unmarshal(bodyBytes, &rawCredits); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 轉換為FundingCredit結構
	result := make([]*FundingCredit, 0, len(rawCredits))
	for _, raw := range rawCredits {
		if len(raw) < 15 { // Bitfinex funding credit響應至少需要15個字段
			continue
		}

		// 安全地轉換每個字段
		credit := &FundingCredit{}
		
		if id, ok := raw[0].(float64); ok {
			credit.ID = int64(id)
		}
		if symbol, ok := raw[1].(string); ok {
			credit.Symbol = symbol
		}
		if amount, ok := raw[5].(float64); ok {
			credit.Amount = amount
		}
		if rate, ok := raw[11].(float64); ok {
			credit.Rate = rate
		}
		if period, ok := raw[12].(float64); ok {
			credit.Period = int64(period)
		}
		if mtsCreated, ok := raw[3].(float64); ok {
			credit.MTSCreated = int64(mtsCreated)
		}
		if mtsOpened, ok := raw[4].(float64); ok {
			credit.MTSOpened = int64(mtsOpened)
		}
		if status, ok := raw[10].(string); ok {
			credit.Status = status
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
		return nil, errors.NewAPIError("failed to get funding candles", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAPIError(fmt.Sprintf("API returned status code %d", resp.StatusCode), nil)
	}

	// 解析響應
	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, errors.NewAPIError("failed to decode candles response", err)
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

// LedgerEntry 代表账本条目
type LedgerEntry struct {
	ID          int64   `json:"id"`
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
	Balance     float64 `json:"balance"`
	Description string  `json:"description"`
	Timestamp   int64   `json:"timestamp"`
}

// GetDailyFundingEarnings 获取指定日期的资金借贷收益
func (c *Client) GetDailyFundingEarnings(currency string, date time.Time) (float64, error) {
	// 计算24小时时间范围：从指定日期开始到现在
	startTime := date
	endTime := time.Now().UTC()
	
	// 转换为毫秒时间戳
	start := startTime.UnixNano() / 1e6
	end := endTime.UnixNano() / 1e6
	
	// 构建API URL
	url := fmt.Sprintf("https://api.bitfinex.com/v2/auth/r/ledgers/%s/hist", currency)
	
	// 构建请求体，包含时间范围限制
	requestBody := fmt.Sprintf(`{"start": %d, "end": %d, "limit": 1000}`, start, end)
	
	// 创建HTTP请求 - 使用POST方法（Bitfinex认证API要求POST）
	req, err := http.NewRequest("POST", url, strings.NewReader(requestBody))
	if err != nil {
		return 0, errors.NewAPIError("failed to create request", err)
	}
	
	// 设置认证头 - 使用客户端的nonce生成器（与SDK统一）
	timestamp := c.nonceGen.GetNonceUint64() // 使用与SDK相同的nonce生成器
	apiPath := fmt.Sprintf("/api/v2/auth/r/ledgers/%s/hist", currency)
	payload := fmt.Sprintf("%s%d%s", apiPath, timestamp, requestBody)
	
	mac := hmac.New(sha512.New384, []byte(c.secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("bfx-nonce", fmt.Sprintf("%d", timestamp))
	req.Header.Set("bfx-apikey", c.key)
	req.Header.Set("bfx-signature", signature)
	
	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, errors.NewAPIError("failed to get ledger entries", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, errors.NewAPIError(fmt.Sprintf("API returned status code %d: %s", resp.StatusCode, string(body)), nil)
	}
	
	// 解析响应
	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return 0, errors.NewAPIError("failed to decode ledger response", err)
	}
	
	// 计算资金借贷收益总和
	// Bitfinex API v2 响应格式: [ID, CURRENCY, WALLET, MTS, null, AMOUNT, BALANCE, null, DESCRIPTION]
	totalEarnings := 0.0
	for _, raw := range rawData {
		if len(raw) < 9 {
			continue
		}
		
		// 检查描述字段是否包含资金收益相关信息 (索引8是DESCRIPTION)
		if description, ok := raw[8].(string); ok {
			if strings.Contains(description, "Margin Funding Payment") || strings.Contains(description, "Funding Payment") {
				// 索引5是AMOUNT
				if amount, ok := raw[5].(float64); ok && amount > 0 {
					totalEarnings += amount
				}
			}
		}
	}
	
	return totalEarnings, nil
}

// GetWeeklyFundingEarnings 获取过去7天的资金借贷收益
func (c *Client) GetWeeklyFundingEarnings(currency string) (float64, error) {
	// 计算时间范围：过去7天
	endTime := time.Now().UTC()
	startTime := endTime.AddDate(0, 0, -7)
	
	// 转换为毫秒时间戳
	start := startTime.UnixNano() / 1e6
	end := endTime.UnixNano() / 1e6
	
	// 构建API URL
	url := fmt.Sprintf("https://api.bitfinex.com/v2/auth/r/ledgers/%s/hist", currency)
	
	// 构建请求体
	requestBody := fmt.Sprintf(`{"start": %d, "end": %d, "limit": 2000}`, start, end)
	
	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, strings.NewReader(requestBody))
	if err != nil {
		return 0, errors.NewAPIError("failed to create request", err)
	}
	
	// 设置认证头 - 使用客户端的nonce生成器（与SDK统一）
	timestamp := c.nonceGen.GetNonceUint64() // 使用与SDK相同的nonce生成器
	apiPath := fmt.Sprintf("/api/v2/auth/r/ledgers/%s/hist", currency)
	payload := fmt.Sprintf("%s%d%s", apiPath, timestamp, requestBody)
	
	mac := hmac.New(sha512.New384, []byte(c.secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("bfx-nonce", fmt.Sprintf("%d", timestamp))
	req.Header.Set("bfx-apikey", c.key)
	req.Header.Set("bfx-signature", signature)
	
	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, errors.NewAPIError("failed to get ledger entries", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, errors.NewAPIError(fmt.Sprintf("API returned status code %d: %s", resp.StatusCode, string(body)), nil)
	}
	
	// 解析响应
	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return 0, errors.NewAPIError("failed to decode ledger response", err)
	}
	
	// 计算资金借贷收益总和
	// Bitfinex API v2 响应格式: [ID, CURRENCY, WALLET, MTS, null, AMOUNT, BALANCE, null, DESCRIPTION]
	totalEarnings := 0.0
	for _, raw := range rawData {
		if len(raw) < 9 {
			continue
		}
		
		// 检查描述字段是否包含资金收益相关信息 (索引8是DESCRIPTION)
		if description, ok := raw[8].(string); ok {
			if strings.Contains(description, "Margin Funding Payment") || strings.Contains(description, "Funding Payment") {
				// 索引5是AMOUNT
				if amount, ok := raw[5].(float64); ok && amount > 0 {
					totalEarnings += amount
				}
			}
		}
	}
	
	return totalEarnings, nil
}
