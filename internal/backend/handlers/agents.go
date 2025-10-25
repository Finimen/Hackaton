package handlers

import (
	"NetScan/internal/backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) RegisterAgent(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid register request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse("invalid_request", "Invalid request body"))
		return
	}

	agent, token, err := h.agentService.RegisterAgent(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to register agent", "error", err, "name", req.Name)
		c.JSON(http.StatusInternalServerError, ErrorResponse("registration_failed", err.Error()))
		return
	}

	h.logger.Info("agent registered", "agent_id", agent.ID, "name", agent.Name)
	c.JSON(http.StatusCreated, SuccessResponse("agent_registered", gin.H{
		"agent_id": agent.ID,
		"token":    token,
		"agent":    agent,
	}))
}

// аутентифицирует агента
func (h *Handlers) AuthenticateAgent(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("invalid_request", "Token is required"))
		return
	}

	agent, err := h.agentService.AuthenticateAgent(c.Request.Context(), req.Token)
	if err != nil {
		h.logger.Error("agent authentication failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse("authentication_failed", "Internal server error"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse("invalid_token", "Invalid agent token"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("authenticated", gin.H{
		"agent_id": agent.ID,
		"agent":    agent,
	}))
}

// обновляет статус агента
func (h *Handlers) Heartbeat(c *gin.Context) {
	agent := h.getAgentFromContext(c)
	if agent == nil {
		return
	}

	var req struct {
		Load int `json:"load" binding:"min=0,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("invalid_request", "Invalid request body"))
		return
	}

	if err := h.agentService.UpdateHeartbeat(c.Request.Context(), agent.ID, req.Load); err != nil {
		h.logger.Error("heartbeat update failed", "error", err, "agent_id", agent.ID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("heartbeat_failed", "Failed to update heartbeat"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("heartbeat_received", nil))
}

// возвращает список агентов
func (h *Handlers) ListAgents(c *gin.Context) {
	agents, err := h.agentService.ListOnlineAgents(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list agents", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse("list_failed", "Failed to list agents"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("agents_list", gin.H{
		"agents": agents,
		"count":  len(agents),
	}))
}

// возвращает информацию об агенте
func (h *Handlers) GetAgent(c *gin.Context) {
	agentID := c.Param("id")

	agent, err := h.agentService.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error("failed to get agent", "error", err, "agent_id", agentID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("get_failed", "Failed to get agent"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, ErrorResponse("not_found", "Agent not found"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("agent_found", gin.H{
		"agent": agent,
	}))
}

// возвращает статистику по агенту
func (h *Handlers) GetAgentStats(c *gin.Context) {
	agentID := c.Param("id")

	stats, err := h.agentService.GetAgentStats(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error("failed to get agent stats", "error", err, "agent_id", agentID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("stats_failed", "Failed to get agent stats"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("agent_stats", gin.H{
		"stats": stats,
	}))
}

// middleware для аутентификации агентов
func (h *Handlers) AgentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse("missing_token", "Authorization header is required"))
			c.Abort()
			return
		}

		// Убираем "Bearer " префикс если есть
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		agent, err := h.agentService.AuthenticateAgent(c.Request.Context(), token)
		if err != nil {
			h.logger.Error("agent auth failed", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse("auth_failed", "Authentication failed"))
			c.Abort()
			return
		}

		if agent == nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse("invalid_token", "Invalid agent token"))
			c.Abort()
			return
		}

		c.Set("agent", agent)
		c.Next()
	}
}

// возвращает агента из контекста
func (h *Handlers) getAgentFromContext(c *gin.Context) *models.Agent {
	agent, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse("unauthorized", "Agent not authenticated"))
		return nil
	}
	return agent.(*models.Agent)
}
