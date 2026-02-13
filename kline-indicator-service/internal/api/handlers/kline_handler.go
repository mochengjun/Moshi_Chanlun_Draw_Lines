package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kline-indicator-service/internal/models"
	"kline-indicator-service/internal/service"
)

// KLineHandler K线数据处理器
type KLineHandler struct {
	klineService *service.KLineService
}

// NewKLineHandler 创建K线处理器
func NewKLineHandler(klineService *service.KLineService) *KLineHandler {
	return &KLineHandler{
		klineService: klineService,
	}
}

// GetKLine 获取K线数据
// @Summary 获取K线数据
// @Description 根据股票代码和参数获取K线数据
// @Tags kline
// @Accept json
// @Produce json
// @Param market query int true "市场代码 (0=深市, 1=沪市)"
// @Param code query string true "股票代码"
// @Param klinetype query int true "K线类型 (1=1分钟, 10=日K, 11=周K等)"
// @Param weight query int false "复权方式 (0=不复权, 1=前复权, 2=后复权)"
// @Param count query int true "数据条数"
// @Param endtime query string false "结束时间"
// @Success 200 {object} models.KLineResponse
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/kline [get]
func (h *KLineHandler) GetKLine(c *gin.Context) {
	var req models.KLineRequest
	if err := c.ShouldBindQuery(&req); err != nil {
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
	
	if req.Count <= 0 || req.Count > 5000 {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			models.ErrCodeBadRequest,
			"count必须在1到5000之间",
		))
		return
	}
	
	// 获取K线数据
	klineData, cacheHit, err := h.klineService.GetKLineData(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			models.ErrCodeExternalAPIError,
			err.Error(),
		))
		return
	}
	
	// 设置响应头
	if cacheHit {
		c.Header("X-Cache-Hit", "true")
	} else {
		c.Header("X-Cache-Hit", "false")
	}
	
	c.JSON(http.StatusOK, &models.KLineResponse{
		Code:     models.ErrCodeSuccess,
		Message:  "success",
		Data:     klineData,
		CacheHit: cacheHit,
	})
}
