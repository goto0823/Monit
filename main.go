package main

import (
	"getData/metrices"
	"github.com/joho/godotenv"
	"log"
	"sync"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	
	go func() {
		defer wg.Done()
		metrices.HttpMetrices()
	}()

	wg.Wait()

}
