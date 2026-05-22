# ISO 8583 Financial Message Parsing

This document explains the mechanics of the ISO 8583-1987 financial message parser implemented in the Connex system. We will explore how message layouts are defined, how binary data is decoded using BCD (Binary Coded Decimal), and how to read the Go code inside `internal/iso8583/parser.go`.

---

## 1. The Structure of an ISO 8583 Message

ISO 8583 is an international standard for financial transaction card originated messages. A typical message consists of three parts:

```
+--------------------+-------------------------+---------------------------------+
| MTI (4 characters) | Bitmap (8 or 16 bytes)  | Data Elements (Variable length) |
+--------------------+-------------------------+---------------------------------+
```

### 1.1 Message Type Identifier (MTI)
A 4-digit numeric code classifying the message:
- **`0200`**: Financial Transaction Request (e.g., a cardholder purchases something).
- **`0210`**: Financial Transaction Response (e.g., approval or rejection from the bank).

### 1.2 The Bitmap
A series of bits indicating which subsequent data fields are present in the packet.
- **Primary Bitmap**: 8 bytes (64 bits). Bit 1 represents the presence of a Secondary Bitmap. Bits 2 to 64 represent the presence of fields 2 to 64.
- **Secondary Bitmap**: Optional 8 bytes (64 bits). Present only if Bit 1 of the Primary Bitmap is set. It maps fields 65 to 128.

#### Bitwise Mapping Example
If only Field 3 (Processing Code) and Field 4 (Transaction Amount) are present:
- Bitmap bits: `00110000 00000000 ...` (Field 3 and 4 bits are `1`).
- Hex representation: `30 00 00 00 00 00 00 00`.

### 1.3 Data Elements (Fields)
Fields are defined by:
- **`Fixed`**: A constant length (e.g., Processing Code is always 6 characters).
- **`LLVAR`**: Variable length up to 99, prefixed by a 2-digit length indicator (e.g., Account Number, up to 19 digits, prefixed by length).
- **`LLLVAR`**: Variable length up to 999, prefixed by a 3-digit length indicator.

---

## 2. BCD (Binary Coded Decimal) vs ASCII

To save network bandwidth, financial systems often encode decimal digits using **BCD** instead of raw ASCII text characters:

- **ASCII**: Each character takes a full byte. The string `"40"` takes 2 bytes: `0x34` (`'4'`) and `0x30` (`'0'`).
- **BCD (Packed BCD)**: Each decimal digit is packed into a single nibble (half-byte / 4 bits). The value `40` is packed into a single byte: `0x40`.
  - Digit `4` = `0100` (binary)
  - Digit `0` = `0000` (binary)
  - Packed Byte = `01000000` = `0x40`.

Connex supports both formats. It automatically detects BCD messages by checking if the first byte of the MTI lies outside the printable ASCII range (`0x20` to `0x7E`).

---

## 3. Go Implementation Walkthrough

Let's break down the Go code in [parser.go](file:///c:/Users/roych/OFFICIAL%20MVP/internal/iso8583/parser.go) to see how it decodes these fields.

### 3.1 Field Definitions
Go uses structs to represent schemas. The parser defines a `fieldDef` struct and maps each field:

```go
type fieldKind int

const (
    kindFixed  fieldKind = iota // 0
    kindLLVAR                   // 1
    kindLLLVAR                  // 2
)

type fieldDef struct {
    Number int
    Name   string
    Kind   fieldKind
    Length int // Exact length for Fixed, max length for LLVAR/LLLVAR
}
```

### 3.2 Reading the Bitmap using Bit Manipulation
To find which fields are present, the parser checks individual bits of the bitmap:

```go
func bitmapBits(raw []byte, offset int) []int {
    var set []int
    for byteIdx, b := range raw {
        for bit := 7; bit >= 0; bit-- {
            // Shift the byte to isolate the bit, then check if it equals 1
            if (b >> uint(bit)) & 1 == 1 {
                // Calculate the actual field number (1-indexed)
                fieldNum := offset + byteIdx*8 + (7 - bit) + 1
                set = append(set, fieldNum)
            }
        }
    }
    return set
}
```

#### How the Bit Shift `(b >> uint(bit)) & 1` works:
Suppose `b` is `0x30` (`00110000` in binary).
- When `bit = 5`: `b >> 5` shifts the bits right by 5, resulting in `00000001`. Anding this with `1` (`& 1`) results in `1`. The bit is set!
- Field number calculation: `0 + 0*8 + (7 - 5) + 1 = 3`. Field 3 is present!

### 3.3 The Core Parsing Loop
The `Parse` function processes the payload byte stream sequentially:

```go
for _, fieldNum := range presentFields {
    if fieldNum == 1 {
        continue // Skip secondary bitmap indicator
    }

    def, known := de[fieldNum]
    if !known {
        // VITAL SECURITY CHECK: abort if we don't know the field's length type.
        // If we skip an unknown field, we won't know how many bytes to skip,
        // misaligning all subsequent fields.
        return nil, fmt.Errorf("unknown field F%d encountered: cannot parse alignment securely", fieldNum)
    }
    
    // Process field depending on encoding (ASCII or BCD)
    ...
}
```

### 3.4 BCD Variable Length Decoding
Here is how the parser decodes a variable-length `LLVAR` field (e.g., Account Number) in BCD mode:

```go
case kindLLVAR:
    // Read the 1-byte BCD length prefix
    lengthByte := raw[pos]
    pos++
    
    // Unpack BCD length byte: e.g., 0x19 -> 1*10 + 9 = 19 digits
    dataLen := int(lengthByte >> 4)*10 + int(lengthByte & 0x0f)
    if dataLen > def.Length {
        return nil, fmt.Errorf("F%d (%s): BCD LLVAR length %d exceeds max %d", fieldNum, def.Name, dataLen, def.Length)
    }
    
    // Calculate BCD byte length: since 1 byte holds 2 digits, we round up: (digits + 1) / 2
    bcdLen := (dataLen + 1) / 2
    if pos + bcdLen > limit {
        bcdLen = limit - pos
    }
    
    // Convert BCD bytes back to a readable string (hex notation represents BCD perfectly)
    msg.Fields[fieldNum] = decodeBCD(raw[pos : pos+bcdLen], dataLen)
    pos += bcdLen
```

---

## 4. Replicating this Parser (Beginner Guide)

If you want to replicate or write a parser for a custom version of ISO 8583:
1.  **Define Your Field Table**: Make sure your map (`de`) covers all fields your system accepts.
2.  **Avoid Skipping**: If your parser encounters a field it doesn't recognize, do **not** attempt to guess the length. Safely abort parsing with an error to prevent corrupted transactions.
3.  **Validate Indexes**: Always verify that the slice index `pos + expectedLength` is less than or equal to `len(raw)` before slicing. Otherwise, Go will panic with an out-of-bounds error.
