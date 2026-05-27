package main

import "fmt"

type FeeCalculator struct {
    FixedFee      float64
    PercentageFee float64
}

// Value receiver: f is a local copy, read-only
func (f FeeCalculator) Calculate(amount float64) float64 {
    return f.FixedFee + (amount * f.PercentageFee)
}

func main() {
    calc := FeeCalculator{
        FixedFee:      50.00,
        PercentageFee: 0.01,
    }

    fee := calc.Calculate(10000.00)
    fmt.Printf("Total Fee: KES %.2f\n", fee)
}
