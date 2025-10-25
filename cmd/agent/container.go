package main

import (
	client "NetScan/internal/agent/clients"
	"NetScan/internal/agent/domain"
	handler "NetScan/internal/agent/handlers"
	runner "NetScan/internal/agent/runners"
	"log/slog"
	"os"
)

type Container struct {
	Logger       *slog.Logger
	AgentHandler *handler.AgentHandler
	APIClient    *client.APIClient
	TaskRunner   *runner.Factory
	TaskHandler  *handler.TaskHandler
}

func GetContainer() *Container {
	container := &Container{}

	container.initLogger()
	container.initAPIClient()
	container.initTaskRunners()
	container.initHandlers()

	return container
}

func (c *Container) GetEnv(key, defaultValue string) string {
	return getEnv(key, defaultValue)
}

func (c *Container) initLogger() {
	logLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	c.Logger = slog.New(handler)
}

func (c *Container) initAPIClient() {
	baseURL := getEnv("BACKEND_URL", "http://localhost:8080")
	token := getEnv("AGENT_TOKEN", "")
	agentID := getEnv("AGENT_ID", "")

	c.APIClient = client.NewAPIClient(baseURL, token, agentID)
}

func (c *Container) initTaskRunners() {
	httpRunner := runner.NewHTTPRunner()
	pingRunner := runner.NewPingRunner()
	dnsRunner := runner.NewDNSRunner()
	tcpRunner := runner.NewTCPRunner()

	c.TaskRunner = runner.NewFactory(httpRunner, pingRunner, dnsRunner, tcpRunner)
}

func (c *Container) initHandlers() {
	c.TaskHandler = handler.NewTaskHandler(c.TaskRunner, c.Logger)
	c.AgentHandler = handler.NewAgentHandler(c.Logger, c.APIClient, c.TaskHandler)
}

func initAgentMetadata() (domain.AgentMetadata, error) {
	hostname, _ := os.Hostname()

	return domain.AgentMetadata{
		IPAddress: getEnv("AGENT_IP", "127.0.0.1"),
		Hostname:  hostname,
		OS:        getEnv("OS", "unknown"),
		Arch:      getEnv("ARCH", "unknown"),
		CPUCount:  getCPUCount(),
		MemoryMB:  getTotalMemory(),
		GoVersion: getGoVersion(),
	}, nil
}
