# Connex Deterministic Rules Enrichment Engine

This document explains the Connex Rules Engine, which translates raw ISO 8583 banking codes into descriptive, regulatory-compliant attributes for modern credit transfers (ISO 20022). It details how to configure `rules.yaml` and how the execution engine (`internal/enrichment/engine.go`) compiles and runs these rules in Go.

---

## 1. What is Transaction Enrichment?

Legacy cards and ATM networks use cryptic codes (e.g., Merchant Category Code `5411` or Processing Code `000000`). Modern settlement networks (like RTGS systems running ISO 20022) require explicit explanations of why money is moving (e.g., Category Purpose Code `GDDS` for purchase of goods, or `SALA` for salary payments).

The **Enrichment Engine** fills this gap by evaluating a list of business rules against transaction details:
- **Input**: Transaction Amount, Currency, Processing Code, Merchant Category Code, and Participating Financial Institutions.
- **Output**: Regulatory Reporting Code, Purpose Code, Settlement Method, Local Instrument, Charge Bearer policy, Service Level, and an **Enrichment Provenance Log**.

---

## 2. Configuration (`rules.yaml`)

Rules are stored in `rules.yaml`. Each rule specifies a **Condition** and an **Outcome**:

```yaml
version: "v1.0"
rules:
  - id: "RULE-CBK-HIGH-VALUE-KES"
    description: "High-value local KES transaction requires RTGS settlement"
    condition:
      amount_min_kes: 1000000.00
      currency_iso4217: "404"
    outcome:
      field: "settlement_method"
      value: "INDA"
      local_instrument: "RTGS"
      confidence: 1.0
```

### Condition Criteria
Rules can match on multiple criteria:
- `amount_min_kes` / `amount_max_kes`: Numeric ranges converted to KES.
- `currency_iso4217` / `currency_iso4217_not`: Matches or excludes specific currency codes.
- `processing_code` / `processing_code_prefix`: Matches transaction types (e.g., prefix `01` for withdrawals).
- `merchant_type` / `merchant_type_in`: Matches Merchant Category Codes (MCC).
- `cross_institutional`: Boolean flag indicating if the sender and receiver banks are different.
- `always`: Standard catch-all fallback selector.

---

## 3. Go Execution Mechanics

Let's trace how [engine.go](file:///c:/Users/roych/OFFICIAL%20MVP/internal/enrichment/engine.go) loads and evaluates these rules.

### 3.1 Compiling Files into Binaries with `go:embed`
To prevent the application from failing if an external configuration file is missing at runtime, Go provides the `embed` package. This compiles the content of `rules.yaml` directly into the gateway binary:

```go
import _ "embed"

//go:embed rules.yaml
var rulesYAML []byte
```
*Note: The blank import `_ "embed"` registers the embedding handler with the compiler.*

### 3.2 Loading Configuration in `init()`
In Go, any function named `init()` executes automatically before the application's `main()` function starts. We use it to parse the embedded YAML file:

```go
var loadedRules []rule

func init() {
    var rf rulesFile
    // Unmarshal parses rulesYAML and stores the Go structures in rf
    if err := yaml.Unmarshal(rulesYAML, &rf); err != nil {
        panic(fmt.Sprintf("enrichment: failed to load rules.yaml: %v", err))
    }
    loadedRules = rf.Rules
}
```

### 3.3 Struct Tagging
Notice the tags attached to the struct fields:
```go
type ruleOutcome struct {
    Field      string  `yaml:"field"`
    Value      string  `yaml:"value"`
    Confidence float64 `yaml:"confidence"`
}
```
These tags tell the YAML parser (`gopkg.in/yaml.v3`) how to map lowercase YAML keys to uppercase Go struct fields. The same concept is used for JSON serialization using `json:"..."`.

### 3.4 Sequential Rule Matching
When a transaction is processed, `Enrich` walks through the compiled rules in order. The first rule to set a field wins, preventing subsequent rules from overwriting it.

```go
func Enrich(in *Input) (*Result, error) {
    result := &Result{Log: make(Log)}

    for _, r := range loadedRules {
        // Check if the input fits the rule's criteria
        if !r.Condition.matches(in, result) {
            continue
        }

        // Prepare the log entry detailing who matched this outcome
        entry := FieldEntry{
            Value:       r.Outcome.Value,
            Source:      r.ID,
            Description: r.Description,
            Confidence:  1.0,
        }

        // Apply outcome to the relevant field if not already set
        switch r.Outcome.Field {
        case "purpose_code":
            if result.PurposeCode == "" {
                result.PurposeCode = r.Outcome.Value
                result.Log["purpose_code"] = entry
            }
        // ... handling other fields ...
        }
    }

    // Apply defaults for any fields that remain empty (fallback logic)
    if result.ChargeBearer == "" {
        result.ChargeBearer = "SHAR"
        result.Log["charge_bearer"] = FieldEntry{Value: "SHAR", Source: "DEFAULT", Confidence: 1.0}
    }
    ...
    return result, nil
}
```

### 3.5 Provenance Logging
Every matched field is added to the `result.Log` map. This creates a detailed audit trail explaining *why* the rules engine made its decisions:

```json
{
  "purpose_code": {
    "value": "OTHR",
    "source": "RULE-CBK-DEFER-PURPOSE",
    "description": "Deferred purpose fallback",
    "confidence": 1.0
  },
  "settlement_method": {
    "value": "INDA",
    "source": "RULE-CBK-HIGH-VALUE-KES",
    "description": "High-value local KES transaction requires RTGS settlement",
    "confidence": 1.0
  }
}
```

---

## 4. Replicating this Engine (Beginner Guide)

To expand or build a rules engine in Go:
1.  **Strict Rule Ordering**: Always structure your YAML file with specific, high-priority rules at the top, and broad, catch-all default rules at the bottom.
2.  **Immutability**: Once a rule outcome sets a field, make sure you check `if result.Field == ""` to preserve the priority mapping and prevent lower-priority rules from overriding it.
3.  **Embed Configs**: Use `//go:embed` for static rules configuration. If you need dynamic updates, read the file at runtime or implement an administrative reload endpoint protected by a mutex lock.
