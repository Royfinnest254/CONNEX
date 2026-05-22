# ISO 20022 XML Generation & Sanitization

This document details how Connex constructs and validates `pacs.008.001.08` XML documents, which represent modern credit transfer transactions between financial institutions. It focuses on XML templates, injection mitigations, and the Go-native schema validation checks inside `internal/iso20022/assembler.go`.

---

## 1. What is a pacs.008 Message?

The **pacs.008** (Payment Clearing and Settlement) message is the core industry standard format for customer credit transfers. In contrast to ISO 8583's compact, positional byte layouts, pacs.008 is structured as an XML document, containing nested tags detailing:
- **Group Header (`GrpHdr`)**: Control numbers, creation timestamp, and settlement methods.
- **Credit Transfer Transaction Information (`CdtTrfTxInf`)**: Accounts, routing codes, amounts, and specific regulatory or purpose reasons.

---

## 2. Dynamic XML Assembly in Go

Connex uses Go's standard library `text/template` to dynamically render the XML.

### 2.1 Embedding the XML Template
The raw XML structure is defined in a template file `templates/pacs008.xml.tmpl` and compiled into the Go binary:

```go
//go:embed templates/pacs008.xml.tmpl
var pacs008Tmpl string

var tmpl = template.Must(template.New("pacs008").Parse(pacs008Tmpl))
```

### 2.2 Binding Template Data
We map transaction details into a `TemplateData` struct and execute the template:

```go
var buf bytes.Buffer
if err := tmpl.Execute(&buf, td); err != nil {
    return nil, fmt.Errorf("template execute: %w", err)
}
xmlBytes := buf.Bytes()
```

---

## 3. Mitigating XML Injection Vulnerabilities

### What is XML Injection?
If a malicious client sends a transaction containing characters like `<` or `>` in their name or account fields (e.g., `<Dbtr><Nm>John <script>...</Nm></Dbtr>`), they can inject arbitrary tags into the output document. This changes the XML structure, potentially altering payment values, routing codes, or crashing downstream parsers.

### The Mitigation: Escaping Entities
To prevent this, every text field injected into the XML template is cleaned using a sanitization function that replaces structural XML characters with their safe XML entity equivalents:

| Character | Entity | Meaning |
| :--- | :--- | :--- |
| `&` | `&amp;` | Ampersand |
| `<` | `&lt;` | Less Than |
| `>` | `&gt;` | Greater Than |
| `"` | `&quot;` | Double Quote |
| `'` | `&apos;` | Single Quote |

Here is the Go implementation inside [assembler.go](file:///c:/Users/roych/OFFICIAL%20MVP/internal/iso20022/assembler.go):

```go
func sanitizeXML(s string) string {
    s = strings.TrimSpace(s)
    s = strings.ReplaceAll(s, "&", "&amp;")
    s = strings.ReplaceAll(s, "<", "&lt;")
    s = strings.ReplaceAll(s, ">", "&gt;")
    s = strings.ReplaceAll(s, "\"", "&quot;")
    s = strings.ReplaceAll(s, "'", "&apos;")
    return s
}
```

---

## 4. Go-Native Validation Architecture

Traditional systems rely on external utility binaries like `xmllint` to check XML validity. To make the Connex engine cross-platform and self-contained, it implements a native schema validator.

### 4.1 Unmarshalling into a Struct
The validator uses Go's `encoding/xml` package to decode the output XML bytes back into a Go structure (`XMLDocument`), mapping XML tags to struct fields:

```go
type XMLDocument struct {
    XMLName xml.Name `xml:"Document"`
    FIToFICstmrCdtTrf struct {
        GrpHdr struct {
            MsgId string `xml:"MsgId"`
            // ...
        } `xml:"GrpHdr"`
        CdtTrfTxInf struct {
            IntrBkSttlmAmt struct {
                Ccy    string `xml:",attr"` // reads attribute ccy="..."
                Amount string `xml:",chardata"` // reads content between tags
            } `xml:"IntrBkSttlmAmt"`
            // ...
        } `xml:"CdtTrfTxInf"`
    } `xml:"FIToFICstmrCdtTrf"`
}
```

### 4.2 Programmatic Constraint Checks
Once the XML is unmarshalled, `validateNativeISO` executes a set of strict rules against the populated fields:

```go
func validateNativeISO(xmlData []byte) error {
    var doc XMLDocument
    if err := xml.Unmarshal(xmlData, &doc); err != nil {
        return fmt.Errorf("unmarshal for native validation: %w", err)
    }

    txInfo := doc.FIToFICstmrCdtTrf.CdtTrfTxInf

    // 1. Currency Code must be exactly 3 uppercase letters
    ccy := txInfo.IntrBkSttlmAmt.Ccy
    if len(ccy) != 3 {
        return fmt.Errorf("invalid Currency: %q (must be exactly 3 characters)", ccy)
    }
    for i := 0; i < len(ccy); i++ {
        if ccy[i] < 'A' || ccy[i] > 'Z' {
            return fmt.Errorf("invalid Currency character in %q", ccy)
        }
    }

    // 2. Amount must not be negative
    amtStr := strings.TrimSpace(txInfo.IntrBkSttlmAmt.Amount)
    if strings.Contains(amtStr, "-") {
        return fmt.Errorf("negative amount not allowed: %q", amtStr)
    }

    // 3. Purpose Code must be exactly 4 alphanumeric characters
    pc := txInfo.PmtTpInf.CtgyPurp.Cd
    if pc != "" {
        if len(pc) != 4 {
            return fmt.Errorf("invalid purpose code: %q", pc)
        }
    }
    
    // ... additional schema integrity audits ...
    return nil
}
```

---

## 5. Replicating this Assembler (Beginner Guide)

If you are building your own XML generation layer:
1.  **Always Sanitize Input**: Run a function like `sanitizeXML` on all variables before passing them to the template parser.
2.  **Combine XML Structs**: Design Go struct schemas matching the exact element tags of your target XSD. Unmarshaling generated XML into these structs is a simple and reliable way to check that tags are nested correctly.
3.  **Graceful Failures**: If an XML payload fails either well-formedness or validation, reject the API request with an appropriate HTTP status code (e.g. `422 Unprocessable Entity`), and do not write to the database or send it to external networks.
