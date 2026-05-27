<picture>
  <source media="(prefers-color-scheme: dark)" srcset="connex_logo_dark.png">
  <source media="(prefers-color-scheme: light)" srcset="connex_logo_light.png">
  <img alt="CONNEX" src="connex_logo_light.png" width="220">
</picture>

---

# Go Programming: Learn by Reading CONNEX
### A Complete Beginner's Course — Built From the Real Codebase

> Every concept you learn here is taken directly from the real CONNEX banking system source code. By the end, you will open any Go file in this project and understand every single line.

---

## How This Course Works

Each chapter follows this pattern:

1. 📖 **Plain English Explanation** — The concept explained like you are 10 years old
2. 🔍 **The Real Code** — Actual lines from CONNEX with a word-by-word breakdown
3. ✏️ **Your Quiz** — A challenge for you to try on your own first
4. ✅ **The Answer** — The full, correct solution with explanations

> ⚠️ **Safety Rule:** Create a folder on your Desktop called `sandbox/`. Do ALL your practice there. Never edit files inside `cmd/` or `internal/`. You cannot break the banking system from your sandbox.

---

## Before You Start: Setting Up Your Sandbox

Open PowerShell and run these two commands:

```powershell
mkdir $HOME\Desktop\sandbox
cd $HOME\Desktop\sandbox
```

Then verify Go is installed:

```powershell
go version
```

You should see something like `go version go1.22.0 windows/amd64`. If not, download Go from https://go.dev/dl/ first.

---

---

# Chapter 1: The Starting Line

## `package` and `import` — Declaring and Borrowing

---

### 📖 Plain English

Before you cook a meal, you declare what kitchen you are in (`package main`) and you gather your ingredients from the pantry (`import`).

- **`package main`** tells Go: *"This file is a complete program that can be turned on and run."*
- **`import`** tells Go: *"Before I start, I need to borrow these toolboxes from Go's standard library."*

Go comes with hundreds of free built-in toolboxes. You just have to name which ones you need. You only pay for what you use — if you import a toolbox and never use it, Go will refuse to compile and tell you to remove it.

---

### 🔍 The Real Code — `cmd/witness/main.go` Lines 11–27

This is the very top of the Witness Node — the program that signs bank transactions:

```go
package main          // "This is a runnable program, not just a library"

import (
    "crypto/ed25519"  // Toolbox: Ed25519 cryptographic signatures
    "crypto/rand"     // Toolbox: Cryptographically secure random bytes
    "crypto/sha256"   // Toolbox: SHA-256 hashing algorithm
    "encoding/base64" // Toolbox: Convert binary bytes to readable Base64 text
    "encoding/hex"    // Toolbox: Convert binary bytes to readable hex strings
    "encoding/json"   // Toolbox: Read and write JSON data
    "flag"            // Toolbox: Read arguments from the command line
    "fmt"             // Toolbox: Format and print text ("format")
    "log/slog"        // Toolbox: Structured logging with key-value pairs
    "net/http"        // Toolbox: Build web servers and make HTTP requests
    "os"              // Toolbox: Read/write files and interact with the OS
    "path/filepath"   // Toolbox: Work with file paths safely on any OS
    "time"            // Toolbox: Dates, clocks, timers, sleeping
)
```

**Key things to notice:**
- The `/` inside import paths is NOT division. It is a folder separator. `"crypto/sha256"` means the `sha256` package inside Go's built-in `crypto` folder.
- The parentheses `( )` let you list many imports at once. Without them you would need to write `import "fmt"` on a separate line for each one.
- Every import is in double quotes `""`.

---

### ✏️ Quiz 1

**Task:** In your `sandbox/` folder, create a file called `quiz1.go`.

Write a program that:
1. Declares `package main`
2. Imports only the `fmt` and `time` toolboxes
3. Has a `main()` function (every runnable Go program must have one)
4. Inside `main()`, prints: `"CONNEX Witness Node — Online"`
5. On the next line, prints the current date and time

**Run it from your sandbox folder with:** `go run quiz1.go`

---

