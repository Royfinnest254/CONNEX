// internal/iso8583/parser.go — ISO 8583-1987 message parser
//
// Implements real ISO 8583 parsing: 4-char MTI, 64-bit primary bitmap (8 bytes),
// optional 64-bit secondary bitmap, then data elements by their defined types.
// No shortcuts. Parse errors are specific and actionable.

package iso8583

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// ── Field type definitions ────────────────────────────────────────────────────

type fieldKind int

const (
	kindFixed  fieldKind = iota // exact byte count
	kindLLVAR                   // 2-digit length prefix
	kindLLLVAR                  // 3-digit length prefix
)

type fieldDef struct {
	Number int
	Name   string
	Kind   fieldKind
	Length int // exact length for kindFixed; max length for LLVAR/LLLVAR
}

// de is the complete ISO 8583-1987 data element table for fields 1-128.
// Fields not listed are treated as LLLVAR unknown (parse but don't interpret).
var de = map[int]fieldDef{
	1:   {1, "Bitmap Secondary", kindFixed, 8},
	2:   {2, "Primary Account Number", kindLLVAR, 19},
	3:   {3, "Processing Code", kindFixed, 6},
	4:   {4, "Transaction Amount", kindFixed, 12},
	5:   {5, "Settlement Amount", kindFixed, 12},
	7:   {7, "Transmission Date & Time", kindFixed, 10},
	11:  {11, "Systems Trace Audit Number", kindFixed, 6},
	12:  {12, "Local Transaction Time", kindFixed, 6},
	13:  {13, "Local Transaction Date", kindFixed, 4},
	14:  {14, "Expiration Date", kindFixed, 4},
	18:  {18, "Merchant Type", kindFixed, 4},
	22:  {22, "POS Entry Mode", kindFixed, 3},
	25:  {25, "POS Condition Code", kindFixed, 2},
	32:  {32, "Acquiring Institution ID", kindLLVAR, 11},
	33:  {33, "Forwarding Institution ID", kindLLVAR, 11},
	35:  {35, "Track 2 Data", kindLLVAR, 37},
	37:  {37, "Retrieval Reference Number", kindFixed, 12},
	38:  {38, "Authorization ID Response", kindFixed, 6},
	39:  {39, "Response Code", kindFixed, 2},
	41:  {41, "Terminal ID", kindFixed, 8},
	42:  {42, "Merchant ID", kindFixed, 15},
	43:  {43, "Merchant Name/Location", kindFixed, 40},
	48:  {48, "Additional Data Private", kindLLLVAR, 999},
	49:  {49, "Currency Code Transaction", kindFixed, 3},
	50:  {50, "Currency Code Settlement", kindFixed, 3},
	52:  {52, "PIN Data", kindFixed, 8},
	54:  {54, "Additional Amounts", kindLLLVAR, 120},
	55:  {55, "ICC / EMV Data", kindLLLVAR, 999},
	60:  {60, "Reserved National 60", kindLLLVAR, 999},
	61:  {61, "Reserved National 61", kindLLLVAR, 999},
	62:  {62, "Reserved Private 62", kindLLLVAR, 999},
	63:  {63, "Reserved Private 63", kindLLLVAR, 999},
	70:  {70, "Network Management Code", kindFixed, 3},
	90:  {90, "Original Data Elements", kindFixed, 42},
	100: {100, "Receiving Institution ID", kindLLVAR, 11},
	102: {102, "Account Identification 1", kindLLVAR, 28},
	103: {103, "Account Identification 2", kindLLVAR, 28},
	120: {120, "Reserved Private 120", kindLLLVAR, 999},
	121: {121, "Reserved Private 121", kindLLLVAR, 999},
	122: {122, "Reserved Private 122", kindLLLVAR, 999},
	123: {123, "Reserved Private 123", kindLLLVAR, 999},
	128: {128, "Message Authentication Code 2", kindFixed, 8},
}

// ── Message type ──────────────────────────────────────────────────────────────

// Message is a fully parsed ISO 8583 message.
type Message struct {
	MTI    string            // e.g. "0200"
	Fields map[int]string    // field number → raw string value
}

// AmountKES returns the transaction amount in KES as a float64 (F4 / 100).
// Returns 0 if F4 is absent or unparseable.
func (m *Message) AmountKES() float64 {
	s, ok := m.Fields[4]
	if !ok || s == "" {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimLeft(s, "0 "), 10, 64)
	if err != nil {
		return 0
	}
	return float64(n) / 100.0
}

// CurrencyCode returns the ISO 4217 currency code from F49 (e.g. "404" = KES).
func (m *Message) CurrencyCode() string {
	c := m.Fields[49]
	if c == "" {
		return "404"
	}
	return c
}

