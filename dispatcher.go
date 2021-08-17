package main

import (
	"context"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/stanlyzoolo/exprgen"
	"go.uber.org/zap"
)

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
		// TODO: NO!!!
		// Preparing the survey by replacing math operators to unicode format.
		prepareExpr := encodeMathOperators(job)

		// TODO: BAD!!!
		// Preparing request expression to restbasiccalc service.
		request := url + prepareExpr

		// TODO: BAD!!!
		data, err := http.Get(request) //nolint
		// TODO: BAD!!!
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

		time.Sleep(time.Millisecond * time.Duration(1000+rand.Intn(2000))) //nolint

		logger.Info("generating survey",
			zap.Int("survey number", id),
			zap.String("survey", job),
			zap.Int("survey result", rd.Result),
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
			logger.Info("*************************Dispatcher is closing jobs and surveys channels!***************************\n")
			close(d.surveys)

			close(d.jobs)

			return
		}
	}
}
