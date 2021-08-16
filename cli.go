package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/joho/godotenv"
	"github.com/stanlyzoolo/exprgen"
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
		log.Fatalf("can`t initialize zap logger: %v", err)
	}

	defer logger.Sync() //nolint

	logger.Info("Let`s start calculate expressions!")

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

	for i := 0; i < workerPoolSize; i++ {
		result := <-d.results
		logger.Info("reading results channel",
			zap.Int("worker number", i),
			zap.Int("calculated result", result),
		)
	}

	// handle input signals (interrupt or terminate)
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan

	// shutdown
	logger.Info("Shutdown signal received from user!")
	cancellation()
	wg.Wait()
	logger.Info("All workers done their job, shutting down! Bye!")
}

// encodeMathOperators process math operators to utf-8 format.
func encodeMathOperators(expr string) string {
	for _, e := range expr {
		if e == '+' {
			expr = strings.ReplaceAll(expr, string(e), "%2B")
		}

		if e == '-' {
			expr = strings.ReplaceAll(expr, string(e), "%2D")
		}

		if unicode.IsSpace(e) {
			expr = strings.ReplaceAll(expr, string(e), "%20")
		}
	}

	return expr
}

// returnData represents data with json tags for Marshal and  Unmarshal http response.
type returnData struct {
	Result int    `json:"result"`
	Error  error  `json:"error"`
	Expr   string `json:"expr"`
}

// unmarshalJSON is custom handler for writing error golang type to json struct field.
func (rd *returnData) unmarshalJSON(b []byte) error {
	type Alias returnData

	aux := &struct {
		Error string `json:"error"`
		*Alias
	}{
		Alias: (*Alias)(rd),
	}

	if err := json.Unmarshal(b, &aux); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	rd.Error = errors.New(aux.Error) // nolint

	return nil
}

// represents work with channels with data from external services (even exprgen).
type dispatcher struct {
	surveys chan string
	jobs    chan string
	results chan int
}

// surveyMaker creates expressions using input length and count with exprgen package.
func (d dispatcher) surveyMaker(surveys chan string, length uint8) {
	expression := exprgen.Generate(length)
	surveys <- expression
}

// surveyWorker prepare and send request to restbasiccalc service using surveys from
// jobs channel. Process http response and write result to results channel.
func (d dispatcher) surveyWorker(jobs chan string, results chan int, url string, id int, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		// Preparing the survey by replacing math operators to unicode format.
		prepareExpr := encodeMathOperators(job)

		// Preparing request expression to restbasiccalc service.
		request := url + prepareExpr

		data, err := http.Get(request) //nolint
		logger, _ := zap.NewDevelopment()

		if err != nil {
			logger.Error("failed Get request\n",
				zap.String("package", "berCLI"),
				zap.String("400", "Bad Request"),
				zap.String("url", url),
				zap.Error(err))
		}

		// read the response.
		response, _ := ioutil.ReadAll(data.Body)

		var rd returnData

		err = rd.unmarshalJSON(response)
		if err != nil {
			logger.Error("failed unmarshal json\n",
				zap.String("package", "berCLI"),
				zap.String("url", request),
				zap.Error(err))
		}
		time.Sleep(time.Millisecond * time.Duration(1000+rand.Intn(2000)))
		logger.Info("generating survey",
			zap.Int("survey number", id),
			zap.String("survey", job),
		)
		results <- rd.Result
	}
}

// startDispatcher acts as the proxy between the surveys and jobs channels,
//  with a select to support graceful shutdown.
func (d dispatcher) startDispatcher(ctx context.Context) {
	logger, _ := zap.NewDevelopment()

	for {
		select {
		case survey := <-d.surveys:
			d.jobs <- survey
		case <-ctx.Done():
			logger.Info("Dispatcher received cancellation signal, closing jobs and surveys channels")

			close(d.jobs)
			close(d.surveys)

			logger.Info("Dispatcher closed jobs channel")

			return
		}
	}
}

// 1. функция опроса калькулятора выражений (используя генератор выражений),
// с вычислением результата, с записью результата в логгер.
// 2. запуск одной и той функции в разном количестве горутин. Количество горутин - по флагу командной строки.
// 3. использовать шаблон worker pool (50/50).
// 4. graceful shutdown!

// каналы, контексты и т.д.