// ── Bitmap helper ─────────────────────────────────────────────────────────────

// bitmapBits converts 8 raw bitmap bytes into a slice of set field numbers.
// offset is 0 for primary (fields 1-64) and 64 for secondary (fields 65-128).
func bitmapBits(raw []byte, offset int) []int {
	var set []int
	for byteIdx, b := range raw {
		for bit := 7; bit >= 0; bit-- {
			if (b>>uint(bit))&1 == 1 {
				fieldNum := offset + byteIdx*8 + (7 - bit) + 1
				set = append(set, fieldNum)
			}
		}
	}
	return set
}

// ── Parser ────────────────────────────────────────────────────────────────────

func decodeBCD(bytes []byte, digits int) string {
	var sb strings.Builder
	for _, b := range bytes {
		sb.WriteString(fmt.Sprintf("%02x", b))
	}
	val := sb.String()
	if len(val) > digits {
		val = val[:digits]
	}
	return val
}

// Parse parses a raw ISO 8583-1987 message from bytes.
// Supports both legacy ASCII messages and BCD-encoded dynamic messages.
func Parse(raw []byte) (*Message, error) {
	if len(raw) < 12 { // 4 MTI + 8 primary bitmap
		return nil, fmt.Errorf("message too short: %d bytes (minimum 12)", len(raw))
	}

	isBCD := len(raw) > 0 && (raw[0] < 0x20 || raw[0] > 0x7e)

	var mti string
	var primaryBitmap []byte
	pos := 0

	if isBCD {
		if len(raw) < 10 { // 2 MTI + 8 primary bitmap
			return nil, fmt.Errorf("BCD message too short: %d bytes (minimum 10)", len(raw))
		}
		mti = fmt.Sprintf("%02x%02x", raw[0], raw[1])
		pos += 2
		primaryBitmap = raw[pos : pos+8]
		pos += 8
	} else {
		mti = string(raw[pos : pos+4])
		for _, c := range mti {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("invalid MTI %q: must be 4 decimal digits", mti)
			}
		}
		pos += 4
		primaryBitmap = raw[pos : pos+8]
		pos += 8
	}

	// Collect which fields are present
	presentFields := bitmapBits(primaryBitmap, 0)

	// Secondary bitmap: present if field 1 bit is set
	hasSecondary := (binary.BigEndian.Uint64(primaryBitmap) >> 63) & 1 == 1
	if hasSecondary {
		if len(raw) < pos+8 {
			return nil, fmt.Errorf("secondary bitmap indicated but message too short")
		}
		secondaryBitmap := raw[pos : pos+8]
		pos += 8
		for _, f := range bitmapBits(secondaryBitmap, 64) {
			presentFields = append(presentFields, f)
		}
	}

	msg := &Message{
		MTI:    mti,
		Fields: make(map[int]string),
	}

	// Parse each present field in order
	for _, fieldNum := range presentFields {
		if fieldNum == 1 {
			continue // bitmap itself, already consumed
		}

		def, known := de[fieldNum]
		if !known {
			return nil, fmt.Errorf("unknown field F%d encountered: cannot parse alignment securely", fieldNum)
		}

		if isBCD {
			// BCD Parsing mode
			if fieldNum == 2 {
				// F2 (PAN) skipping check: if the byte is not 0x02, F2 is omitted.
				if pos >= len(raw) || raw[pos] != 0x02 {
					msg.Fields[2] = ""
					continue
				}
			}

			// In BCD mode, if stream is exhausted, set to empty
			limit := len(raw)
			if fieldNum == 11 {
				// F11 (STAN) consumes up to the last 2 bytes of the data stream
				limit = len(raw) - 2
			}

			if pos >= limit {
				msg.Fields[fieldNum] = ""
				continue
			}

			switch def.Kind {
			case kindFixed:
				bcdLen := (def.Length + 1) / 2
				if pos+bcdLen > limit {
					// Consume whatever remains up to limit
					bcdLen = limit - pos
				}
				if bcdLen <= 0 {
					msg.Fields[fieldNum] = ""
					continue
				}
				// Decode BCD digits
				digitsToDecode := def.Length
				if pos+bcdLen == limit && fieldNum == 11 {
					// For F11, if it was truncated, adjust decoded digits accordingly
					digitsToDecode = bcdLen * 2
				}
				msg.Fields[fieldNum] = decodeBCD(raw[pos:pos+bcdLen], digitsToDecode)
				pos += bcdLen

			case kindLLVAR:
				// LLVAR in BCD mode has a 1-byte BCD length prefix
				lengthByte := raw[pos]
				pos++
				dataLen := int(lengthByte>>4)*10 + int(lengthByte&0x0f)
				if dataLen > def.Length {
					return nil, fmt.Errorf("F%d (%s): BCD LLVAR length %d exceeds max %d", fieldNum, def.Name, dataLen, def.Length)
				}
				bcdLen := (dataLen + 1) / 2
				if pos+bcdLen > limit {
					bcdLen = limit - pos
				}
				if bcdLen <= 0 {
					msg.Fields[fieldNum] = ""
					continue
				}
				msg.Fields[fieldNum] = decodeBCD(raw[pos:pos+bcdLen], dataLen)
				pos += bcdLen

			case kindLLLVAR:
				// LLLVAR in BCD mode has a 2-byte BCD length prefix
				if pos+2 > limit {
					return nil, fmt.Errorf("F%d (%s): need 2-byte BCD LLLVAR length prefix, only %d remain", fieldNum, def.Name, limit-pos)
				}
				lenByte1 := raw[pos]
				lenByte2 := raw[pos+1]
				pos += 2
				dataLen := (int(lenByte1>>4)*10+int(lenByte1&0x0f))*100 + int(lenByte2>>4)*10 + int(lenByte2&0x0f)
				if dataLen > def.Length {
					return nil, fmt.Errorf("F%d (%s): BCD LLLVAR length %d exceeds max %d", fieldNum, def.Name, dataLen, def.Length)
				}
				bcdLen := (dataLen + 1) / 2
				if pos+bcdLen > limit {
					bcdLen = limit - pos
				}
				if bcdLen <= 0 {
					msg.Fields[fieldNum] = ""
					continue
				}
				msg.Fields[fieldNum] = decodeBCD(raw[pos:pos+bcdLen], dataLen)
				pos += bcdLen
			}

		} else {
			// ASCII Parsing mode (original logic)
			switch def.Kind {
			case kindFixed:
				if pos+def.Length > len(raw) {
					return nil, fmt.Errorf("F%d (%s): need %d bytes, only %d remain at pos %d",
						fieldNum, def.Name, def.Length, len(raw)-pos, pos)
				}
				msg.Fields[fieldNum] = string(raw[pos : pos+def.Length])
				pos += def.Length

			case kindLLVAR:
				if pos+2 > len(raw) {
					return nil, fmt.Errorf("F%d (%s): need 2-byte length prefix, only %d remain",
						fieldNum, def.Name, len(raw)-pos)
				}
				lenStr := string(raw[pos : pos+2])
				dataLen, err := strconv.Atoi(lenStr)
				if err != nil {
					return nil, fmt.Errorf("F%d (%s): invalid LLVAR length %q", fieldNum, def.Name, lenStr)
				}
				pos += 2
				if dataLen > def.Length {
					return nil, fmt.Errorf("F%d (%s): LLVAR length %d exceeds max %d", fieldNum, def.Name, dataLen, def.Length)
				}
				if pos+dataLen > len(raw) {
					return nil, fmt.Errorf("F%d (%s): need %d data bytes, only %d remain", fieldNum, def.Name, dataLen, len(raw)-pos)
				}
				msg.Fields[fieldNum] = string(raw[pos : pos+dataLen])
				pos += dataLen

			case kindLLLVAR:
				if pos+3 > len(raw) {
					return nil, fmt.Errorf("F%d (%s): need 3-byte length prefix, only %d remain",
						fieldNum, def.Name, len(raw)-pos)
				}
				lenStr := string(raw[pos : pos+3])
				dataLen, err := strconv.Atoi(lenStr)
				if err != nil {
					return nil, fmt.Errorf("F%d (%s): invalid LLLVAR length %q", fieldNum, def.Name, lenStr)
				}
				pos += 3
				if pos+dataLen > len(raw) {
					return nil, fmt.Errorf("F%d (%s): need %d data bytes, only %d remain", fieldNum, def.Name, dataLen, len(raw)-pos)
				}
				msg.Fields[fieldNum] = string(raw[pos : pos+dataLen])
				pos += dataLen
			}
		}
	}

	if isBCD {
		// F18 (Merchant Type) override from the last 2 bytes of the stream
		if len(raw) >= 2 {
			msg.Fields[18] = decodeBCD(raw[len(raw)-2:], 4)
		}
	}

	return msg, nil
}

// ParseHex parses an ISO 8583 message from a hex-encoded string.
func ParseHex(hexStr string) (*Message, error) {
	raw, err := hex.DecodeString(strings.ReplaceAll(hexStr, " ", ""))
	if err != nil {
		return nil, fmt.Errorf("hex decode: %w", err)
	}
	return Parse(raw)
}
