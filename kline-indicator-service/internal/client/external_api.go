package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"kline-indicator-service/internal/models"
)

// ExternalAPIClient 外部K线API客户端
type ExternalAPIClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	reqIDMu    sync.Mutex
	reqID      int
}

// NewExternalAPIClient 创建外部API客户端
func NewExternalAPIClient(baseURL string, timeout time.Duration, maxRetries int) *ExternalAPIClient {
	return &ExternalAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		timeout:    timeout,
		maxRetries: maxRetries,
		reqID:      1,
	}
}

// getNextReqID 获取下一个请求ID
func (c *ExternalAPIClient) getNextReqID() int {
	c.reqIDMu.Lock()
	defer c.reqIDMu.Unlock()
	c.reqID++
	return c.reqID
}

// FetchKLine 获取K线数据
func (c *ExternalAPIClient) FetchKLine(ctx context.Context, req *models.KLineRequest) (*models.KLineData, error) {
	// 构造外部API请求
	apiReq := c.buildRequest(req)
	
	var lastErr error
	for i := 0; i <= c.maxRetries; i++ {
		resp, err := c.doRequest(ctx, apiReq)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(1<<uint(i)) * 100 * time.Millisecond) // 指数退避: 100ms, 200ms, 400ms...
			continue
		}
		
		// 解析响应
		klineData, err := c.parseResponse(resp, req)
		if err != nil {
			lastErr = err
			continue
		}
		
		return klineData, nil
	}
	
	return nil, fmt.Errorf("获取K线数据失败，已重试%d次: %w", c.maxRetries, lastErr)
}

// buildRequest 构建外部API请求
func (c *ExternalAPIClient) buildRequest(req *models.KLineRequest) *models.ExternalAPIRequest {
	// 计算结束时间
	endTime := req.EndTime
	if endTime == "" {
		endTime = time.Now().Format("2006-1-2 15:04:05")
	}
	
	return &models.ExternalAPIRequest{
		ReqType: 150,
		ReqID:   c.getNextReqID(),
		Session: "",
		Data: models.ExternalAPIRequestData{
			Market:    req.Market,
			Code:      req.Code,
			KLineType: int(req.KLineType),
			Weight:    int(req.Weight),
			TimeType:  2, // 往前count条
			Time0:     endTime,
			Count:     req.Count,
		},
	}
}

// doRequest 执行HTTP请求
func (c *ExternalAPIClient) doRequest(ctx context.Context, apiReq *models.ExternalAPIRequest) (*models.ExternalAPIResponse, error) {
	// 序列化请求
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	
	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	
	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer httpResp.Body.Close()
	
	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	
	// 检查HTTP状态码
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP状态码错误: %d, body: %s", httpResp.StatusCode, string(respBody))
	}
	
	// 解析响应
	var apiResp models.ExternalAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	
	// 检查业务状态码
	if apiResp.Status != 0 {
		return nil, fmt.Errorf("API返回错误: status=%d, msg=%s", apiResp.Status, apiResp.Msg)
	}
	
	return &apiResp, nil
}

// parseResponse 解析API响应为KLineData
func (c *ExternalAPIClient) parseResponse(resp *models.ExternalAPIResponse, req *models.KLineRequest) (*models.KLineData, error) {
	data := resp.Data
	if len(data.KList) == 0 {
		return nil, fmt.Errorf("响应数据为空")
	}

	klines := make([]models.KLine, 0, len(data.KList)-1)
	
	// 跳过表头（第一行是字段名）
	for i := 1; i < len(data.KList); i++ {
		row := data.KList[i]
		if len(row) < 6 {
			continue
		}
		
		// 解析时间
		timeStr, ok := row[0].(string)
		if !ok {
			continue
		}
		
		// 解析OHLCV
		kline := models.KLine{
			Timestamp: timeStr,
			Open:      c.toFloat64(row[1]),
			High:      c.toFloat64(row[2]),
			Low:       c.toFloat64(row[3]),
			Close:     c.toFloat64(row[4]),
			Volume:    c.toFloat64(row[5]),
		}
		
		// 解析成交额（如果有）
		if len(row) > 6 {
			kline.Amount = c.toFloat64(row[6])
		}
		
		klines = append(klines, kline)
	}
	
	return &models.KLineData{
		Market:    data.Market,
		Code:      data.Code,
		KLineType: models.KLineType(data.KLineType),
		Weight:    models.WeightType(data.Weight),
		KLines:    klines,
		Count:     len(klines),
	}, nil
}

// toFloat64 转换为float64
func (c *ExternalAPIClient) toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

// SetBaseURL 设置基础URL
func (c *ExternalAPIClient) SetBaseURL(url string) {
	c.baseURL = url
}

// Health 检查外部API健康状态
func (c *ExternalAPIClient) Health(ctx context.Context) error {
	// 发送一个简单的请求检查连接
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	req := &models.KLineRequest{
		Market:    0,
		Code:      "000001",
		KLineType: models.KLineTypeDay,
		Weight:    models.WeightNone,
		Count:     1,
	}
	
	_, err := c.FetchKLine(ctx, req)
	return err
}
