package main

import (
    "errors"
    "fmt"
)

// Custom error struct
type FundError struct {
    Required  float64
    Available float64
}

// Implement the error interface contract
func (e *FundError) Error() string {
    return fmt.Sprintf("insufficient funds: need KES %.2f, only have KES %.2f", e.Required, e.Available)
}

// Standard sentinel error
var ErrSystemMaintenance = errors.New("system undergoing maintenance")

func processPayment(amount float64, balance float64, maintenance bool) error {
    if maintenance {
        return ErrSystemMaintenance
    }
    if amount > balance {
        return &FundError{Required: amount, Available: balance}
    }
    return nil
}

func main() {
    // Test Case 1: Insufficient funds
    fmt.Println("--- Test Case 1 ---")
    err1 := processPayment(1000.00, 500.00, false)
    if err1 != nil {
        var fundErr *FundError
        // errors.As checks if err1 contains a *FundError and extracts it
        if errors.As(err1, &fundErr) {
            fmt.Println("Fund Error caught!")
            fmt.Printf("Required: KES %.2f, Available: KES %.2f\n", fundErr.Required, fundErr.Available)
        } else {
            fmt.Println("Other error:", err1)
        }
    }

    // Test Case 2: System maintenance
    fmt.Println("\n--- Test Case 2 ---")
    err2 := processPayment(100.00, 500.00, true)
    if err2 != nil {
        // errors.Is checks if err2 is ErrSystemMaintenance
        if errors.Is(err2, ErrSystemMaintenance) {
            fmt.Println("Error: System is currently down for maintenance.")
            fmt.Println("Action: Retry transaction later.")
        } else {
            fmt.Println("Other error:", err2)
        }
    }
}
