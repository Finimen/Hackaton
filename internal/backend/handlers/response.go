package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
)

// создает успешный JSON ответ
func SuccessResponse(message string, data interface{}) gin.H {
	response := gin.H{
		"success":   true,
		"message":   message,
		"timestamp": time.Now().UTC(),
	}

	if data != nil {
		response["data"] = data
	}

	return response
}

// создает JSON ответ с ошибкой
func ErrorResponse(code string, message string) gin.H {
	return gin.H{
		"success":   false,
		"error":     code,
		"message":   message,
		"timestamp": time.Now().UTC(),
	}
}

// PaginatedResponse создает ответ с пагинацией
func PaginatedResponse(message string, data interface{}, total int, limit int, offset int) gin.H {
	return gin.H{
		"success": true,
		"message": message,
		"data":    data,
		"pagination": gin.H{
			"total":    total,
			"limit":    limit,
			"offset":   offset,
			"has_more": offset+len(data.([]interface{})) < total,
		},
		"timestamp": time.Now().UTC(),
	}
}
