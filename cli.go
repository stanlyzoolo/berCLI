package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/joho/godotenv"
	"github.com/stanlyzoolo/exprgen"
	"go.uber.org/zap"
)

func init() {

	logger, _ := zap.NewDevelopment()
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading environment")
	}
}

var wg sync.WaitGroup

func main() {

	fmt.Println("Hello! Let`s start!")

	logger, _ := zap.NewDevelopment()

	// Set new config form .env file
	conf := NewConfig()
	logger.Info("\nPackage berCLI use the next .env variables:",
		zap.String("URL for survey", conf.berCLI.url),
		zap.String("ExpressionLength", conf.berCLI.expressionLength),
		zap.String("WorkerPoolSize", conf.berCLI.workerPoolSize))

	// get expression length from config file
	length, err := strconv.Atoi(conf.berCLI.expressionLength)
	if err != nil {
		logger.Error("failed to convert length of expression  to int type",
			zap.String("package", "berCLI"),
			zap.Error(err))
	}

	// get worker pool size from config file
	workerPoolSize, err := strconv.Atoi(conf.berCLI.workerPoolSize)
	if err != nil {
		logger.Error("failed to convert worker pool size to int type",
			zap.String("package", "berCLI"),
			zap.Error(err))
	}

	var surveys = make(chan string, workerPoolSize)
	var results = make(chan int, workerPoolSize)

	var url = conf.berCLI.url

	for i := 0; i < workerPoolSize; i++ {
		// wg.Add(1)
		go surveyMaker(surveys, uint8(length))

		go surveyWorker(surveys, results, url, i)
		// wg.Done()
	}

	for i := 0; i < workerPoolSize; i++ {
		result := <-results
		fmt.Printf("[worker #%v] Calculated result: %v\n\n", i, result)
	}

	wg.Wait()

	fmt.Println("All workers done their job!")

}

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

type returnData struct {
	Result int    `json:"result"`
	Error  error  `json:"error"`
	Expr   string `json:"expr"`
}

func (rd *returnData) unmarshalJSON(b []byte) error {
	type Alias returnData
	aux := &struct {
		Error string `json:"error"`
		*Alias
	}{
		Alias: (*Alias)(rd),
	}

	err := json.Unmarshal(b, &aux)

	if err != nil {
		return err
	}

	rd.Error = errors.New(aux.Error)

	return nil
}

func surveyMaker(surveys chan string, length uint8) {
	// Create expressions using input length and count with exprgen package
	expression := exprgen.Generate(length)
	surveys <- expression
}

func surveyWorker(surveys chan string, results chan int, url string, id int) {

	for survey := range surveys {
		// Preparing the survey by replacing math operators to unicode format
		prepareExpr := encodeMathOperators(survey)

		// Preparing request expression to restbasiccalc service
		request := url + prepareExpr

		data, err := http.Get(request)
		logger, _ := zap.NewDevelopment()
		if err != nil {
			logger.Error("failed Get request\n",
				zap.String("package", "berCLI"),
				zap.String("400", "Bad Request"),
				zap.String("url", url),
				zap.Error(err))
		}

		// read the response
		responce, _ := ioutil.ReadAll(data.Body)

		rd := returnData{}
		err = rd.unmarshalJSON(responce)
		if err != nil {
			logger.Error("failed unmarshal json\n",
				zap.String("package", "berCLI"),
				zap.String("url", request),
				zap.Error(err))
		}
		fmt.Printf("[expression #%v] Generated survey: (%s);\n\n", id, survey)
		results <- rd.Result
	}

	wg.Done()

}

// 1. функция опроса калькулятора выражений (используя генератор выражений), с вычислением результата, с записью результата в логгер.
// 2. запуск одной и той функции в разном количестве горутин. Количество горутин - по флагу командной строки.
// 3. использовать шаблон worker pool (50/50).
// 4. graceful shutdown!

// каналы, контексты и т.д.
