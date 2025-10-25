package main

import (
	client "NetScan/internal/agent/clients"
	"NetScan/internal/agent/domain"
	handler "NetScan/internal/agent/handlers"
	runner "NetScan/internal/agent/runners"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("Choose test mode:")
	fmt.Println("1 - Direct runner testing")
	fmt.Println("2 - Mock server integration")
	fmt.Println("3 - Real targets integration")

	var choice int
	fmt.Scan(&choice)

	switch choice {
	case 1:
		TestRunnersDirectly()
	case 2:
		TestAgentStandalone()
	case 3:
		TestWithRealTargets()
	default:
		fmt.Println("Invalid choice")
	}
}

type MockServer struct {
	port   string
	tasks  []interface{}
	agents map[string]interface{}
}

func NewMockServer(port string) *MockServer {
	return &MockServer{
		port:   port,
		agents: make(map[string]interface{}),
		tasks:  []interface{}{},
	}
}

func (m *MockServer) Start() {
	http.HandleFunc("/api/agents/register", m.handleRegister)
	http.HandleFunc("/api/agents/tasks", m.handleGetTasks)
	http.HandleFunc("/api/agents/results", m.handleSubmitResult)
	http.HandleFunc("/api/agents/heartbeat", m.handleHeartbeat)

	log.Printf("Mock server starting on port %s", m.port)
	log.Fatal(http.ListenAndServe(":"+m.port, nil))
}

func (m *MockServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"token":    "mock-token-12345",
		"agent_id": "mock-agent-67890",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("Agent registered")
}

func (m *MockServer) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	// Чередуем возврат задач и пустой ответ
	if time.Now().Second()%2 == 0 {
		task := map[string]interface{}{
			"id":         "task-1",
			"type":       "http",
			"target":     "https://httpbin.org/json",
			"agent_id":   "mock-agent-67890",
			"created_at": time.Now(),
			"options": map[string]interface{}{
				"timeout": 30,
				"method":  "GET",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
		log.Println("Sent HTTP task")
	} else {
		w.WriteHeader(http.StatusNoContent)
		log.Println("No tasks available")
	}
}

func (m *MockServer) handleSubmitResult(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received result: TaskID=%s, Success=%v",
		result["task_id"], result["success"])

	w.WriteHeader(http.StatusOK)
}

func (m *MockServer) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	log.Println("Heartbeat received")
	w.WriteHeader(http.StatusOK)
}

func TestAgentStandalone() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger.Info("Starting standalone agent test")

	mockServer := NewMockServer("8081")
	go mockServer.Start()

	time.Sleep(2 * time.Second)

	apiClient := client.NewAPIClient("http://localhost:8081", "mock-token", "mock-agent")

	httpRunner := runner.NewHTTPRunner()
	pingRunner := runner.NewPingRunner()
	dnsRunner := runner.NewDNSRunner()
	tcpRunner := runner.NewTCPRunner()

	factory := runner.NewFactory(httpRunner, pingRunner, dnsRunner, tcpRunner)
	taskHandler := handler.NewTaskHandler(factory, logger)
	agentHandler := handler.NewAgentHandler(logger, apiClient, taskHandler)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Starting agent handler for 30 seconds...")
	agentHandler.Run(ctx)
	logger.Info("Agent test completed")
}

func TestRunnersDirectly() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	logger.Info("For dev only!")

	fmt.Println("=== Testing HTTP Runner ===")
	httpRunner := runner.NewHTTPRunner()
	httpResult, err := httpRunner.Execute(ctx, "https://httpbin.org/json", map[string]interface{}{
		"timeout": 10,
		"method":  "GET",
	})
	if err != nil {
		log.Printf("HTTP Runner error: %v", err)
	} else {
		prettyPrint(httpResult)
	}

	fmt.Println("\n=== Testing Ping Runner ===")
	pingRunner := runner.NewPingRunner()
	pingResult, err := pingRunner.Execute(ctx, "google.com:80", map[string]interface{}{
		"count":   3,
		"timeout": 5,
	})
	if err != nil {
		log.Printf("Ping Runner error: %v", err)
	} else {
		prettyPrint(pingResult)
	}

	fmt.Println("\n=== Testing DNS Runner ===")
	dnsRunner := runner.NewDNSRunner()
	dnsResult, err := dnsRunner.Execute(ctx, "google.com", map[string]interface{}{
		"record_type": "A",
		"server":      "8.8.8.8:53",
	})
	if err != nil {
		log.Printf("DNS Runner error: %v", err)
	} else {
		prettyPrint(dnsResult)
	}

	fmt.Println("\n=== Testing TCP Runner ===")
	tcpRunner := runner.NewTCPRunner()
	tcpResult, err := tcpRunner.Execute(ctx, "google.com:80", map[string]interface{}{
		"timeout": 5,
	})
	if err != nil {
		log.Printf("TCP Runner error: %v", err)
	} else {
		prettyPrint(tcpResult)
	}
}

