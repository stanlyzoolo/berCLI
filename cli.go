package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func init() { // nolint
	logger, _ := zap.NewDevelopment()

	// Load (without arguments) loads values from .env into the system from current path
	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading environment")
	}
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		logger.Error("can`t initialize zap logger: ",
			zap.Error(err))
	}

	defer logger.Sync() //nolint //FIXME: // Need help --> https://github.com/vmware-tanzu/octant/pull/263 !!!

	logger.Info("Let`s start calculate expressions!\n")

	// Set new config form .env file
	config := New()
	length := config.expressionLength
	workerPoolSize := config.workerPoolSize

	wg := &sync.WaitGroup{}
	wg.Add(workerPoolSize)

	d := dispatcher{
		surveys: make(chan string, workerPoolSize),
		jobs:    make(chan string, workerPoolSize),
		results: make(chan int, workerPoolSize),
	}

	for i := 0; i < workerPoolSize; i++ {
		go d.surveyMaker(d.surveys, uint8(length))
	}

	ctx, cancellation := context.WithCancel(context.Background())
	go d.startDispatcher(ctx)

	var url = config.berCLI.url
	for i := 0; i < workerPoolSize; i++ {
		go d.surveyWorker(d.jobs, d.results, url, i, wg)
	}

	// handle input signals (interrupt or terminate)
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan

	// shutdown
	logger.Info("*************************Shutdown signal received from user!****************************************\n")
	cancellation()
	wg.Wait()
	logger.Info("*************************All workers done their job, shutting down! Bye!****************************\n")
}
