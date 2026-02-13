package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"kline-indicator-service/internal/models"
	"kline-indicator-service/internal/service"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

// WSHandler WebSocket处理器
type WSHandler struct {
	klineService     *service.KLineService
	indicatorService *service.IndicatorService
	clients          map[*websocket.Conn]*WSClient
	clientsMu        sync.RWMutex
	stopChan         chan struct{}
}

// WSClient WebSocket客户端
type WSClient struct {
	conn          *websocket.Conn
	subscriptions map[string]bool // 订阅的股票
	mu            sync.Mutex
}

// WSMessage WebSocket消息
type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// wsEnvelope 前端消息信封格式: {"type":"...", "data":{...}, "timestamp":"..."}
type wsEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// wsSubscribeData 订阅data字段内部结构: {"stocks":[...]}
type wsSubscribeData struct {
	Stocks []struct {
		Market    int    `json:"market"`
		Code      string `json:"code"`
		KLineType int    `json:"klinetype"`
	} `json:"stocks"`
}

// NewWSHandler 创建WebSocket处理器
func NewWSHandler(klineService *service.KLineService, indicatorService *service.IndicatorService) *WSHandler {
	handler := &WSHandler{
		klineService:     klineService,
		indicatorService: indicatorService,
		clients:          make(map[*websocket.Conn]*WSClient),
		stopChan:         make(chan struct{}),
	}

	// 启动心跳检测
	go handler.heartbeatLoop()
	// 启动指标推送循环
	go handler.updateLoop()

	return handler
}

// HandleKLine 处理K线WebSocket连接
func (h *WSHandler) HandleKLine(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	client := &WSClient{
		conn:          conn,
		subscriptions: make(map[string]bool),
	}

	h.clientsMu.Lock()
	h.clients[conn] = client
	h.clientsMu.Unlock()

	log.Printf("新WebSocket连接: %s", conn.RemoteAddr())

	// 发送欢迎消息
	h.sendMessage(client, &WSMessage{
		Type:      "connected",
		Data:      map[string]string{"message": "连接成功"},
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// 处理消息
	go h.handleMessages(client)
}

// handleMessages 处理客户端消息
func (h *WSHandler) handleMessages(client *WSClient) {
	defer func() {
		h.removeClient(client.conn)
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket读取错误: %v", err)
			}
			break
		}

		// 解析消息信封
		var env wsEnvelope
		if err := json.Unmarshal(message, &env); err != nil {
			h.sendError(client, "消息格式错误")
			continue
		}

		switch env.Type {
		case "ping":
			h.sendMessage(client, &WSMessage{
				Type:      "pong",
				Timestamp: time.Now().Format(time.RFC3339),
			})

		case "subscribe":
			h.handleSubscribe(client, env.Data)

		case "unsubscribe":
			h.handleUnsubscribe(client, env.Data)

		default:
			h.sendError(client, "未知的消息类型")
		}
	}
}

