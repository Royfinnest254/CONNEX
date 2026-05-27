package main

import "fmt"

func main() {
    balance := 50000.75
    p := &balance // p holds the memory address (pointer) of balance

    fmt.Printf("Memory address: %p\n", p)
    fmt.Printf("Value at address: %.2f\n", *p) // *p opens the drawer to read the value

    *p = 65000.20 // opens the drawer at address p and overwrites the contents

    fmt.Printf("Updated balance: %.2f\n", balance)
}
