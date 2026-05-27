package main

import "fmt"

func main() {
    witnesses := []string{"alpha", "beta"}
    fmt.Printf("Start: len=%d, cap=%d, contents=%v\n", len(witnesses), cap(witnesses), witnesses)

    witnesses = append(witnesses, "gamma")
    fmt.Printf("After 1 append: len=%d, cap=%d, contents=%v\n", len(witnesses), cap(witnesses), witnesses)

    witnesses = append(witnesses, "delta", "epsilon")
    fmt.Printf("After 3 appends: len=%d, cap=%d, contents=%v\n", len(witnesses), cap(witnesses), witnesses)
}
