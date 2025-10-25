package main

import (
	"log/slog"
	"sync"
)

var wg = sync.WaitGroup{}
var logger *slog.Logger

func main() {
	run()
	stop()
}

func run() {
	container := GetContainer()
	logger = container.Logger
	logger.Info("Agent service initialized")

	wg.Add(1)
	go func() {
		defer wg.Done()
		container.Handler.Run()
	}()
}

func stop() {
	wg.Wait()
	logger.Info("Agent service stopped")
}