func prettyPrint(data interface{}) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	fmt.Println(string(jsonBytes))
}

func TestWithRealTargets() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	logger.Info("For dev only!")

	testTargets := []struct {
		name    string
		target  string
		check   domain.CheckType
		options map[string]interface{}
	}{
		{
			name:   "HTTP Check - JSON API",
			target: "https://httpbin.org/json",
			check:  domain.HTTPCheck,
			options: map[string]interface{}{
				"timeout": 10,
				"method":  "GET",
			},
		},
		{
			name:   "Ping Check - Google",
			target: "google.com:80",
			check:  domain.PingCheck,
			options: map[string]interface{}{
				"count":   2,
				"timeout": 5,
			},
		},
		{
			name:   "DNS Check - Google",
			target: "google.com",
			check:  domain.DNSCheck,
			options: map[string]interface{}{
				"record_type": "A",
			},
		},
		{
			name:   "TCP Check - SSH Port",
			target: "github.com:22",
			check:  domain.TCPCheck,
			options: map[string]interface{}{
				"timeout": 5,
			},
		},
		{
			name:   "HTTPS Check with SSL",
			target: "https://github.com",
			check:  domain.HTTPCheck,
			options: map[string]interface{}{
				"timeout":          10,
				"verify_ssl":       true,
				"follow_redirects": false,
			},
		},
	}

	factory := createRunnerFactory()

	for _, test := range testTargets {
		fmt.Printf("\n=== Testing: %s ===\n", test.name)
		fmt.Printf("Target: %s, Type: %s\n", test.target, test.check)

		runner, err := factory.GetRunner(test.check)
		if err != nil {
			log.Printf("Error getting runner: %v", err)
			continue
		}

		start := time.Now()
		result, err := runner.Execute(ctx, test.target, test.options)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Test failed: %v", err)
		} else {
			fmt.Printf("✅ Test completed in %v\n", duration)

			printKeyMetrics(test.check, result)
		}

		time.Sleep(1 * time.Second)
	}
}

func createRunnerFactory() *runner.Factory {
	return runner.NewFactory(
		runner.NewHTTPRunner(),
		runner.NewPingRunner(),
		runner.NewDNSRunner(),
		runner.NewTCPRunner(),
	)
}

func printKeyMetrics(checkType domain.CheckType, result map[string]interface{}) {
	switch checkType {
	case domain.HTTPCheck, domain.HTTPSCheck:
		if status, ok := result["status_code"]; ok {
			fmt.Printf("   Status: %v, Response Time: %vms\n",
				status, result["response_time"])
		}
	case domain.PingCheck:
		fmt.Printf("   Packet Loss: %.1f%%, Avg RTT: %.2fms\n",
			result["packet_loss"], result["avg_rtt"])
	case domain.DNSCheck:
		if records, ok := result["records"]; ok {
			if recordList, ok := records.([]string); ok {
				fmt.Printf("   Records: %d, First: %s\n",
					len(recordList), safeGetFirst(recordList))
			}
		}
	case domain.TCPCheck:
		fmt.Printf("   Port Open: %v, Connect Time: %vms\n",
			result["port_open"], result["connect_time"])
	}
}

func safeGetFirst(slice []string) string {
	if len(slice) > 0 {
		if len(slice[0]) > 50 {
			return slice[0][:50] + "..."
		}
		return slice[0]
	}
	return "none"
}