### ✅ Answer — Quiz 1

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    fmt.Println("CONNEX Witness Node — Online")
    fmt.Println("Current time:", time.Now())
}
```

**What each line does:**
- `fmt.Println(...)` — The `Println` function from the `fmt` toolbox prints text and automatically adds a new line at the end.
- `time.Now()` — Calls the `Now()` function from the `time` toolbox. It returns a `Time` object representing this exact moment.
- `Println` is smart enough to convert the `Time` object to a human-readable string automatically.

**Expected output:**
```
CONNEX Witness Node — Online
Current time: 2026-05-27 18:31:00.123456789 +0300 EAT m=+0.001234567
```

---

---

# Chapter 2: Storing Data

## Variables — Labeled Boxes in Memory

---

### 📖 Plain English

A **variable** is a labeled box that holds a piece of data in the computer's memory. When you create a variable, you are telling the computer: *"Reserve a small space in memory, call it `amount`, and put the number 5000 in it."*

In Go there are two ways to create variables:

**Method 1 — Short declaration (most common):**
```go
amount := 5000
```
The `:=` operator creates the box AND fills it at the same time. Go automatically figures out the type.

**Method 2 — Explicit declaration:**
```go
var amount int = 5000
```
Here you explicitly tell Go it is an `int` (integer = whole number).

**The main data types you will see in CONNEX:**

| Type | What it holds | Example |
|------|--------------|---------|
| `string` | Text | `"Alice"` |
| `int` | Whole numbers | `42` |
| `float64` | Decimal numbers | `98750.50` |
| `bool` | True or false | `true` |
| `[]byte` | Raw binary data | `[]byte{0x02, 0x00}` |
| `error` | An error message or `nil` | `nil` means "no error" |

---

### 🔍 The Real Code — `cmd/witness/main.go` Lines 34–35

```go
privPath := keyPath          // Create box "privPath", copy keyPath's value into it
pubPath  := keyPath + ".pub" // Create box "pubPath", same value but with ".pub" glued on
```

If `keyPath` is `"keys/witness.key"`, then after these two lines:
- `privPath` = `"keys/witness.key"`
- `pubPath`  = `"keys/witness.key.pub"`

The `+` operator on strings glues them together (this is called **concatenation**).

---

### 🔍 The Real Code — `cmd/gateway/main.go` Line 198

```go
bundleID := fmt.Sprintf("CX-%s-%x", time.Now().UTC().Format("20060102150405.000000"), randBytes)
```

This creates a unique ID for every single bank transaction. Let's break it apart piece by piece:

| Piece | Meaning |
|-------|---------|
| `bundleID :=` | Create a new variable called `bundleID` |
| `fmt.Sprintf(...)` | Build a string from a template (like filling in blanks) |
| `"CX-%s-%x"` | The template: `CX-` then a string `%s` then `-` then hex bytes `%x` |
| `time.Now().UTC()` | Get the current time in UTC timezone |
| `.Format("20060102150405.000000")` | Format the time as `YYYYMMDDHHmmss.microseconds` |
| `randBytes` | 4 random bytes that get converted to hex by `%x` |

A finished bundle ID looks like: `CX-20260522150405.000000-3f8a1c2b`

---

### ✏️ Quiz 2

**Task:** Create `sandbox/quiz2.go`.

Write a program that:
1. Creates a `string` variable called `bankName` containing `"Central Bank of Kenya"`
2. Creates an `int` variable called `transactionCount` containing `1247`
3. Creates a `float64` variable called `totalAmountKES` containing `98750.50`
4. Creates a `bool` variable called `systemOnline` set to `true`
5. Prints them all in a single formatted sentence using `fmt.Printf`

The output should look exactly like:
```
Bank: Central Bank of Kenya | Transactions: 1247 | Total: 98750.50 KES | Online: true
```

**Hint:** `fmt.Printf` uses `%s` for strings, `%d` for integers, `%.2f` for floats with 2 decimal places, and `%v` for booleans.

---

### ✅ Answer — Quiz 2

```go
package main

import "fmt"

