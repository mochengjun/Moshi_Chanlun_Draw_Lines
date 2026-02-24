package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kline-indicator-service/internal/models"
	"kline-indicator-service/internal/service"
)

// IndicatorHandler 指标计算处理器
type IndicatorHandler struct {
	indicatorService *service.IndicatorService
}

// NewIndicatorHandler 创建指标处理器
func NewIndicatorHandler(indicatorService *service.IndicatorService) *IndicatorHandler {
	return &IndicatorHandler{
		indicatorService: indicatorService,
	}
}

// Calculate 计算指标
// @Summary 计算技术指标
// @Description 根据K线数据计算多种技术指标
// @Tags indicator
// @Accept json
// @Produce json
// @Param request body models.IndicatorCalculateRequest true "指标计算请求"
// @Success 200 {object} models.IndicatorCalculateResponse
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/indicators/calculate [post]
func (h *IndicatorHandler) Calculate(c *gin.Context) {
	var req models.IndicatorCalculateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			models.ErrCodeBadRequest,
			models.ErrMsgBadRequest+": "+err.Error(),
		))
		return
	}
	
	// 参数验证
	if req.Code == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			models.ErrCodeBadRequest,
			"股票代码不能为空",
		))
		return
	}
	
	if len(req.Indicators) == 0 {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			models.ErrCodeBadRequest,
			"至少需要指定一个指标",
		))
		return
	}
	
	if req.Count <= 0 || req.Count > 5000 {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			models.ErrCodeBadRequest,
			"count必须在1到5000之间",
		))
		return
	}
	
	// 计算指标
	result, err := h.indicatorService.Calculate(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			models.ErrCodeInternalError,
			err.Error(),
		))
		return
	}
	
	c.JSON(http.StatusOK, result)
}

// ListIndicators 列出所有指标
// @Summary 获取指标列表
// @Description 获取系统支持的所有技术指标
// @Tags indicator
// @Accept json
// @Produce json
// @Success 200 {object} models.IndicatorListResponse
// @Router /api/v1/indicators/list [get]
func (h *IndicatorHandler) ListIndicators(c *gin.Context) {
	indicators := h.indicatorService.ListIndicators()
	
	c.JSON(http.StatusOK, &models.IndicatorListResponse{
		Code:       models.ErrCodeSuccess,
		Message:    "success",
		Indicators: indicators,
	})
}
