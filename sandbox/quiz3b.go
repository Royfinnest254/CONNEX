package main

import (
    "encoding/json"
    "fmt"
)

type SystemConfig struct {
    Port        int    `json:"port"`
    DBPath      string `json:"db_path"`
    secretToken string // Lowercase! Private field.
}

func main() {
    config := SystemConfig{
        Port:        8080,
        DBPath:      "data/connex.db",
        secretToken: "SUPER_SECRET",
    }

    // 1. Convert struct to JSON text bytes
    jsonBytes, err := json.Marshal(config)
    if err != nil {
        fmt.Println("Error marshalling:", err)
        return
    }
    fmt.Println("JSON output:", string(jsonBytes))

    // 2. Convert JSON text back into a Go struct
    inputJSON := `{"port":9000,"db_path":"/tmp/test.db"}`
    var newConfig SystemConfig

    // We MUST pass a pointer (&newConfig) so json.Unmarshal can modify the fields!
    err = json.Unmarshal([]byte(inputJSON), &newConfig)
    if err != nil {
        fmt.Println("Error unmarshalling:", err)
        return
    }

    fmt.Printf("Parsed Struct: Port=%d, DBPath=%s, secretToken=%q\n",
        newConfig.Port, newConfig.DBPath, newConfig.secretToken)
}