func main() {
    bankName         := "Central Bank of Kenya"
    transactionCount := 1247
    totalAmountKES   := 98750.50
    systemOnline     := true

    fmt.Printf("Bank: %s | Transactions: %d | Total: %.2f KES | Online: %v\n",
        bankName, transactionCount, totalAmountKES, systemOnline)
}
```

**What each format verb means:**
- `%s` → insert a string
- `%d` → insert an integer (decimal number)
- `%.2f` → insert a float, rounded to 2 decimal places
- `%v` → insert any value using its default format (works for bool, slices, structs)
- `\n` → a newline character (goes to the next line)

**Expected output:**
```
Bank: Central Bank of Kenya | Transactions: 1247 | Total: 98750.50 KES | Online: true
```

---

---

# Chapter 3: Grouping Data

## Structs — Custom Data Blueprints

---

### 📖 Plain English

A single bank transaction is not just one value. It has many pieces: an ID, an amount, a receiver, a timestamp, and signatures. In Go, we group related data together into a **struct** (short for "structure").

Think of a struct like a form with labeled fields. Once you define the form (the `type`), you can fill it out many times to create many instances.

```
BANK TRANSACTION FORM
─────────────────────
ID:           [ TX-001      ]
Amount:       [ 5000.00 KES ]
ReceiverBank: [ Equity Bank ]
IsApproved:   [ true        ]
```

In Go code, that form is defined once and reused endlessly.

---

### 🔍 The Real Code — `cmd/gateway/main.go` Lines 41–51

```go
type Bundle struct {
    BundleID      string          `json:"bundle_id"`
    Timestamp     string          `json:"timestamp"`
    OriginalHash  string          `json:"original_hash"`
    EnrichedHash  string          `json:"enriched_hash"`
    PrevChainHash string          `json:"prev_chain_hash"`
    ChainHash     string          `json:"chain_hash"`
    Signatures    []SignatureEntry `json:"signatures"`
    QuorumStatus  string          `json:"quorum_status"`
    EnrichmentLog json.RawMessage `json:"enrichment_log"`
}
```

**Word-by-word breakdown:**

| Part | Meaning |
|------|---------|
| `type Bundle struct` | Define a new custom type called `Bundle`. Its shape follows. |
| `BundleID string` | A field called `BundleID` that holds text |
| `` `json:"bundle_id"` `` | A **struct tag**: tells the JSON toolbox to call this field `bundle_id` (not `BundleID`) in the output |
| `[]SignatureEntry` | A **slice** (list) of `SignatureEntry` items. The `[]` means "zero or more of these" |
| `json.RawMessage` | A special type that holds raw JSON without parsing it |

**To fill out (instantiate) a Bundle:**
```go
myBundle := Bundle{
    BundleID:     "CX-20260522-3f8a",
    Timestamp:    "2026-05-22T15:04:05Z",
    QuorumStatus: "QUORUM_MET",
}
```

**To read a specific field, use a dot `.`:**
```go
fmt.Println(myBundle.BundleID)     // prints: CX-20260522-3f8a
fmt.Println(myBundle.QuorumStatus) // prints: QUORUM_MET
```

---

### 🔍 The Real Code — `cmd/witness/main.go` Lines 74–80

```go
type witness struct {
    priv        ed25519.PrivateKey // The secret key — never shared
    pub         ed25519.PublicKey  // The public key — shared with the gateway
    fp          string             // Short fingerprint (first 16 chars of SHA-256)
    witnessName string             // "alpha", "beta", or "gamma"
    token       string             // Bearer token for authentication
}
```

Notice the field names are **lowercase** (`priv`, `pub`, `fp`). In Go, lowercase means the field is **private** — it cannot be accessed from outside this package. This is intentional security design: the private key should never be exposed to outside code.

---

### ✏️ Quiz 3

**Task:** Create `sandbox/quiz3.go`.

1. Define a struct called `BankTransaction` with these fields and JSON tags:

| Field Name | Type | JSON Tag |
|------------|------|----------|
| `ID` | `string` | `"id"` |
| `SenderBank` | `string` | `"sender_bank"` |
| `ReceiverBank` | `string` | `"receiver_bank"` |
| `AmountKES` | `float64` | `"amount_kes"` |
| `IsApproved` | `bool` | `"is_approved"` |

2. In `main()`, create a `BankTransaction` and fill in all fields with realistic values.
3. Use `json.Marshal` to convert it to JSON bytes, then print the JSON string.

---

### ✅ Answer — Quiz 3

```go
package main

import (
    "encoding/json"
    "fmt"
)

