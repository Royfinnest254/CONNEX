<picture>
  <source media="(prefers-color-scheme: dark)" srcset="connex_logo_dark.png">
  <source media="(prefers-color-scheme: light)" srcset="connex_logo_light.png">
  <img alt="CONNEX" src="connex_logo_light.png" width="220">
</picture>

---

# Go Programming: Learn by Reading CONNEX

> **This course is built 100% from the actual CONNEX source code.**
> Every concept you learn here is illustrated with real lines from the real banking system. By the end, you will be able to open any Go file in this project and understand every single line.

---

## How This Course Works

1. **Read the lesson** — I explain the concept in plain English.
2. **See the real code** — I show you the exact lines from CONNEX and explain them word by word.
3. **Take the quiz** — A challenge for you to write in your own sandbox folder without touching the project files.

> ⚠️ **Safety Rule:** Never edit any file inside `cmd/` or `internal/`. Create a folder on your Desktop called `sandbox/` and practice there. You cannot break the bank from there.

---

## Chapter 1: The Starting Line — `package main` and `import`

### 1.1 The Lesson

Every Go program starts with two things: a **package declaration** and an **import block**. Think of it like this:

- **`package main`** = "This file is a program that can be turned on."
- **`import (...)`** = "These are the toolboxes I need to borrow before I start working."

Go comes with hundreds of free toolboxes built-in (like `fmt` for printing text, `net/http` for web servers, and `crypto/sha256` for cryptography). You just have to say which ones you need.

---

### 1.2 The Real Code — `cmd/witness/main.go` Lines 11–27

```go
package main                    // Line 11: "This is a runnable program."

import (                        // Line 13: "I need to borrow these toolboxes:"
    "crypto/ed25519"            // The toolbox for Ed25519 cryptographic signatures
    "crypto/rand"               // The toolbox for generating random bytes
    "crypto/sha256"             // The toolbox for SHA-256 hashing
    "encoding/base64"           // The toolbox for converting binary data to readable text
    "encoding/hex"              // The toolbox for converting binary data to hex strings
    "encoding/json"             // The toolbox for reading and writing JSON
    "flag"                      // The toolbox for reading command-line arguments
    "fmt"                       // The toolbox for printing text ("format")
    "log/slog"                  // The toolbox for structured log messages
    "net/http"                  // The toolbox for creating web servers
    "os"                        // The toolbox for reading and writing files
    "path/filepath"             // The toolbox for working with file and folder paths
    "time"                      // The toolbox for dates, times, and sleeping
)
```

**Word-by-word breakdown:**
- `"crypto/ed25519"` — The `/` doesn't mean division here. It's a folder path. This imports the `ed25519` package from inside Go's built-in `crypto` folder.
- The parentheses `(...)` around the imports let you list multiple toolboxes at once instead of writing `import` over and over.

---

### 1.3 Quiz 1 ✏️

**Your Challenge:**
Create a file called `sandbox/quiz1.go` on your Desktop. Write a program that:
1. Declares `package main`
2. Imports the `fmt` and `time` toolboxes
3. In the `main()` function, prints the message: `"CONNEX Sandbox: System is online"`
4. Also prints the current time using `time.Now()`

**Run it with:** `go run quiz1.go`

---

## Chapter 2: Storing Data — Variables and Data Types

### 2.1 The Lesson

A **variable** is a labeled box in your computer's memory. In Go, you create a variable using `:=`. The computer automatically figures out what type of data you are storing (text, a number, etc.).

Go has a few common data types:
- **`string`** — Text, always wrapped in double quotes: `"Alice"`
- **`int`** — A whole number: `42`
- **`float64`** — A number with a decimal: `3.14`
- **`bool`** — True or false: `true`
- **`[]byte`** — A list of raw binary bytes (very important in CONNEX)

---

### 2.2 The Real Code — `cmd/witness/main.go` Lines 34–36

```go
func loadOrGenerate(keyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
    privPath := keyPath          // Line 34: Create a box called "privPath", put keyPath's value inside it
    pubPath := keyPath + ".pub"  // Line 35: Create a box called "pubPath", same value but with ".pub" at the end
```

**Word-by-word breakdown:**
- `privPath := keyPath` — The `:=` operator creates a new variable AND fills it at the same time.
- `keyPath + ".pub"` — The `+` operator on strings just glues them together. If `keyPath` is `"keys/witness.key"`, then `pubPath` becomes `"keys/witness.key.pub"`.

---

### 2.3 The Real Code — `cmd/gateway/main.go` Line 198

