package strategy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ApexLGF/BitfinexLBot/internal/bitfinex"
	"github.com/ApexLGF/BitfinexLBot/internal/config"
)

// LLMStrategyType 策略類型
type LLMStrategyType int

const (
	StrategySimple         LLMStrategyType = 1 // 結構化數據 + 簡潔提示詞
	StrategyDetailed       LLMStrategyType = 2 // 完整市場深度 + 多維分析
	StrategyChainOfThought LLMStrategyType = 3 // 時序特徵 + 思維鏈推理
)

// LLMPrediction AI 預測結果
type LLMPrediction struct {
	PredictedRateLow  float64         `json:"predicted_rate_low"`  // 預測利率下限（日利率百分比）
	PredictedRateMid  float64         `json:"predicted_rate_mid"`  // 預測最可能利率
	PredictedRateHigh float64         `json:"predicted_rate_high"` // 預測利率上限
	Confidence        float64         `json:"confidence"`          // 置信度 0-1
	Trend             string          `json:"trend"`               // 趨勢判斷
	Reasoning         string          `json:"reasoning"`           // 推理說明
	Strategy          LLMStrategyType `json:"strategy"`            // 使用的策略
}

// LLMAIStrategy LLM AI 策略
type LLMAIStrategy struct {
	config     *config.Config
	analyzer   *MarketAnalyzer
	httpClient *http.Client
}

// OpenAI API 請求/響應結構
type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// NewLLMAIStrategy 創建 LLM AI 策略實例
func NewLLMAIStrategy(cfg *config.Config, analyzer *MarketAnalyzer) *LLMAIStrategy {
	return &LLMAIStrategy{
		config:   cfg,
		analyzer: analyzer,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.LLMTimeoutSeconds) * time.Second,
		},
	}
}

// Predict 執行預測（根據策略類型選擇方案）
func (s *LLMAIStrategy) Predict(strategyType LLMStrategyType, fundingBook []*bitfinex.FundingBookEntry) (*LLMPrediction, error) {
	if s.config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API Key 未配置")
	}

	condition := s.analyzer.AnalyzeMarket(fundingBook)

	var prompt string
	switch strategyType {
	case StrategySimple:
		prompt = s.buildSimplePrompt(condition)
	case StrategyDetailed:
		prompt = s.buildDetailedPrompt(condition, fundingBook)
	case StrategyChainOfThought:
		prompt = s.buildChainOfThoughtPrompt(condition)
	default:
		return nil, fmt.Errorf("未知的策略類型: %d", strategyType)
	}

	response, err := s.callOpenAI(prompt)
	if err != nil {
		return nil, fmt.Errorf("調用 OpenAI API 失敗: %w", err)
	}

	prediction, err := s.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析響應失敗: %w", err)
	}

	prediction.Strategy = strategyType
	return prediction, nil
}

// buildSimplePrompt 構建方案一的提示詞（簡潔版）
func (s *LLMAIStrategy) buildSimplePrompt(condition *MarketCondition) string {
	rateHistory := s.analyzer.GetRateHistory()
	var rateList strings.Builder
	for i, snapshot := range rateHistory {
		if i > 0 {
			rateList.WriteString(", ")
		}
		rateList.WriteString(fmt.Sprintf("%.4f%%", snapshot.Rate*100))
	}

	currentRate := 0.0
	if len(rateHistory) > 0 {
		currentRate = rateHistory[len(rateHistory)-1].Rate
	}

	return fmt.Sprintf(`你是 Bitfinex 放貸市場利率預測專家。根據以下市場數據，預測未來 4 小時的利率走勢。

當前市場狀態:
- 趨勢: %s
- 波動率: %.4f%%
- 當前利率: %.4f%%
- 12小時平均利率: %.4f%%
- 利率比率(當前/平均): %.2f

最近利率歷史 (每15分鐘):
%s

請以 JSON 格式輸出預測結果，格式如下:
{
  "predicted_rate_low": 0.01,
  "predicted_rate_mid": 0.015,
  "predicted_rate_high": 0.02,
  "confidence": 0.7,
  "trend": "rising",
  "reasoning": "簡要說明"
}

注意：利率值使用日利率百分比格式（如 0.01 表示 0.01%%/天）`,
		condition.Trend,
		condition.Volatility*100,
		currentRate*100,
		condition.AvgRate*100,
		condition.RateRatio,
		rateList.String())
}

