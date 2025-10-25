package models

import "time"

type QueueStats struct {
	// Основные метрики очереди
	QueueLength  int64 `json:"queue_length"`  // Количество задач в очереди Redis
	OnlineAgents int   `json:"online_agents"` // Количество онлайн агентов
	ActiveTasks  int   `json:"active_tasks"`  // Количество взятых на выполнение задач

	// Статистика по проверкам
	PendingChecks   int `json:"pending_checks"`   // Ожидающие проверки
	RunningChecks   int `json:"running_checks"`   // Выполняющиеся проверки
	CompletedChecks int `json:"completed_checks"` // Завершенные проверки
	FailedChecks    int `json:"failed_checks"`    // Проваленные проверки

	// Производительность
	QueueThroughput float64 `json:"queue_throughput"` // Пропускная способность (проверок/мин)
	AvgWaitTime     float64 `json:"avg_wait_time"`    // Среднее время ожидания в секундах
	SuccessRate     float64 `json:"success_rate"`     // Процент успешных проверок

	// Системные метрики
	StuckTasks      int     `json:"stuck_tasks"`       // Количество зависших задач
	AvgResponseTime float64 `json:"avg_response_time"` // Среднее время ответа (сек)

	// Временные метки
	Timestamp time.Time `json:"timestamp"` // Время сбора статистики
	Uptime    string    `json:"uptime"`    // Время работы системы
}

type TaskProgress struct {
	CheckID   string                 `json:"check_id"`
	AgentID   string                 `json:"agent_id"`
	Stage     string                 `json:"stage"`
	Progress  float64                `json:"progress"` // 0.0 - 1.0
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type ResultNotification struct {
	CheckID   string    `json:"check_id"`
	AgentID   string    `json:"agent_id"`
	Success   bool      `json:"success"`
	Duration  float64   `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}
