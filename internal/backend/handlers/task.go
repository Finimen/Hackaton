package handlers

import (
	"net/http"
	"time"

	"NetScan/internal/backend/models"

	"github.com/gin-gonic/gin"
)

// GetNextTask возвращает следующую задачу для агента
func (h *Handlers) GetNextTask(c *gin.Context) {
	agent := h.getAgentFromContext(c)
	if agent == nil {
		return
	}

	task, err := h.queueService.GetNextTask(c.Request.Context(), agent.ID)
	if err != nil {
		h.logger.Error("failed to get next task", "error", err, "agent_id", agent.ID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("get_task_failed", "Failed to get next task"))
		return
	}

	if task == nil {
		c.JSON(http.StatusNoContent, SuccessResponse("no_tasks", nil))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("task_assigned", gin.H{
		"task": task,
	}))
}

// AckTask подтверждает выполнение задачи
func (h *Handlers) AckTask(c *gin.Context) {
	taskID := c.Param("task_id")
	agent := h.getAgentFromContext(c)
	if agent == nil {
		return
	}

	// TODO: Реализовать подтверждение задачи в queueService
	// Пока возвращаем заглушку

	h.logger.Info("task acknowledged", "task_id", taskID, "agent_id", agent.ID)
	c.JSON(http.StatusOK, SuccessResponse("task_acknowledged", gin.H{
		"task_id":   taskID,
		"agent_id":  agent.ID,
		"timestamp": time.Now(),
	}))
}

// отклоняет задачу
func (h *Handlers) NackTask(c *gin.Context) {
	taskID := c.Param("task_id")
	agent := h.getAgentFromContext(c)
	if agent == nil {
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
		Retry  bool   `json:"retry"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("invalid_request", "Invalid request body"))
		return
	}

	// TODO: Реализовать отклонение задачи в queueService

	h.logger.Warn("task rejected", "task_id", taskID, "agent_id", agent.ID, "reason", req.Reason)
	c.JSON(http.StatusOK, SuccessResponse("task_rejected", gin.H{
		"task_id":   taskID,
		"agent_id":  agent.ID,
		"reason":    req.Reason,
		"retry":     req.Retry,
		"timestamp": time.Now(),
	}))
}

// SubmitResult отправляет результат проверки
func (h *Handlers) SubmitResult(c *gin.Context) {
	checkID := c.Param("check_id")
	agent := h.getAgentFromContext(c)
	if agent == nil {
		return
	}

	var req struct {
		Success  bool                   `json:"success" binding:"required"`
		Data     map[string]interface{} `json:"data" binding:"required"`
		Error    string                 `json:"error,omitempty"`
		Duration float64                `json:"duration" binding:"required,min=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("invalid_request", "Invalid request body"))
		return
	}

	result := &models.CheckResult{
		CheckID:  checkID,
		AgentID:  agent.ID,
		Success:  req.Success,
		Data:     req.Data,
		Error:    req.Error,
		Duration: req.Duration,
	}

	if err := h.queueService.SubmitTaskResult(c.Request.Context(), result); err != nil {
		h.logger.Error("failed to submit result", "error", err, "check_id", checkID, "agent_id", agent.ID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("submit_failed", "Failed to submit result"))
		return
	}

	h.logger.Info("result submitted", "check_id", checkID, "agent_id", agent.ID, "success", req.Success)
	c.JSON(http.StatusOK, SuccessResponse("result_submitted", gin.H{
		"check_id":  checkID,
		"agent_id":  agent.ID,
		"timestamp": time.Now(),
	}))
}

// SubmitProgress отправляет прогресс выполнения
func (h *Handlers) SubmitProgress(c *gin.Context) {
	checkID := c.Param("check_id")
	agent := h.getAgentFromContext(c)
	if agent == nil {
		return
	}

	var req struct {
		Stage    string                 `json:"stage" binding:"required"`
		Progress float64                `json:"progress" binding:"required,min=0,max=1"`
		Data     map[string]interface{} `json:"data,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("invalid_request", "Invalid request body"))
		return
	}

	progress := &models.TaskProgress{
		CheckID:   checkID,
		AgentID:   agent.ID,
		Stage:     req.Stage,
		Progress:  req.Progress,
		Data:      req.Data,
		Timestamp: time.Now(),
	}

	if err := h.queueService.PublishTaskProgress(c.Request.Context(), progress); err != nil {
		h.logger.Error("failed to publish progress", "error", err, "check_id", checkID, "agent_id", agent.ID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("progress_failed", "Failed to submit progress"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("progress_submitted", nil))
}