// buildDetailedPrompt 構建方案二的提示詞（詳細版）
func (s *LLMAIStrategy) buildDetailedPrompt(condition *MarketCondition, fundingBook []*bitfinex.FundingBookEntry) string {
	rateHistory := s.analyzer.GetRateHistory()

	// 構建歷史利率表格
	var historyTable strings.Builder
	historyTable.WriteString("| 時間 | 日利率(%) | 成交量 |\n")
	historyTable.WriteString("|------|-----------|--------|\n")
	for _, snapshot := range rateHistory {
		historyTable.WriteString(fmt.Sprintf("| %s | %.4f | %.2f |\n",
			snapshot.Timestamp.Format("15:04"),
			snapshot.Rate*100,
			snapshot.Volume))
	}

	// 構建訂單簿表格（前20層）
	var bookTable strings.Builder
	bookTable.WriteString("| 層級 | 利率(%) | 可用金額 |\n")
	bookTable.WriteString("|------|---------|----------|\n")
	maxLayers := 20
	if len(fundingBook) < maxLayers {
		maxLayers = len(fundingBook)
	}
	for i := 0; i < maxLayers; i++ {
		entry := fundingBook[i]
		bookTable.WriteString(fmt.Sprintf("| %d | %.4f | %.2f |\n",
			i+1, entry.Rate*100, entry.Amount))
	}

	// 計算競爭分析數據
	avgSpread := 0.0
	if len(fundingBook) >= 10 {
		var totalSpread float64
		for i := 0; i < 9; i++ {
			spread := fundingBook[i+1].Rate - fundingBook[i].Rate
			if spread > 0 {
				totalSpread += spread
			}
		}
		avgSpread = totalSpread / 9
	}

	currentRate := 0.0
	if len(rateHistory) > 0 {
		currentRate = rateHistory[len(rateHistory)-1].Rate
	}

	return fmt.Sprintf(`作為量化金融分析師，分析 Bitfinex USD 放貸市場並預測未來 4 小時利率。

## 歷史利率序列 (過去 12 小時)
%s

## 當前訂單簿深度 (前 20 層)
%s

## 市場指標
- 趨勢判斷: %s (基於最近 6 個數據點)
- 波動率: %.4f%% (標準差)
- 流動性深度: %d 層
- 當前利率: %.4f%%
- 平均利率: %.4f%%
- 利率比率: %.2f
- 平均利差: %.6f%%

請分析:
1. 短期(1h)、中期(4h) 利率預測區間
2. 關鍵支撐/阻力利率位
3. 可能觸發利率變化的因素
4. 放貸策略建議 (激進/保守/觀望)

以 JSON 格式輸出:
{
  "predicted_rate_low": 0.01,
  "predicted_rate_mid": 0.015,
  "predicted_rate_high": 0.02,
  "confidence": 0.7,
  "trend": "rising",
  "reasoning": "詳細分析說明"
}

注意：利率值使用日利率百分比格式`,
		historyTable.String(),
		bookTable.String(),
		condition.Trend,
		condition.Volatility*100,
		condition.LiquidityDepth,
		currentRate*100,
		condition.AvgRate*100,
		condition.RateRatio,
		avgSpread*100)
}