```go
bundleID := fmt.Sprintf("CX-%s-%x", time.Now().UTC().Format("20060102150405.000000"), randBytes)
```

This single line creates a unique ID for every bank transaction. Let's break it apart:
- `bundleID :=` — Creates a new variable called `bundleID`.
- `fmt.Sprintf(...)` — Builds a text string by filling in a template. The `"CX-%s-%x"` is the template:
  - `%s` means "insert a string here"
  - `%x` means "insert bytes here, but convert them to a hexadecimal string"
- `time.Now().UTC().Format("20060102150405.000000")` — Gets the current time and formats it.

**Result:** A bundle ID looks like `CX-20260522150405.000000-3f8a1c2b`

---

### 2.4 Quiz 2 ✏️

**Your Challenge:**
Create `sandbox/quiz2.go`. Write a program that:
1. Creates a `string` variable called `bankName` with the value `"Central Bank of Kenya"`
2. Creates an `int` variable called `transactionCount` with the value `1247`
3. Creates a `float64` variable called `totalAmount` with the value `98750.50`
4. Prints all three in a single sentence like: `"Central Bank of Kenya processed 1247 transactions worth 98750.50 KES"`

**Hint:** Use `fmt.Printf` with `%s`, `%d`, and `%.2f` placeholders.

---

## Chapter 3: Grouping Data — Structs

### 3.1 The Lesson

A single bank transaction is not just one value — it has many pieces of data: an ID, an amount, a receiver, a timestamp, signatures. In Go, we group related data into a **struct** (short for "structure"). It is like a custom form with labeled fields.

---

### 3.2 The Real Code — `cmd/gateway/main.go` Lines 41–51

```go
type Bundle struct {
    BundleID      string           `json:"bundle_id"`       // The unique transaction ID
    Timestamp     string           `json:"timestamp"`        // When the transaction happened
    OriginalHash  string           `json:"original_hash"`   // Fingerprint of the original message
    EnrichedHash  string           `json:"enriched_hash"`   // Fingerprint of the translated message
    PrevChainHash string           `json:"prev_chain_hash"` // Link to the previous transaction
    ChainHash     string           `json:"chain_hash"`       // This transaction's chain link
    Signatures    []SignatureEntry `json:"signatures"`       // The witness signatures (a list)
    QuorumStatus  string           `json:"quorum_status"`   // Was quorum achieved?
    EnrichmentLog json.RawMessage  `json:"enrichment_log"`  // The rules that were applied
}
```

**Word-by-word breakdown:**
- `type Bundle struct { ... }` — "Define a new custom data type called `Bundle`. Its shape is defined by everything inside the curly braces."
- `BundleID string` — A field named `BundleID` that holds text.
- `` `json:"bundle_id"` `` — This is a **struct tag**. It tells the JSON toolbox: "When you write this to a JSON file, call it `bundle_id` (lowercase), not `BundleID`." This is how CONNEX keeps its JSON output clean and readable.
- `[]SignatureEntry` — The `[]` at the front means this is a **list** (called a slice in Go) of `SignatureEntry` items. One transaction can have multiple witness signatures.

---

### 3.3 The Real Code — `cmd/witness/main.go` Lines 74–80

```go
type witness struct {
    priv        ed25519.PrivateKey  // The witness's secret signing key
    pub         ed25519.PublicKey   // The witness's public key (shareable)
    fp          string              // A short fingerprint identifier
    witnessName string              // "alpha", "beta", or "gamma"
    token       string              // The security token for authentication
}
```

This is the "shape" of a Witness node. Every witness has these 5 fields that describe who it is and what keys it holds.

---

### 3.4 Quiz 3 ✏️

**Your Challenge:**
Create `sandbox/quiz3.go`. Define a struct called `BankTransaction` with these fields:
- `ID` (string)
- `SenderBank` (string)
- `ReceiverBank` (string)
- `AmountKES` (float64)
- `IsApproved` (bool)

Add JSON struct tags to all fields (use lowercase, underscore-separated names like `sender_bank`).

In `main()`, create one transaction, fill in all the fields, then use `json.Marshal` to convert it to JSON and print it.

---

## Chapter 4: Functions — Reusable Recipes

### 4.1 The Lesson

A **function** is a named recipe. You write it once, give it a name, and call that name whenever you need to run it. Functions can take "ingredients" (inputs called **parameters**) and give you something back (an **return value**).

---

### 4.2 The Real Code — `cmd/witness/main.go` Lines 67–70

```go
// fingerprint returns the first 16 hex characters of SHA-256(pubkey).
func fingerprint(pub ed25519.PublicKey) string {
    h := sha256.Sum256(pub)
    return hex.EncodeToString(h[:])[:16]
}
```