type BankTransaction struct {
    ID            string  `json:"id"`
    SenderBank    string  `json:"sender_bank"`
    ReceiverBank  string  `json:"receiver_bank"`
    AmountKES     float64 `json:"amount_kes"`
    IsApproved    bool    `json:"is_approved"`
}

func main() {
    tx := BankTransaction{
        ID:           "TX-2026-001",
        SenderBank:   "KCB Bank",
        ReceiverBank: "Equity Bank",
        AmountKES:    15750.00,
        IsApproved:   true,
    }

    // Convert the struct to JSON bytes
    jsonBytes, err := json.Marshal(tx)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }

    // Convert bytes to a readable string and print
    fmt.Println(string(jsonBytes))
}
```

**Expected output:**
```json
{"id":"TX-2026-001","sender_bank":"KCB Bank","receiver_bank":"Equity Bank","amount_kes":15750,"is_approved":true}
```

Notice how the JSON uses `sender_bank` (from the struct tag) instead of `SenderBank`. That is the struct tag doing its job.

---

---

# Chapter 4: Functions — Reusable Recipes

---

### 📖 Plain English

A **function** is a named, reusable recipe. You write it once, give it a name, and call it by name whenever you need it. Functions keep your code organized and prevent repetition.

A function can:
- Take **parameters** (ingredients you pass in)
- Return **values** (results it hands back)
- Return **multiple values** (Go allows this — most functions return a result AND an error)

**Basic structure:**
```
func functionName(parameter1 type, parameter2 type) returnType {
    // do things here
    return result
}
```

---

### 🔍 The Real Code — `cmd/witness/main.go` Lines 67–70

```go
// fingerprint returns the first 16 hex characters of SHA-256(pubkey).
func fingerprint(pub ed25519.PublicKey) string {
    h := sha256.Sum256(pub)               // Hash the public key bytes
    return hex.EncodeToString(h[:])[:16]  // Convert to hex, take first 16 chars
}
```

**Breakdown:**

| Part | Meaning |
|------|---------|
| `func fingerprint` | Define a function called `fingerprint` |
| `(pub ed25519.PublicKey)` | It takes one parameter named `pub` of type `ed25519.PublicKey` |
| `string` (after `)`) | It returns one `string` value |
| `sha256.Sum256(pub)` | Compute the SHA-256 hash of the public key. Returns `[32]byte` |
| `h[:]` | Convert the fixed-size array `[32]byte` to a flexible slice `[]byte` |
| `hex.EncodeToString(...)` | Convert bytes to a hex string like `"adf9bb79c93556f9..."` |
| `[:16]` | Slice it: take only characters 0 through 15 |
| `return` | Hand the result back to whatever called this function |

---

### 🔍 The Real Code — `cmd/gateway/main.go` Lines 125–128

```go
func sha256Hex(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}
```

This tiny function is one of the most important in CONNEX. Every ISO 8583 message and every ISO 20022 XML document gets passed through this function. The output is a "fingerprint" — if even a single character changes, the entire fingerprint changes completely. This is how tampering is detected.

---

### 🔍 The Real Code — Multiple Return Values — `cmd/witness/main.go` Lines 33–64

Go functions can return multiple values at once. The Witness keypair loader returns THREE things:

```go
func loadOrGenerate(keyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
    // ... does work ...
    return pub, priv, nil   // Return: public key, private key, no error
}
```

And the caller receives all three:

```go
pub, priv, err := loadOrGenerate(*keyPath)
```

The third return value `error` follows a Go convention: if it is `nil`, everything worked. If it is not `nil`, something went wrong and you should stop.

---

### ✏️ Quiz 4

**Task:** Create `sandbox/quiz4.go`.

Write TWO functions:

**Function 1:** `hashText(input string) string`
- Takes any string
- Computes its SHA-256 hash using `sha256.Sum256([]byte(input))`
- Returns the hex string of the hash

**Function 2:** `makeTransactionID(bankCode string, sequenceNumber int) string`
- Takes a bank code like `"KCB"` and a sequence number like `42`
- Returns a string formatted as: `"TX-KCB-042-20260527"` (bank code + zero-padded number + today's date)

In `main()`, call both functions and print their results.

---

### ✅ Answer — Quiz 4

```go
package main

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"
)

