package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	duration := flag.Int("duration", 10, "Duration in seconds for the program to run")
	flag.Parse()

	if *duration <= 0 {
		return fmt.Errorf("duration must be a positive integer")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*duration)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countdown(ctx, *duration)
	}()

	go func() {
		defer wg.Done()
		generateRandomNumbers(ctx)
	}()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nInterrupt received. Shutting down...")
		cancel()
	}()

	wg.Wait()
	fmt.Println("Program completed. Exiting...")
	return nil
}

func countdown(ctx context.Context, duration int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for remaining := duration; remaining > 0; remaining-- {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Printf("\rTime remaining: %d seconds", remaining)
		}
	}
	fmt.Println("\nTime's up!")
}

func generateRandomNumbers(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Printf("\nRandom number: %d", rand.Intn(100))
		}
	}
}
