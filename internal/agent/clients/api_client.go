package client

import (
	domain "NetScan/internal/agent/domain"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type APIClient struct {
	baseURL    string
	token      string
	agentID    string
	httpClient *http.Client
	cbState    circuitBreakerState
	metrics    APIClientMetrics
}

type circuitBreakerState struct {
	failures    int
	lastFailure time.Time
	state       string // "closed", "open", "half-open"
	mutex       sync.RWMutex
}

type APIClientMetrics interface {
	ObserveRequestDuration(method string, statusCode int, duration time.Duration)
	IncRequestCounter(method string, statusCode int)
	IncErrorCounter(method string, errorType string)
}

func NewAPIClient(baseURL, token, agentID string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		agentID: agentID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
}

func (a *APIClient) GetAgentID() string {
	return a.agentID
}

// FetchTask - From Backend to Agent
func (a *APIClient) FetchTask(ctx context.Context) (*domain.Task, error) {
	const maxRetries = 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		task, err := a.fetchTaskSingle(ctx)
		if err == nil {
			return task, nil
		}

		if errors.Is(err, ErrNoTasks) || errors.Is(err, ErrNotRegistered) {
			return nil, err
		}

		lastErr = err

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second * time.Duration(i*i)):
			continue
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (a *APIClient) fetchTaskSingle(ctx context.Context) (*domain.Task, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/api/v1/tasks/next", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("X-Agent-ID", a.agentID)

	fmt.Printf("🔐 DEBUG: Auth - Token present: %t, AgentID: %s\n", a.token != "", a.agentID)

	fmt.Printf("🔍 DEBUG: Fetching task from: %s\n", a.baseURL+"/api/v1/tasks/next")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackendDown, err)
	}
	defer resp.Body.Close()

	fmt.Printf("🔍 DEBUG: Task response status: %d\n", resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Printf("🔍 DEBUG: No tasks available\n")
		return nil, ErrNoTasks
	case http.StatusOK:
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("🔍 DEBUG: Raw task response: %s\n", string(body))

		var response struct {
			Data struct {
				Task struct {
					CheckID   string    `json:"check_id"`   // это будет ID задачи
					Type      string    `json:"type"`       // это соответствует Type
					Target    string    `json:"target"`     // это соответствует Target
					CreatedAt time.Time `json:"created_at"` // это соответствует CreatedAt
				} `json:"task"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode task response: %w", err)
		}

		// ПРАВИЛЬНОЕ СОЗДАНИЕ TASK С СООТВЕТСТВИЕМ ПОЛЕЙ
		task := &domain.Task{
			ID:        response.Data.Task.CheckID,                // check_id -> ID
			Type:      domain.CheckType(response.Data.Task.Type), // string -> CheckType
			Target:    response.Data.Task.Target,                 // target -> Target
			Options:   make(map[string]interface{}),              // пустые опции по умолчанию
			CreatedAt: response.Data.Task.CreatedAt,              // created_at -> CreatedAt
			AgentID:   a.agentID,                                 // добавляем agent_id
		}

		fmt.Printf("🔍 DEBUG: Parsed task: ID=%s, Type=%s, Target=%s, AgentID=%s\n",
			task.ID, task.Type, task.Target, task.AgentID)
		return task, nil
	case http.StatusUnauthorized:
		return nil, ErrNotRegistered
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
}

// SubmitResult - From Agent to Backend
func (a *APIClient) SubmitResult(ctx context.Context, result *domain.Result) error {
	return a.withCircuitBreaker(ctx, func() error {
		data := result.Data
		if data == nil {
			data = make(map[string]interface{})
		}

		backendResult := map[string]interface{}{
			"check_id":   result.TaskID, // task_id -> check_id
			"agent_id":   result.AgentID,
			"success":    result.Success,
			"data":       data,
			"error":      result.Error,
			"duration":   result.ResponseTime, // response_time -> duration
			"created_at": result.Timestamp,    // timestamp -> created_at
		}

		body, err := json.Marshal(backendResult)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}

		fmt.Printf("🔍 DEBUG: Submitting result body: %s\n", string(body))
		fmt.Printf("🔍 DEBUG: Submitting result to: %s\n", a.baseURL+"/api/v1/results/"+result.TaskID)

		req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/v1/results/"+result.TaskID, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+a.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Agent-ID", a.agentID)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrBackendDown, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			errorBody, _ := io.ReadAll(resp.Body)
			fmt.Printf("🔍 DEBUG: Result submission error response: %s\n", string(errorBody))
		} else {
			fmt.Printf("✅ DEBUG: Result submitted successfully!\n")
		}

		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated:
			return nil
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: invalid token", ErrNotRegistered)
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limit exceeded")
		case http.StatusRequestTimeout:
			return fmt.Errorf("request timeout")
		case http.StatusServiceUnavailable:
			return fmt.Errorf("service temporarily unavailable")
		default:
			var errorResp struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error != "" {
				return fmt.Errorf("submit failed %d: %s", resp.StatusCode, errorResp.Error)
			}
			return fmt.Errorf("submit failed with status %d", resp.StatusCode)
		}
	})
}

func (a *APIClient) RegisterAgent(ctx context.Context, agent *domain.Agent) error {
	const maxRetries = 3

	for i := 0; i < maxRetries; i++ {
		err := a.registerAgentSingle(ctx, agent)
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) ||
			strings.Contains(err.Error(), "validation") {
			return err
		}

		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(i+1) * time.Second):
				continue
			}
		}

		return fmt.Errorf("registration failed after %d attempts: %w", maxRetries, err)
	}

	return nil
}

func (a *APIClient) registerAgentSingle(ctx context.Context, agent *domain.Agent) error {
	body, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/v1/agents/register", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("backend unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// ИСПРАВЛЕННАЯ СТРУКТУРА - с data оберткой
	var response struct {
		Data struct {
			Token   string `json:"token"`
			AgentID string `json:"agent_id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	// Проверяем что поля не пустые
	if response.Data.AgentID == "" {
		return fmt.Errorf("server returned empty agent_id")
	}
	if response.Data.Token == "" {
		return fmt.Errorf("server returned empty token")
	}

	a.token = response.Data.Token
	a.agentID = response.Data.AgentID

	fmt.Printf("✅ DEBUG: Registration successful - AgentID: %s\n", a.agentID)

	return nil
}

// SendHeartbeat - sending heartbeat for monitoring of the agent activity
func (a *APIClient) SendHeartbeat(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// ПРАВИЛЬНАЯ структура для бэкенда
	heartbeatData := map[string]interface{}{
		"load": 0, // ← обязательное поле типа int от 0 до 100
	}

	body, err := json.Marshal(heartbeatData)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	fmt.Printf("🚀 DEBUG: Sending heartbeat with body: %s\n", string(body))

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/v1/agents/heartbeat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("X-Agent-ID", a.agentID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat received status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ DEBUG: Heartbeat successful\n")
	return nil
}

func (a *APIClient) withCircuitBreaker(ctx context.Context, operation func() error) error {
	a.cbState.mutex.RLock()

	if a.cbState.state == "open" && time.Since(a.cbState.lastFailure) < 30*time.Second {
		a.cbState.mutex.RUnlock()
		return fmt.Errorf("circuit breaker open")
	}
	a.cbState.mutex.RUnlock()

	err := operation()

	a.cbState.mutex.Lock()
	defer a.cbState.mutex.Unlock()

	if err != nil {
		a.cbState.failures++
		a.cbState.lastFailure = time.Now()

		if a.cbState.failures >= 5 {
			a.cbState.state = "open"
		}
	} else {
		a.cbState.failures = 0
		a.cbState.state = "closed"
	}

	return err
}

func (a *APIClient) doRequestWithMetrics(ctx context.Context, req *http.Request) (*http.Response, error) {
	start := time.Now()

	resp, err := a.httpClient.Do(req)
	duration := time.Since(start)

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	if a.metrics != nil {
		a.metrics.ObserveRequestDuration(req.Method, statusCode, duration)
		a.metrics.IncRequestCounter(req.Method, statusCode)

		if err != nil || statusCode >= 400 {
			errorType := "network"
			if err == nil {
				errorType = "http"
			}
			a.metrics.IncErrorCounter(req.Method, errorType)
		}
	}

	return resp, err
}