**Word-by-word breakdown:**
- `func fingerprint(...)` — "Define a recipe called `fingerprint`."
- `(pub ed25519.PublicKey)` — The recipe takes one ingredient: a public key, which we will call `pub` inside the recipe.
- `string` (after the parentheses) — The recipe will give back a `string` when it is done.
- `sha256.Sum256(pub)` — Hash the public key bytes using SHA-256.
- `hex.EncodeToString(h[:])` — Convert the raw hash bytes to a readable hex string.
- `[:16]` — Take only the first 16 characters of that hex string. This creates a short, readable fingerprint.
- `return` — Hand the result back to whoever called this recipe.

---

### 4.3 The Real Code — `cmd/gateway/main.go` Lines 125–128

```go
func sha256Hex(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}
```

This is one of the most important functions in CONNEX. Every transaction is "fingerprinted" by this function. If even one byte of the transaction data changes, the fingerprint (hash) will be completely different, which is how tamper detection works.

---

### 4.4 Quiz 4 ✏️

**Your Challenge:**
Create `sandbox/quiz4.go`. Write a function called `makeTransactionID` that:
- Takes one parameter: a `bankName` string
- Gets the current time using `time.Now()`
- Builds and returns a string in this format: `"TX-CENTRALBANK-20260522"` (the bank name in uppercase + the date)

**Hint:** Use `strings.ToUpper(bankName)` and `time.Now().Format("20060102")`.

Call your function from `main()` and print the result.

---

## Chapter 5: Methods — Functions Attached to Structs

### 5.1 The Lesson

In Go, you can attach a function directly to a struct. This is called a **method**. It is like giving a form (struct) its own built-in actions. You call it using a dot `.`

---

### 5.2 The Real Code — `cmd/witness/main.go` Lines 82–93

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

**Word-by-word breakdown:**
- `func (w *witness)` — This function is attached to the `witness` struct. The `w` is how we refer to "this specific witness" inside the function. The `*` means we are working with the original witness data, not a copy.
- `handlePubkey(rw http.ResponseWriter, r *http.Request)` — The function takes two parameters: `rw` (where we write the response) and `r` (the incoming request from the browser or client).
- `r.Method != http.MethodGet` — Check if the incoming HTTP request was a GET request. If it is not, reject it.
- `json.NewEncoder(rw).Encode(...)` — Build a JSON object and send it directly as the response.
- `w.witnessName` — Access the `witnessName` field of this specific witness using the dot `.`.

---

### 5.3 The Real Code — `internal/iso8583/parser.go` Lines 91–101

```go
func (m *Message) AmountKES() float64 {
    s, ok := m.Fields[4]       // Look inside the message for field number 4 (the amount field)
    if !ok || s == "" {         // If field 4 doesn't exist, or is empty...
        return 0                // ...return zero
    }
    n, err := strconv.ParseInt(strings.TrimLeft(s, "0 "), 10, 64)
    if err != nil {
        return 0
    }
    return float64(n) / 100.0  // Divide by 100 because ISO 8583 stores amounts in cents
}
```

This method is attached to `Message`. It reads field 4 (the amount) from the raw bank message and converts it from cents to a proper decimal number. If a bank sends `000000100000`, it means 1000.00 KES.

---

### 5.4 Quiz 5 ✏️

**Your Challenge:**
Using your `BankTransaction` struct from Quiz 3, add a **method** called `Summary()` that:
- Takes no parameters
- Returns a string that reads: `"Transaction [ID]: [SenderBank] sent [AmountKES] KES to [ReceiverBank]"`

Call `myTransaction.Summary()` in `main()` and print the result.

---

## Chapter 6: The Biggest Concept — Goroutines (Doing Things at the Same Time)

### 6.1 The Lesson

Usually a computer does things one at a time. But CONNEX needs to ask 3 different Witness nodes for signatures **simultaneously**. If it asked them one by one, it would take 3× as long.

In Go, you put the word `go` in front of a function call to launch it in the background instantly. This is called a **goroutine**. It is like asking 3 people to do a job at the same time instead of waiting for each one to finish.

---

### 6.2 The Real Code — `cmd/gateway/main.go` Lines 87–121

This is the most important function in the entire CONNEX system. Let's read it slowly:

```go
func collectSignatures(witnesses []string, tokens []string, hashBytes []byte, timeout time.Duration) []SignatureEntry {
```
- This function takes a list of witness addresses, their tokens, the hash to sign, and a timeout.
- It returns a list of signatures.

