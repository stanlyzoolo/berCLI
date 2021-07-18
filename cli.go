package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/stanlyzoolo/exprgen"
)

func main() {

	// default number is 1
	g := flag.Int("g", 1, "Number of goroutines to create")

	// default number is 1
	c := flag.Int("c", 1, "Length to generate expression")

	flag.Parse()

	fmt.Printf("Going to create %v goroutines\n", *g)

	var wg sync.WaitGroup

	t := uint8(*c)
	fmt.Printf("Length of expression: %v\n\n", t)

	ch := make(chan string)
	for i := 0; i < *g; i++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			ch <- exprgen.Generate(t)
		}(i)

		reqExpr := <-ch

		fmt.Printf("Request expression is: %s\n", reqExpr)

		path := fmt.Sprintf("http://localhost:8080/?expr=%s", url.QueryEscape(reqExpr))

		fmt.Printf("Query is: %s \n", path)

		resp, err := http.Get(path)

		if err != nil {
			fmt.Printf("error is %d", err)
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			fmt.Printf("error body is %d", err)
		}

		fmt.Printf("Responce form service: %s\n\n", body)

	}

	wg.Wait()
	fmt.Println("\nExiting...")
}