// hashText computes the SHA-256 hash of any string and returns it as hex.
func hashText(input string) string {
    h := sha256.Sum256([]byte(input)) // Convert string to bytes first, then hash
    return hex.EncodeToString(h[:])   // Convert the 32-byte hash to a hex string
}

// makeTransactionID creates a unique transaction ID from bank code + sequence + date.
func makeTransactionID(bankCode string, sequenceNumber int) string {
    today := time.Now().Format("20060102") // Format: YYYYMMDD
    return fmt.Sprintf("TX-%s-%03d-%s", bankCode, sequenceNumber, today)
    // %03d means: print the integer with at least 3 digits, zero-padded (so 42 becomes "042")
}

func main() {
    // Test hashText
    hash := hashText("Hello CONNEX")
    fmt.Println("SHA-256 hash:", hash)

    // Test makeTransactionID
    txID := makeTransactionID("KCB", 42)
    fmt.Println("Transaction ID:", txID)
}
```

**Expected output:**
```
SHA-256 hash: 3b5d5c3712955042212316173ccf37be9baaea1bc23b9f1ec95b938db4c4d96c
Transaction ID: TX-KCB-042-20260527
```

---

---

# Chapter 5: Methods — Functions Attached to Structs

---

### 📖 Plain English

A **method** is a function that "belongs to" a struct. Instead of being called like `doSomething(myStruct)`, it is called like `myStruct.doSomething()`.

You attach a function to a struct by adding a **receiver** before the function name:

```go
func (variableName *StructType) MethodName() returnType {
    // use variableName.FieldName to access the struct's data
}
```

The `*` before the type means the method works on the **original** struct, not a copy. This is important for large data structures. Without the `*`, Go makes a copy of the struct before calling the method, and any changes inside the method are thrown away.

---

### 🔍 The Real Code — `cmd/witness/main.go` Lines 82–93

```go
func (w *witness) handlePubkey(rw http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(rw, "GET required", http.StatusMethodNotAllowed)
        return
    }
    rw.Header().Set("Content-Type", "application/json")
    json.NewEncoder(rw).Encode(map[string]string{
        "witness":     w.witnessName,
        "public_key":  base64.StdEncoding.EncodeToString(w.pub),
        "fingerprint": w.fp,
    })
}
```

**Breakdown:**

| Part | Meaning |
|------|---------|
| `(w *witness)` | This method is attached to the `witness` struct. Inside this method, `w` refers to this specific witness |
| `w.witnessName` | Access the `witnessName` field of this witness using the dot `.` |
| `r.Method != http.MethodGet` | Check if the HTTP request method is GET. If not, reject it |
| `http.Error(rw, ..., 405)` | Send an HTTP 405 "Method Not Allowed" error response |
| `json.NewEncoder(rw).Encode(...)` | Convert the map to JSON and write it directly to the HTTP response |

---

### 🔍 The Real Code — `internal/iso8583/parser.go` Lines 91–101

```go
func (m *Message) AmountKES() float64 {
    s, ok := m.Fields[4]          // Look up field 4 (Transaction Amount) in the map
    if !ok || s == "" {           // If field 4 is missing or empty...
        return 0                  // ...return zero (no amount)
    }
    n, err := strconv.ParseInt(strings.TrimLeft(s, "0 "), 10, 64)
    if err != nil {
        return 0
    }
    return float64(n) / 100.0    // Divide by 100 because ISO 8583 stores amounts in cents
}
```

This method is how CONNEX reads the money amount from a legacy bank message. ISO 8583 stores `1000.00 KES` as the string `"000000100000"` (in cents, zero-padded). This method converts that to `1000.0`.

---

### ✏️ Quiz 5

**Task:** Create `sandbox/quiz5.go`.

1. Reuse your `BankTransaction` struct from Quiz 3
2. Add a method called `Summary()` that returns a `string` describing the transaction:
   `"[ID]: [SenderBank] → [ReceiverBank] | KES [AmountKES] | Approved: [IsApproved]"`
3. Add a second method called `IsLargeTransaction()` that returns a `bool`:
   - Returns `true` if `AmountKES` is greater than `100000`
   - Returns `false` otherwise
4. In `main()`, create a transaction and call both methods

---

### ✅ Answer — Quiz 5

```go
package main

