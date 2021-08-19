package main

import (
	"context"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
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
func (d dispatcher) surveyWorker(jobs chan string, results chan int, baseURL string, id int, wg *sync.WaitGroup) {
	defer wg.Done()

	logger, _ := zap.NewDevelopment()

	for job := range jobs {
		URL, err := url.Parse(baseURL)
		if err != nil {
			logger.Error("URL parsing failed",
				zap.Error(err))
		}

		params := url.Values{}
		params.Add("expr", job) // Может выделить expr в конфиг?

		URL.RawQuery = params.Encode()

		cli := &http.Client{} //nolint //FIXME: need help!
		ctx := context.Background()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL.String(), nil)
		if err != nil {
			logger.Error("preparing the request failed\n",
				zap.String("package", "berCLI"),
				zap.String("url", URL.String()),
				zap.Error(err))
		}

		resp, err := cli.Do(req)
		if err != nil {
			logger.Error("failed Get response\n",
				zap.String("package", "berCLI"),
				zap.String("400", "Bad Request"),
				zap.String("url", URL.String()),
				zap.Error(err))
		}

		defer resp.Body.Close() //nolint //FIXME: need help!

		response, _ := ioutil.ReadAll(resp.Body)

		var rd returnData

		err = rd.unmarshalJSON(response)
		if err != nil {
			logger.Error("failed unmarshal json\n",
				zap.String("package", "berCLI"),
				zap.String("url", URL.String()),
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
