package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшене нужно ограничить домены
	},
}

// CheckWebSocket WebSocket для отслеживания проверки
func (h *Handlers) CheckWebSocket(c *gin.Context) {
	checkID := c.Param("id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade to websocket", "error", err, "check_id", checkID)
		c.JSON(http.StatusInternalServerError, ErrorResponse("websocket_failed", "Failed to establish WebSocket connection"))
		return
	}
	defer conn.Close()

	h.logger.Info("websocket connected for check", "check_id", checkID)

	// TODO: Реализовать подписку на обновления проверки
	// Пока отправляем заглушку
	conn.WriteJSON(SuccessResponse("connected", gin.H{
		"check_id": checkID,
		"message":  "WebSocket connected, real-time updates not implemented yet",
	}))

	// Держим соединение открытым
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			h.logger.Debug("websocket disconnected", "check_id", checkID, "error", err)
			break
		}

		// Эхо-ответ для тестирования
		if err := conn.WriteMessage(messageType, p); err != nil {
			h.logger.Debug("websocket write error", "check_id", checkID, "error", err)
			break
		}
	}
}

// AgentsWebSocket WebSocket для отслеживания статуса агентов
func (h *Handlers) AgentsWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade to websocket", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse("websocket_failed", "Failed to establish WebSocket connection"))
		return
	}
	defer conn.Close()

	h.logger.Info("websocket connected for agents monitoring")

	// TODO: Реализовать подписку на обновления агентов
	conn.WriteJSON(SuccessResponse("connected", gin.H{
		"message": "Agents WebSocket connected, real-time updates not implemented yet",
	}))

	// Держим соединение открытым
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.logger.Debug("agents websocket disconnected", "error", err)
			break
		}
	}
}
