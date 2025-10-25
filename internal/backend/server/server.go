package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"NetScan/internal/backend/dependencies"
	"NetScan/internal/backend/handlers"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router     *gin.Engine
	config     *Config
	container  *dependencies.Container
	handlers   *handlers.Handlers
	httpServer *http.Server
}

type Config struct {
	Port int
	Mode string
}

// New создает сервер с dependency injection
func New(config *Config, container *dependencies.Container) *Server {
	if config.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	server := &Server{
		router:    gin.New(),
		config:    config,
		container: container,
		handlers:  handlers.NewHandlers(container),
	}

	server.setupMiddlewares()
	server.setupRoutes()

	return server
}

func (s *Server) setupMiddlewares() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(s.loggerMiddleware())

	// CORS middleware
	s.router.Use(s.corsMiddleware())

	// Request ID middleware
	s.router.Use(s.requestIDMiddleware())
}

func (s *Server) setupRoutes() {
	// Health checks
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/ready", s.readyCheck)

	// API v1 group
	api := s.router.Group("/api/v1")
	{
		// Agents routes
		agents := api.Group("/agents")
		{
			agents.POST("/register", s.handlers.RegisterAgent)
			agents.POST("/auth", s.handlers.AuthenticateAgent)
			agents.POST("/heartbeat", s.handlers.AgentAuthMiddleware(), s.handlers.Heartbeat)
			agents.GET("", s.handlers.ListAgents)
			agents.GET("/:id", s.handlers.GetAgent)
			agents.GET("/:id/stats", s.handlers.GetAgentStats)
		}

		// Checks routes
		checks := api.Group("/checks")
		{
			checks.POST("", s.handlers.CreateCheck)
			checks.GET("/:id", s.handlers.GetCheck)
			checks.GET("/:id/results", s.handlers.GetCheckResults)
			checks.GET("/:id/stats", s.handlers.GetCheckStats)
			checks.GET("", s.handlers.ListChecks)
		}

		// Tasks routes (для агентов)
		tasks := api.Group("/tasks")
		tasks.Use(s.handlers.AgentAuthMiddleware())
		{
			tasks.GET("/next", s.handlers.GetNextTask)
			tasks.POST("/:task_id/ack", s.handlers.AckTask)
			tasks.POST("/:task_id/nack", s.handlers.NackTask)
		}

		// Results routes (для агентов)
		results := api.Group("/results")
		results.Use(s.handlers.AgentAuthMiddleware())
		{
			results.POST("/:check_id", s.handlers.SubmitResult)
			results.POST("/:check_id/progress", s.handlers.SubmitProgress)
		}

		// Metrics routes
		//metrics := api.Group("/metrics")
		//{
		//	metrics.GET("", s.handlers.GetMetrics)
		//	metrics.GET("/queue", s.handlers.GetQueueStats)
		//	metrics.GET("/health", s.handlers.GetSystemHealth)
		//	metrics.POST("/cleanup", s.handlers.CleanupStuckTasks)
		//}
	}

	// WebSocket routes
	ws := s.router.Group("/ws")
	{
		ws.GET("/checks/:id", s.handlers.CheckWebSocket)
		ws.GET("/agents", s.handlers.AgentsWebSocket)
	}

	// 404 handler
	s.router.NoRoute(s.notFoundHandler)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "checkmesh-backend",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
	})
}

func (s *Server) readyCheck(c *gin.Context) {
	// Проверяем подключение к БД
	if s.container.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "Database not connected",
		})
		return
	}

	// Проверяем подключение к Redis
	if s.container.Queue == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "Redis not connected",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"database":  "connected",
		"redis":     "connected",
		"timestamp": time.Now().UTC(),
	})
}

func (s *Server) notFoundHandler(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error":   "not_found",
		"message": "Endpoint not found",
		"path":    c.Request.URL.Path,
	})
}

func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Продолжаем обработку
		c.Next()

		// Логируем после обработки
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if query != "" {
			path = path + "?" + query
		}

		logger := slog.Info
		if statusCode >= 400 {
			logger = slog.Warn
		}
		if statusCode >= 500 {
			logger = slog.Error
		}

		logger("HTTP request",
			"status", statusCode,
			"method", method,
			"path", path,
			"ip", clientIP,
			"latency", latency,
			"error", errorMessage,
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// Start запускает HTTP сервер
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Starting HTTP server",
		"port", s.config.Port,
		"mode", s.config.Mode,
		"address", addr,
	)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown выполняет graceful shutdown сервера
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down HTTP server...")

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	}

	if s.container != nil {
		if err := s.container.Close(); err != nil {
			slog.Error("Failed to close dependencies", "error", err)
		}
	}

	slog.Info("Server shutdown completed")
	return nil
}

// GetRouter возвращает router для тестирования
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
