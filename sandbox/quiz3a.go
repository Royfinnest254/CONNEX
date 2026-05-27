package main

import "fmt"

type WitnessState struct {
    Name   string
    Active bool
}

// Since w is a pointer (*), we are editing the original form in the pantry
func deactivate(w *WitnessState) {
    w.Active = false // Go automatically opens the drawer at address w
}

func main() {
    // Instantiate as a pointer using the & operator
    witness := &WitnessState{
        Name:   "Beta",
        Active: true,
    }

    fmt.Printf("Before: Name=%s, Active=%t\n", witness.Name, witness.Active)

    deactivate(witness) // Passes the address card to deactivate()

    fmt.Printf("After:  Name=%s, Active=%t\n", witness.Name, witness.Active)
}
