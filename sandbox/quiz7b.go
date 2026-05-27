package main

import (
    "fmt"
    "time"
)

func simulateWitness(ch chan string) {
    time.Sleep(2 * time.Second) // Simulate slow network call
    ch <- "Alpha Approved ✓"
}

func main() {
    response := make(chan string)

    go simulateWitness(response)

    // Test Case 1: Timeout is shorter than network call
    fmt.Println("--- Running with 1-second timeout ---")
    select {
    case msg := <-response:
        fmt.Println("Received:", msg)
    case <-time.After(1 * time.Second):
        fmt.Println("⏰ Timeout reached! Payment rejected.")
    }

    // Test Case 2: Timeout is longer than network call
    response2 := make(chan string)
    go simulateWitness(response2)

    fmt.Println("\n--- Running with 3-second timeout ---")
    select {
    case msg := <-response2:
        fmt.Println("Received:", msg)
    case <-time.After(3 * time.Second):
        fmt.Println("⏰ Timeout reached! Payment rejected.")
    }
}
