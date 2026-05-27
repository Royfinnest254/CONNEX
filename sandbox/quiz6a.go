package main

import (
    "errors"
    "fmt"
)

func checkDatabaseConnection() error {
    return errors.New("connection timeout")
}

func initializeLedger() error {
    err := checkDatabaseConnection()
    if err != nil {
        // Wrap the error with context using %w
        return fmt.Errorf("initialize ledger: %w", err)
    }
    return nil
}

func main() {
    err := initializeLedger()
    if err != nil {
        fmt.Println("Full Error Chain:", err)
        
        // Unwrap opens the outermost doll to reveal the root error
        rootErr := errors.Unwrap(err)
        fmt.Println("Root Cause:", rootErr)
        return
    }
    fmt.Println("Ledger initialized successfully!")
}
