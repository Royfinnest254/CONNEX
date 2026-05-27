package main

import (
    "fmt"
    "time"
)

func worker(jobs chan string) {
    for i := 0; i < 3; i++ {
        job := <-jobs // Receives a job capsule from the tube
        fmt.Println("Processing:", job)
        time.Sleep(100 * time.Millisecond)
    }
}

func main() {
    jobs := make(chan string, 3) // Buffered mailbox

    // Launch worker in background
    go worker(jobs)

    // Push 3 items into the mailbox (does not block because cap is 3)
    jobs <- "Verify MTI"
    jobs <- "Validate Bitmap"
    jobs <- "Hash Transaction"

    fmt.Println("All jobs queued!")

    // Keep main running so the program doesn't close before the worker finishes
    time.Sleep(500 * time.Millisecond)
}
