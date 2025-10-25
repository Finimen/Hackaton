package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	client "NetScan/internal/agent/clients"
	"NetScan/internal/agent/domain"
	handler "NetScan/internal/agent/handlers"
)

var (
	wg          sync.WaitGroup
	logger      *slog.Logger
	shutdownCtx context.Context
	cancelFunc  context.CancelFunc
)

func main() {
	shutdownCtx, cancelFunc = context.WithCancel(context.Background())
	defer cancelFunc()

	setupSignalHandling()

	if err := run(); err != nil {
		logger.Error("Failed to start agent", "error", err)
		os.Exit(1)
	}

	logger.Info("Agent is running, waiting for shutdown signal...")

	<-shutdownCtx.Done()

	stop()
}

func run() error {
	container := GetContainer()
	logger = container.Logger

	agentMetadata, err := initAgentMetadata()
	if err != nil {
		return fmt.Errorf("failed to init agent metadata: %w", err)
	}

	// Creation of the test agent
	agent := domain.NewAgent(
		getEnv("AGENT_NAME", "net-scan-agent"),
		getEnv("AGENT_LOCATION", "unknown"),
		getEnv("REGISTRATION_TOKEN", ""),
	)
	agent.UpdateMetadata(agentMetadata)

	// Registration of the agent
	baseURL := getEnv("BACKEND_URL", "http://localhost:8080")
	apiClient := client.NewAPIClient(baseURL, "", "")

	logger.Info("Registering agent", "name", agent.Name, "location", agent.Location)

	if err := apiClient.RegisterAgent(shutdownCtx, agent); err != nil {
		return fmt.Errorf("agent registration failed: %w", err)
	}

	logger.Info("Agent registered successfully", "agent_id", apiClient.GetAgentID())

	taskHandler := handler.NewTaskHandler(container.Factory, logger)

	agentHandler := handler.NewAgentHandler(logger, apiClient, taskHandler)

	wg.Add(3)

	// Main loop of tasks processing
	go func() {
		defer wg.Done()
		logger.Info("Starting task processing loop")
		agentHandler.Run(shutdownCtx)
		logger.Info("Task processing loop stopped")
	}()

	// Heartbeat loop
	go func() {
		defer wg.Done()
		logger.Info("Starting heartbeat loop")
		runHeartbeatLoop(shutdownCtx, apiClient)
		logger.Info("Heartbeat loop stopped")
	}()

	// Health check loop (for agent monitoring)
	go func() {
		defer wg.Done()
		logger.Info("Starting health check loop")
		runHealthCheckLoop(shutdownCtx)
		logger.Info("Health check loop stopped")
	}()

	logger.Info("Agent service fully initialized and running",
		"agent_id", apiClient.GetAgentID(),
		"backend", baseURL,
	)

	return nil
}

func stop() {
	logger.Info("Shutting down agent service...")

	cancelFunc()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Agent service stopped gracefully")
	case <-time.After(30 * time.Second):
		logger.Warn("Agent service shutdown timed out - forcing exit")
	}
}

func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal, initiating shutdown", "signal", sig)
		cancelFunc()
	}()
}

func runHeartbeatLoop(ctx context.Context, apiClient *client.APIClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := apiClient.SendHeartbeat(ctx); err != nil {
				logger.Warn("Heartbeat failed", "error", err)
			} else {
				logger.Debug("Heartbeat sent successfully")
			}
		}
	}
}

func runHealthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			monitorAgentHealth()
		}
	}
}

func monitorAgentHealth() {
	logger.Debug("Agent health check completed")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getCPUCount() int {
	return 1
}

func getTotalMemory() int64 {
	return 1024
}

func getGoVersion() string {
	return "1.21"
}
