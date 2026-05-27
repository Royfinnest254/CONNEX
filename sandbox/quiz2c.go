package main

import "fmt"

func main() {
    // make initializes a map so it is ready to store data
    isoFields := make(map[int]string)
    isoFields[3] = "310100"
    isoFields[4] = "000000500000"

    // Lookup with comma ok check
    val4, ok4 := isoFields[4]
    if ok4 {
        fmt.Printf("Field 4 found: %s\n", val4)
    } else {
        fmt.Println("Field 4 not found!")
    }

    val11, ok11 := isoFields[11]
    if ok11 {
        fmt.Printf("Field 11 found: %s\n", val11)
    } else {
        fmt.Println("Field 11 not found (returned empty string: \"" + val11 + "\")")
    }
}
