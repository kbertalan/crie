package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	endpoint := os.Getenv("CRIE_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://crie:10000"
	}

	functionName := os.Getenv("CRIE_LAMBDA_NAME")
	if functionName == "" {
		functionName = "my-function"
	}

	concurrency := 5
	if v := os.Getenv("CLIENT_CONCURRENCY"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid CLIENT_CONCURRENCY: %s\n", v)
			os.Exit(1)
		}
		concurrency = n
	}

	url := fmt.Sprintf("%s/2015-03-31/functions/%s/invocations", endpoint, functionName)
	payload := []byte(`{"key": "value"}`)

	fmt.Printf("invoking %s with concurrency=%d\n", url, concurrency)

	var wg sync.WaitGroup
	for i := range concurrency {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			start := time.Now()

			resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
			if err != nil {
				fmt.Printf("[%d] error: %v\n", id, err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			fmt.Printf("[%d] status=%d body=%s duration=%s\n", id, resp.StatusCode, string(body), time.Since(start))
		}(i)
	}
	wg.Wait()
	fmt.Println("done")
}
