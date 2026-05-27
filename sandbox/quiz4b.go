package main

import "fmt"

// Package scope: visible to all functions in this file
const CommissionRate = 0.02

func calculateFee(amount float64) (float64, float64) {
    fee := amount * CommissionRate // Local function scope
    net := amount - fee            // Local function scope
    return fee, net
}

func main() {
    txAmount := 50000.00

    // Receive the multiple return values
    fee, net := calculateFee(txAmount)

    fmt.Printf("Transaction Amount: KES %.2f\n", txAmount)
    fmt.Printf("Commission Fee (2%%): KES %.2f\n", fee)
    fmt.Printf("Net Amount Received: KES %.2f\n", net)

    // Note: If you try to print "fee" variable declared inside calculateFee() here,
    // the code will fail to compile. It is locked inside calculateFee()'s fence.
}