// buildChainOfThoughtPrompt 構建方案三的提示詞（思維鏈版）
func (s *LLMAIStrategy) buildChainOfThoughtPrompt(condition *MarketCondition) string {
	rateHistory := s.analyzer.GetRateHistory()

	// 計算時序特徵
	currentRate := 0.0
	rate1hAgo := 0.0
	rate4hAgo := 0.0
	rate12hAgo := 0.0

	historyLen := len(rateHistory)
	if historyLen > 0 {
		currentRate = rateHistory[historyLen-1].Rate
	}
	if historyLen >= 4 { // 1小時前 (4個15分鐘)
		rate1hAgo = rateHistory[historyLen-4].Rate
	}
	if historyLen >= 16 { // 4小時前
		rate4hAgo = rateHistory[historyLen-16].Rate
	}
	if historyLen >= 48 { // 12小時前
		rate12hAgo = rateHistory[0].Rate
	}

	// 計算動量
	momentum := 0.0
	if rate1hAgo > 0 {
		momentum = (currentRate - rate1hAgo) / rate1hAgo
	}

	// 計算趨勢強度
	trendStrength := 0
	if historyLen >= 6 {
		recentHistory := rateHistory[historyLen-6:]
		for i := 1; i < len(recentHistory); i++ {
			diff := recentHistory[i].Rate - recentHistory[i-1].Rate
			if diff > 0.0001 {
				trendStrength++
			} else if diff < -0.0001 {
				trendStrength--
			}
		}
	}

	features := map[string]interface{}{
		"current_rate":          currentRate * 100,
		"rate_1h_ago":           rate1hAgo * 100,
		"rate_4h_ago":           rate4hAgo * 100,
		"rate_12h_ago":          rate12hAgo * 100,
		"rate_momentum":         momentum * 100,
		"volatility":            condition.Volatility * 100,
		"trend":                 condition.Trend,
		"trend_strength":        trendStrength,
		"liquidity_depth":       condition.LiquidityDepth,
		"rate_ratio":            condition.RateRatio,
		"data_points_available": historyLen,
	}

	featuresJSON, _ := json.MarshalIndent(features, "", "  ")

	return fmt.Sprintf(`## 任務
預測 Bitfinex fUSD 放貸利率未來 4 小時走勢。

## 時序特徵 (已預處理)
%s

## 推理要求
請按以下步驟思考:

1. **趨勢識別**: 當前處於什麼週期？動量是否在衰減？
2. **波動分析**: 波動率處於什麼水平？是否有異常？
3. **供需判斷**: 流動性深度說明什麼？
4. **綜合預測**: 基於以上分析，4 小時後利率最可能在什麼範圍？

## 輸出格式
請以 JSON 格式輸出:
{
  "predicted_rate_low": 0.01,
  "predicted_rate_mid": 0.015,
  "predicted_rate_high": 0.02,
  "confidence": 0.7,
  "trend": "up|down|sideways",
  "reasoning": "基於以上步驟的推理說明"
}

注意：利率值使用日利率百分比格式（如 0.01 表示 0.01%%/天）`, string(featuresJSON))
}

// callOpenAI 調用 OpenAI API
func (s *LLMAIStrategy) callOpenAI(prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: s.config.OpenAIModel,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化請求失敗: %w", err)
	}

	url := s.config.OpenAIBaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("創建請求失敗: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.OpenAIAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("發送請求失敗: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("讀取響應失敗: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("解析響應失敗: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API 錯誤: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI 返回空響應")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// parseResponse 解析 AI 響應
func (s *LLMAIStrategy) parseResponse(response string) (*LLMPrediction, error) {
	// 嘗試從響應中提取 JSON
	response = strings.TrimSpace(response)

	// 處理 markdown 代碼塊
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// 嘗試找到 JSON 對象
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		response = response[startIdx : endIdx+1]
	}

	var prediction LLMPrediction
	if err := json.Unmarshal([]byte(response), &prediction); err != nil {
		return nil, fmt.Errorf("JSON 解析失敗: %w, 原始響應: %s", err, response)
	}

	// 驗證預測結果
	if prediction.PredictedRateMid <= 0 {
		return nil, fmt.Errorf("預測利率無效: %.4f", prediction.PredictedRateMid)
	}

	return &prediction, nil
}

// GetStrategyName 獲取策略名稱
func GetStrategyName(strategyType LLMStrategyType) string {
	switch strategyType {
	case StrategySimple:
		return "Simple (結構化數據 + 簡潔提示詞)"
	case StrategyDetailed:
		return "Detailed (完整市場深度 + 多維分析)"
	case StrategyChainOfThought:
		return "ChainOfThought (時序特徵 + 思維鏈推理)"
	default:
		return "Unknown"
	}
}
