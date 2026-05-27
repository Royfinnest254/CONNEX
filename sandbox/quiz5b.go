package main

import "fmt"

type BankAccount struct {
    AccountHolder string
    Balance       float64
}

// Pointer receiver: modifies the actual balance in the pantry
func (a *BankAccount) Deposit(amount float64) {
    a.Balance += amount
}

// Pointer receiver: modifies the balance and returns success/failure status
func (a *BankAccount) Withdraw(amount float64) bool {
    if a.Balance < amount {
        return false // Insufficient funds!
    }
    a.Balance -= amount
    return true
}

func main() {
    acc := &BankAccount{
        AccountHolder: "John Doe",
        Balance:       500.00,
    }

    fmt.Printf("Initial Balance: KES %.2f\n", acc.Balance)

    acc.Deposit(1000.00)
    fmt.Printf("After Deposit:   KES %.2f\n", acc.Balance)

    if acc.Withdraw(300.00) {
        fmt.Println("Withdrawal of 300.00 successful ✓")
    } else {
        fmt.Println("Withdrawal of 300.00 failed ❌")
    }
    fmt.Printf("Balance:         KES %.2f\n", acc.Balance)

    if acc.Withdraw(2000.00) {
        fmt.Println("Withdrawal of 2000.00 successful ✓")
    } else {
        fmt.Println("Withdrawal of 2000.00 failed ❌ (Insufficient Funds)")
    }
    fmt.Printf("Final Balance:   KES %.2f\n", acc.Balance)
}
