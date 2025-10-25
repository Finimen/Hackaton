package handlers

import (
	"NetScan/internal/backend/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateCheck создает новую проверку
func (h *Handlers) CreateCheck(c *gin.Context) {
	var req struct {
		Type   models.CheckType `json:"type" binding:"required"`
		Target string           `json:"target" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("invalid_request", "Type and target are required"))
		return
	}

	check, err := h.checkService.CreateCheck(c.Request.Context(), req.Type, req.Target)
	if err != nil {
		h.logger.Error("failed to create check", "error", err, "type", req.Type, "target", req.Target)
		c.JSON(http.StatusInternalServerError, ErrorResponse("create_failed", err.Error()))
		return
	}

	h.logger.Info("check created", "check_id", check.ID, "type", req.Type, "target", req.Target)
	c.JSON(http.StatusCreated, SuccessResponse("check_created", gin.H{
		"check_id": check.ID,
		"check":    check,
	}))
}

// GetCheck возвращает информацию о проверке
func (h *Handlers) GetCheck(c *gin.Context) {
	checkID := c.Param("id")

	checkWithResults, err := h.checkService.GetCheckByID(c.Request.Context(), checkID)
	if err != nil {
		h.logger.Error("failed to get check", "error", err, "check_id", checkID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("get_failed", "Failed to get check"))
		return
	}

	if checkWithResults == nil {
		c.JSON(http.StatusNotFound, ErrorResponse("not_found", "Check not found"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("check_found", gin.H{
		"check": checkWithResults,
	}))
}

// GetCheckResults возвращает результаты проверки
func (h *Handlers) GetCheckResults(c *gin.Context) {
	checkID := c.Param("id")

	checkWithResults, err := h.checkService.GetCheckByID(c.Request.Context(), checkID)
	if err != nil {
		h.logger.Error("failed to get check results", "error", err, "check_id", checkID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("get_failed", "Failed to get check results"))
		return
	}

	if checkWithResults == nil {
		c.JSON(http.StatusNotFound, ErrorResponse("not_found", "Check not found"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("results_found", gin.H{
		"check_id": checkID,
		"results":  checkWithResults.Results,
		"count":    len(checkWithResults.Results),
	}))
}

// ListChecks возвращает список проверок
func (h *Handlers) ListChecks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Валидация параметров
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	checks, err := h.checkService.ListChecks(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("failed to list checks", "error", err, "limit", limit, "offset", offset)
		c.JSON(http.StatusInternalServerError, ErrorResponse("list_failed", "Failed to list checks"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("checks_list", gin.H{
		"checks": checks,
		"count":  len(checks),
		"limit":  limit,
		"offset": offset,
	}))
}

// GetCheckStats возвращает статистику по проверке
func (h *Handlers) GetCheckStats(c *gin.Context) {
	checkID := c.Param("id")

	stats, err := h.checkService.GetCheckStats(c.Request.Context(), checkID)
	if err != nil {
		h.logger.Error("failed to get check stats", "error", err, "check_id", checkID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("stats_failed", "Failed to get check stats"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("check_stats", gin.H{
		"check_id": checkID,
		"stats":    stats,
	}))
}
