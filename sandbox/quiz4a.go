package main

import "fmt"

func doubleAmountValue(val float64) {
    val = val * 2
    fmt.Printf("Inside doubleAmountValue: %.2f\n", val)
}

func doubleAmountPointer(val *float64) {
    *val = *val * 2 // Modify the original variable in the pantry
}

func main() {
    balance := 150.50

    fmt.Println("--- Pass-by-Value Test ---")
    doubleAmountValue(balance)
    fmt.Printf("Original balance afterwards: %.2f\n", balance)

    fmt.Println("\n--- Pass-by-Pointer Test ---")
    doubleAmountPointer(&balance) // Pass the memory address
    fmt.Printf("Original balance afterwards: %.2f\n", balance)
}