import "fmt"

type BankTransaction struct {
    ID           string
    SenderBank   string
    ReceiverBank string
    AmountKES    float64
    IsApproved   bool
}

// Summary returns a human-readable description of the transaction.
func (t *BankTransaction) Summary() string {
    return fmt.Sprintf("%s: %s → %s | KES %.2f | Approved: %v",
        t.ID, t.SenderBank, t.ReceiverBank, t.AmountKES, t.IsApproved)
}

// IsLargeTransaction returns true if the amount exceeds 100,000 KES.
func (t *BankTransaction) IsLargeTransaction() bool {
    return t.AmountKES > 100000
}

func main() {
    tx := &BankTransaction{
        ID:           "TX-2026-001",
        SenderBank:   "KCB Bank",
        ReceiverBank: "Equity Bank",
        AmountKES:    250000.00,
        IsApproved:   true,
    }

    fmt.Println(tx.Summary())

    if tx.IsLargeTransaction() {
        fmt.Println("⚠️  Large transaction — flagged for compliance review")
    } else {
        fmt.Println("✅  Standard transaction — cleared")
    }
}
```

**Expected output:**
```
TX-2026-001: KCB Bank → Equity Bank | KES 250000.00 | Approved: true
⚠️  Large transaction — flagged for compliance review
```

---

---

# Chapter 6: Error Handling — Never Ignore a Problem

---

### 📖 Plain English

In a banking system, silent failures are catastrophic. If something goes wrong when writing to the ledger and the program just moves on, a transaction could be lost forever.

Go solves this with a rule: **functions that can fail return an `error` as their last value.** You MUST check it. If the error is `nil`, everything is fine. If it is not `nil`, something went wrong and you need to stop.

```go
result, err := someFunction()
if err != nil {
    // Handle the problem here — log it, return it, or stop the program
}
// If we reach this line, everything worked
```

---

### 🔍 The Real Code — `cmd/witness/main.go` Lines 47–63

```go
// Generate a fresh keypair
pub, priv, err := ed25519.GenerateKey(rand.Reader)
if err != nil {
    return nil, nil, fmt.Errorf("generate keypair: %w", err)
}

// Save the private key to disk with strict permissions (0600 = owner read/write only)
if err := os.WriteFile(privPath, priv, 0600); err != nil {
    return nil, nil, fmt.Errorf("write private key: %w", err)
}

// Save the public key to disk with looser permissions (0644 = anyone can read)
if err := os.WriteFile(pubPath, pub, 0644); err != nil {
    return nil, nil, fmt.Errorf("write public key: %w", err)
}
```

**Breakdown:**

| Part | Meaning |
|------|---------|
| `pub, priv, err :=` | Receive three values from the function |
| `if err != nil` | "If there was an error..." |
| `return nil, nil, fmt.Errorf(...)` | Stop this function and return the error upward |
| `fmt.Errorf("generate keypair: %w", err)` | Wrap the error with context. The `%w` verb wraps the original error so it can be inspected later |
| `0600` | Unix file permission: only the owner can read and write. The private key must stay secret |
| `0644` | Unix file permission: anyone can read but only the owner can write. Public keys are meant to be shared |

---

### 🔍 The Real Code — `cmd/gateway/main.go` Lines 268–272

```go
if err != nil {
    slog.Error("db write failed", "bundle", bundleID, "err", err)
    http.Error(w, "database write error", http.StatusInternalServerError)
    return
}
```

If writing to the database fails, CONNEX:
1. Logs the error with structured fields (bundle ID + the actual error)
2. Returns an HTTP 500 error to the client
3. Stops the handler with `return` (no bundle is returned)

This is correct banking behavior: never claim a transaction succeeded if you cannot prove it was stored.

---

### ✏️ Quiz 6

**Task:** Create `sandbox/quiz6.go`.

Write a function called `readTransactionFile(filename string) (string, error)` that:
1. Uses `os.ReadFile(filename)` to read a file
2. If it fails, returns an empty string `""` and a wrapped error: `fmt.Errorf("readTransactionFile: %w", err)`
3. If it succeeds, returns the file content as a string and `nil` for the error

In `main()`:
1. Call `readTransactionFile("transactions.json")` (this file does NOT exist)
2. Check the error properly
3. If there is an error, print: `"Failed to load transactions: [error message]"`
4. If there is no error, print the file contents

---

### ✅ Answer — Quiz 6

```go
package main