// handleSubscribe 处理订阅请求
// 前端发送格式: {"type":"subscribe","data":{"stocks":[{"market":0,"code":"000001","klinetype":10}]}}
func (h *WSHandler) handleSubscribe(client *WSClient, data json.RawMessage) {
	var subData wsSubscribeData
	if err := json.Unmarshal(data, &subData); err != nil {
		h.sendError(client, "订阅请求格式错误")
		return
	}

	client.mu.Lock()
	for _, stock := range subData.Stocks {
		key := buildSubscriptionKey(stock.Market, stock.Code, stock.KLineType)
		client.subscriptions[key] = true
	}
	client.mu.Unlock()

	log.Printf("WebSocket订阅: %d个股票", len(subData.Stocks))

	h.sendMessage(client, &WSMessage{
		Type:      "subscribed",
		Data:      map[string]int{"count": len(subData.Stocks)},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// handleUnsubscribe 处理取消订阅请求
func (h *WSHandler) handleUnsubscribe(client *WSClient, data json.RawMessage) {
	var subData wsSubscribeData
	if err := json.Unmarshal(data, &subData); err != nil {
		h.sendError(client, "取消订阅请求格式错误")
		return
	}

	client.mu.Lock()
	for _, stock := range subData.Stocks {
		key := buildSubscriptionKey(stock.Market, stock.Code, stock.KLineType)
		delete(client.subscriptions, key)
	}
	client.mu.Unlock()

	h.sendMessage(client, &WSMessage{
		Type:      "unsubscribed",
		Data:      map[string]int{"count": len(subData.Stocks)},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// sendMessage 发送消息
func (h *WSHandler) sendMessage(client *WSClient, msg *WSMessage) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if err := client.conn.WriteJSON(msg); err != nil {
		log.Printf("发送WebSocket消息失败: %v", err)
	}
}

// sendError 发送错误消息
func (h *WSHandler) sendError(client *WSClient, errMsg string) {
	h.sendMessage(client, &WSMessage{
		Type:      "error",
		Data:      map[string]string{"message": errMsg},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// removeClient 移除客户端
func (h *WSHandler) removeClient(conn *websocket.Conn) {
	h.clientsMu.Lock()
	delete(h.clients, conn)
	h.clientsMu.Unlock()
	log.Printf("WebSocket连接断开: %s", conn.RemoteAddr())
}

// heartbeatLoop 心跳检测循环
func (h *WSHandler) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.clientsMu.RLock()
			for _, client := range h.clients {
				h.sendMessage(client, &WSMessage{
					Type:      "heartbeat",
					Timestamp: time.Now().Format(time.RFC3339),
				})
			}
			h.clientsMu.RUnlock()
		case <-h.stopChan:
			return
		}
	}
}

// updateLoop 定时轮询并推送指标更新（仅在交易时间内推送）
func (h *WSHandler) updateLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if isTradingTime(time.Now()) {
				h.pushIndicatorUpdates()
			}
		case <-h.stopChan:
			return
		}
	}
}

// isTradingTime 判断给定时间是否处于A股交易时段
// 交易时间：周一至周五 9:30-11:30, 13:00-15:00（Asia/Shanghai时区）
func isTradingTime(now time.Time) bool {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 时区加载失败时回退为允许推送，避免功能中断
		return true
	}
	now = now.In(loc)

	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hhmm := now.Hour()*60 + now.Minute()
	// 上午 9:30 (570) ~ 11:30 (690)
	// 下午 13:00 (780) ~ 15:00 (900)
	if (hhmm >= 570 && hhmm < 690) || (hhmm >= 780 && hhmm < 900) {
		return true
	}
	return false
}

// pushIndicatorUpdates 收集所有订阅key并逐一推送指标更新
func (h *WSHandler) pushIndicatorUpdates() {
	// 收集所有唯一的订阅key
	uniqueKeys := make(map[string]bool)
	h.clientsMu.RLock()
	for _, client := range h.clients {
		client.mu.Lock()
		for key := range client.subscriptions {
			uniqueKeys[key] = true
		}
		client.mu.Unlock()
	}
	h.clientsMu.RUnlock()

	if len(uniqueKeys) == 0 {
		return
	}

	for key := range uniqueKeys {
		market, code, klineType, err := parseSubscriptionKey(key)
		if err != nil {
			log.Printf("解析订阅key失败: %s, %v", key, err)
			continue
		}
		h.checkAndPushUpdates(market, code, klineType)
	}
}

// checkAndPushUpdates 检查K线更新并推送指标计算结果
func (h *WSHandler) checkAndPushUpdates(market int, code string, klineType int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("指标推送panic: market=%d, code=%s, klineType=%d, err=%v", market, code, klineType, r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 使缓存失效，强制获取最新数据
	klineReq := &models.KLineRequest{
		Market:    market,
		Code:      code,
		KLineType: models.KLineType(klineType),
		Weight:    models.WeightNone,
		Count:     2000,
	}
	_ = h.klineService.InvalidateCache(ctx, klineReq)

	// 构建指标计算请求
	calcReq := &models.IndicatorCalculateRequest{
		Market:    market,
		Code:      code,
		KLineType: models.KLineType(klineType),
		Weight:    models.WeightNone,
		Count:     2000,
		Indicators: []models.IndicatorConfig{
			{
				Type: models.IndicatorTypeMoshiChanlun,
				Params: map[string]interface{}{
					"kline_type":        klineType,
					"show_level_sub_x1": true,
					"show_level_1x":     true,
					"show_level_2x":     true,
					"show_level_4x":     true,
					"show_level_8x":     true,
				},
			},
		},
	}

	result, err := h.indicatorService.Calculate(ctx, calcReq)
	if err != nil {
		log.Printf("WebSocket指标计算失败: market=%d, code=%s, err=%v", market, code, err)
		return
	}

	// 获取最新K线数据（Calculate已触发缓存回填，此处命中缓存）
	klineData, _, err := h.klineService.GetKLineData(ctx, klineReq)
	if err != nil {
		log.Printf("WebSocket获取K线数据失败: market=%d, code=%s, err=%v", market, code, err)
		return
	}

	// 广播K线+指标更新给订阅该股票的客户端
	h.broadcastIndicatorUpdate(market, code, klineType, klineData, result.Indicators)
}

// broadcastIndicatorUpdate 向订阅客户端广播K线+指标更新
func (h *WSHandler) broadcastIndicatorUpdate(market int, code string, klineType int, klineData *models.KLineData, indicators []models.IndicatorResult) {
	key := buildSubscriptionKey(market, code, klineType)

	msg := &WSMessage{
		Type: "indicator_update",
		Data: map[string]interface{}{
			"market":     market,
			"code":       code,
			"klinetype":  klineType,
			"klines":     klineData.KLines,
			"indicators": indicators,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 先收集目标客户端，释放锁后再发送
	h.clientsMu.RLock()
	targetClients := make([]*WSClient, 0)
	for _, client := range h.clients {
		client.mu.Lock()
		if client.subscriptions[key] {
			targetClients = append(targetClients, client)
		}
		client.mu.Unlock()
	}
	h.clientsMu.RUnlock()

	for _, client := range targetClients {
		h.sendMessage(client, msg)
	}

	if len(targetClients) > 0 {
		log.Printf("WebSocket推送指标更新: %s -> %d个客户端", key, len(targetClients))
	}
}

// BroadcastKLineUpdate 广播K线更新
func (h *WSHandler) BroadcastKLineUpdate(market int, code string, klineType int, kline *models.KLine) {
	key := buildSubscriptionKey(market, code, klineType)

	msg := &WSMessage{
		Type: "kline_update",
		Data: map[string]interface{}{
			"market":    market,
			"code":      code,
			"klinetype": klineType,
			"kline":     kline,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	h.clientsMu.RLock()
	targetClients := make([]*WSClient, 0)
	for _, client := range h.clients {
		client.mu.Lock()
		if client.subscriptions[key] {
			targetClients = append(targetClients, client)
		}
		client.mu.Unlock()
	}
	h.clientsMu.RUnlock()

	for _, client := range targetClients {
		h.sendMessage(client, msg)
	}
}

// GetClientCount 获取客户端数量
func (h *WSHandler) GetClientCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// Close 关闭WSHandler，停止后台goroutine
func (h *WSHandler) Close() {
	close(h.stopChan)
}

// buildSubscriptionKey 构建订阅key
func buildSubscriptionKey(market int, code string, klineType int) string {
	return fmt.Sprintf("%d:%s:%d", market, code, klineType)
}

// parseSubscriptionKey 解析订阅key为market/code/klineType
func parseSubscriptionKey(key string) (market int, code string, klineType int, err error) {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) != 3 {
		return 0, "", 0, fmt.Errorf("invalid key format: %s", key)
	}
	market, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", 0, fmt.Errorf("invalid market in key: %s", key)
	}
	code = parts[1]
	klineType, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, "", 0, fmt.Errorf("invalid klineType in key: %s", key)
	}
	return market, code, klineType, nil
}