```go
    type result struct {
        sig *SignatureEntry
        err error
    }
    ch := make(chan result, len(witnesses))
```
- We define a temporary `result` struct to hold either a signature OR an error.
- `ch := make(chan result, len(witnesses))` — We create a **channel**. Think of it as a pipe. Goroutines will push their results through this pipe.

```go
    for i, w := range witnesses {
        w := w
        go func() {
            sig, err := requestSignature(w, token, hashBytes, timeout)
            ch <- result{sig, err}   // Push the result into the pipe
        }()
    }
```
- `for i, w := range witnesses` — Loop through every witness address.
- `go func() { ... }()` — For EACH witness, launch a goroutine immediately. All 3 run at the exact same time.
- `ch <- result{sig, err}` — When a goroutine finishes, it pushes its result into the channel pipe.

```go
    var sigs []SignatureEntry
    deadline := time.After(timeout)
    for range witnesses {
        select {
        case r := <-ch:                           // A result arrived through the pipe
            if r.err != nil {
                slog.Warn("witness error", "err", r.err)
            } else {
                sigs = append(sigs, *r.sig)       // Add the signature to our list
            }
        case <-deadline:                          // The timer ran out
            slog.Warn("witness timeout reached", "collected", len(sigs))
            return sigs                           // Return whatever we collected so far
        }
    }
    return sigs
}
```
- `select { ... }` — Wait for whichever happens first: a result arrives, OR the timer runs out.
- If a witness is offline, the `deadline` case fires and we return whatever signatures we already collected.
- This is how CONNEX achieves **degraded mode**: if 1 witness is offline, 2 signatures still = quorum met.

---

### 6.3 Quiz 6 ✏️

**Your Challenge:**
Create `sandbox/quiz6.go`. Simulate 3 witness nodes:
1. Create a channel called `results` that carries strings: `make(chan string, 3)`
2. Launch 3 goroutines. Each should sleep for a different time (1 second, 2 seconds, and 5 seconds), then send a message like `"Witness Alpha signed"` into the channel.
3. In `main()`, use a `select` statement inside a loop. Add a `time.After(3 * time.Second)` timeout case.
4. Your program should print the first two witnesses' messages, but timeout before the 5-second witness finishes.

---

## Chapter 7: Error Handling — Never Ignore Problems

### 7.1 The Lesson

In Go, functions that can fail return **two things**: the result AND an error. You always check the error. This is one of Go's most important rules. Ignoring errors in a banking system would be catastrophic.

---

### 7.2 The Real Code — `cmd/witness/main.go` Lines 47–50

```go
pub, priv, err := ed25519.GenerateKey(rand.Reader)
if err != nil {
    return nil, nil, fmt.Errorf("generate keypair: %w", err)
}
```

**Word-by-word breakdown:**
- `pub, priv, err :=` — This function returns 3 things at once: a public key, a private key, and an error.
- `if err != nil` — "If there WAS an error (i.e. err is not empty)..."
- `return nil, nil, fmt.Errorf(...)` — "...stop everything and report the error upward."
- `%w` inside `fmt.Errorf` — Wraps the original error so it can be traced back to its source.

---

### 7.3 Quiz 7 ✏️

**Your Challenge:**
Create `sandbox/quiz7.go`. Write a function called `readConfig` that:
- Takes a `filename` string parameter
- Uses `os.ReadFile(filename)` to read a file
- Returns the file contents as a `string` AND an `error`
- If `os.ReadFile` fails, return an empty string and the error wrapped with `fmt.Errorf("readConfig failed: %w", err)`

In `main()`, call `readConfig("nonexistent.txt")`, check the error, and print a meaningful message if it fails.

---

## Chapter 8: Putting It All Together — Read the Real Code

You now know enough to read the real CONNEX code. Open this file: `cmd/witness/main.go`

Read the `main()` function starting at line 164. You will now recognize:
- **Line 165–168**: Variables being created from command-line flags
- **Line 171**: A function call that returns two values + an error
- **Line 172–175**: Error handling
- **Line 178**: Building a `witness` struct using the `&` operator (creating a pointer)
- **Line 187–190**: Registering HTTP route handlers
- **Line 193–196**: Starting the web server and handling errors

Congratulations — you can now read production banking code.

---

## What To Do Next

1. Complete all 7 quizzes in your sandbox folder.
2. Open `cmd/gateway/main.go` and read through `handleCoordinate()` from line 148.
3. Try to follow the 12 numbered steps in the comments (steps 1–12) and identify which concepts from this course each step uses.

If you get stuck on any step, come back and ask — I am right here!