import (
    "fmt"
    "os"
)

// readTransactionFile reads a file and returns its contents as a string.
// If the file cannot be read, it returns an empty string and a descriptive error.
func readTransactionFile(filename string) (string, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        // Wrap the error with context so the caller knows where it came from
        return "", fmt.Errorf("readTransactionFile: %w", err)
    }
    // Convert the raw bytes to a string and return with nil error
    return string(data), nil
}

func main() {
    content, err := readTransactionFile("transactions.json")
    if err != nil {
        fmt.Println("Failed to load transactions:", err)
        return
    }
    fmt.Println("File contents:", content)
}
```

**Expected output** (since `transactions.json` does not exist):
```
Failed to load transactions: readTransactionFile: open transactions.json: The system cannot find the file specified.
```

Notice how the error message chains together: your wrapper message `"readTransactionFile:"` + the original OS error. This chain is very useful for debugging in a production banking system.

---

---

# Chapter 7: Goroutines — Doing Things at the Same Time

---

### 📖 Plain English

Normally a computer executes instructions one at a time, line by line. But CONNEX needs to ask 3 Witness nodes for cryptographic signatures **at the exact same moment**. If it asked them one by one, it would take 3× as long.

In Go, you put the keyword `go` in front of a function call to launch it instantly in the background. This background task is called a **goroutine**. It is extremely lightweight — you can run thousands of them simultaneously.

**Without goroutines (slow):**
```
Ask Witness Alpha → Wait 50ms → Get signature
Ask Witness Beta  → Wait 50ms → Get signature
Ask Witness Gamma → Wait 50ms → Get signature
Total time: 150ms
```

**With goroutines (fast):**
```
Ask Witness Alpha ──┐
Ask Witness Beta  ──┼──→ All three run at the same time
Ask Witness Gamma ──┘
Total time: ~50ms (the slowest one)
```

---

### 🔍 The Real Code — `cmd/gateway/main.go` Lines 87–121

This is the most important function in the entire CONNEX system:

```go
func collectSignatures(witnesses []string, tokens []string, hashBytes []byte, timeout time.Duration) []SignatureEntry {

    // A temporary struct to hold either a good signature OR an error
    type result struct {
        sig *SignatureEntry
        err error
    }

    // Create a channel — a pipe that goroutines send their results through
    ch := make(chan result, len(witnesses))

    // Launch one goroutine per witness — all three start at the same time
    for i, w := range witnesses {
        w := w // Important: capture the variable for the goroutine (explained below)
        var token string
        if i < len(tokens) {
            token = tokens[i]
        }
        go func() {
            sig, err := requestSignature(w, token, hashBytes, timeout)
            ch <- result{sig, err} // Send result through the pipe
        }()
    }

    // Now collect results as they arrive
    var sigs []SignatureEntry
    deadline := time.After(timeout) // Set a countdown timer

    for range witnesses {
        select {
        case r := <-ch:         // A result came through the pipe
            if r.err != nil {
                slog.Warn("witness error", "err", r.err)
            } else {
                sigs = append(sigs, *r.sig) // Add the signature to our list
            }
        case <-deadline:        // The countdown timer ran out
            slog.Warn("witness timeout reached", "collected", len(sigs))
            return sigs         // Return whatever we have collected so far
        }
    }
    return sigs
}
```

**Key concepts used here:**

| Concept | What it is |
|---------|-----------|
| `ch := make(chan result, ...)` | A **channel** — a safe pipe for passing data between goroutines |
| `go func() { ... }()` | Launch an anonymous function as a goroutine (background task) |
| `ch <- result{sig, err}` | **Send** a result into the channel pipe |
| `r := <-ch` | **Receive** a result from the channel pipe |
| `select { case ...: case ...: }` | Wait for whichever event happens first |
| `time.After(timeout)` | A channel that receives one value after the timer expires |

---

### ✏️ Quiz 7

**Task:** Create `sandbox/quiz7.go`.

Simulate 3 witness nodes responding at different speeds:

1. Create a channel that carries strings: `make(chan string, 3)`
2. Launch 3 goroutines:
   - Goroutine 1: sleeps 1 second, then sends `"Alpha signed ✓"` to the channel
   - Goroutine 2: sleeps 2 seconds, then sends `"Beta signed ✓"` to the channel
   - Goroutine 3: sleeps 5 seconds, then sends `"Gamma signed ✓"` to the channel
3. In `main()`, use a `for` loop that runs exactly 3 times. Inside it, use a `select` with:
   - A `case` to receive from the channel and print the message
   - A `case <-time.After(3 * time.Second)` that prints `"Timeout — quorum check"` and stops

**Expected behavior:** Alpha and Beta respond in time. Gamma does not.

---

### ✅ Answer — Quiz 7

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    results := make(chan string, 3) // Buffered channel for 3 results

    // Launch 3 goroutines simultaneously
    go func() {
        time.Sleep(1 * time.Second)
        results <- "Alpha signed ✓"
    }()

    go func() {
        time.Sleep(2 * time.Second)
        results <- "Beta signed ✓"
    }()

    go func() {
        time.Sleep(5 * time.Second) // Too slow — will miss the deadline
        results <- "Gamma signed ✓"
    }()

    // Collect results with a 3-second total deadline
    signaturesCollected := 0
    for i := 0; i < 3; i++ {
        select {
        case msg := <-results:
            fmt.Println("Received:", msg)
            signaturesCollected++
        case <-time.After(3 * time.Second):
            fmt.Println("Timeout — quorum check")
            fmt.Printf("Signatures collected: %d/3\n", signaturesCollected)
            if signaturesCollected >= 2 {
                fmt.Println("QUORUM_MET ✓")
            } else {
                fmt.Println("QUORUM_FAILED ✗")
            }
            return // Stop waiting
        }
    }
}
```

**Expected output:**
```
Received: Alpha signed ✓
Received: Beta signed ✓
Timeout — quorum check
Signatures collected: 2/3
QUORUM_MET ✓
```

This output perfectly mirrors what CONNEX does every time a bank transaction is processed!

---

---

# Chapter 8: Reading the Real Code

## You Are Now Ready

You have now learned every concept used in the CONNEX Gateway and Witness Node. Let's prove it by reading the real `main()` function:

Open `cmd/witness/main.go` and find `main()` starting at line 164. Here is what you will see and what you now know it means:

```go
func main() {
    // Chapter 1: package main means this is the entry point
    // These are command-line flags (from the "flag" toolbox)
    port    := flag.Int("port", 8091, "Port to listen on")
    keyPath := flag.String("keypath", "keys/witness.key", "Path for Ed25519 keypair")
    name    := flag.String("name", "witness", "Human-readable witness name")
    token   := flag.String("token", "", "Shared authentication token secret")
    flag.Parse() // Actually read the command-line arguments

    // Chapter 6: Error handling — loadOrGenerate returns 3 values
    pub, priv, err := loadOrGenerate(*keyPath)
    if err != nil {
        slog.Error("keypair setup failed", "err", err)
        os.Exit(1) // Stop the program with an error code
    }

    // Chapter 4 & 5: Call the fingerprint function, build a witness struct
    fp := fingerprint(pub)
    w := &witness{priv: priv, pub: pub, fp: fp, witnessName: *name, token: *token}

    // Chapter 5: Register the HTTP route handlers (methods on the witness struct)
    mux := http.NewServeMux()
    mux.HandleFunc("/v1/pubkey", w.handlePubkey)
    mux.HandleFunc("/v1/sign",   w.handleSign)
    mux.HandleFunc("/health",    w.handleHealth)

    // Chapter 6: Start the server and handle errors
    addr := fmt.Sprintf(":%d", *port)
    if err := http.ListenAndServe(addr, mux); err != nil {
        slog.Error("server error", "err", err)
        os.Exit(1)
    }
}
```

Every single line — you now understand it.

---

## Your Final Challenge

Open `cmd/gateway/main.go` and find the `handleCoordinate` function starting at line 148. It has 12 numbered steps in the comments. For each step, write in a notebook which **chapter** from this course the code in that step relates to.

When you can do that, you are ready to start contributing to CONNEX.
