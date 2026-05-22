package iso8583

import (
	"encoding/hex"
	"testing"
)

func TestParseASCII(t *testing.T) {
	// Let's craft a simple valid ASCII ISO 8583 message.
	// MTI = "0200" (4 bytes)
	// Bitmap = "3000000000000000" (8 bytes). 
	// Hex: '3' = 00110000 in binary. So fields 3 and 4 are present.
	// F3 = "000000" (fixed 6 ASCII digits)
	// F4 = "000000001000" (fixed 12 ASCII digits)
	// Total message = "0200" + "\x30\x00\x00\x00\x00\x00\x00\x00" + "000000" + "000000001000"
	bitmapBytes, _ := hex.DecodeString("3000000000000000")
	msgBytes := append([]byte("0200"), bitmapBytes...)
	msgBytes = append(msgBytes, []byte("000000")...)
	msgBytes = append(msgBytes, []byte("000000001000")...)

	msg, err := Parse(msgBytes)
	if err != nil {
		t.Fatalf("Failed to parse ASCII message: %v", err)
	}

	if msg.MTI != "0200" {
		t.Errorf("Expected MTI '0200', got %q", msg.MTI)
	}
	if msg.Fields[3] != "000000" {
		t.Errorf("Expected F3 '000000', got %q", msg.Fields[3])
	}
	if msg.Fields[4] != "000000001000" {
		t.Errorf("Expected F4 '000000001000', got %q", msg.Fields[4])
	}
	if msg.AmountKES() != 10.0 {
		t.Errorf("Expected AmountKES 10.0, got %f", msg.AmountKES())
	}
}

func TestParseBCD(t *testing.T) {
	// Let's test a BCD message based on our corpus findings.
	// MTI = 0200 (stored as 02 00, 2 bytes BCD)
	// Bitmap = 72 38 44 81 08 C0 80 00 (8 bytes)
	// Present fields: [2, 3, 4, 7, 11, 12, 13, 18, 22, 25, 32, 37, 41, 42, 49]
	// Raw data for TX-2026000:
	// 02 00 00 00 00 08 55 10 00 05 16 15 12 57 00 00 00 50 45
	// This includes:
	// - F2 = "00" (starts with 02, value 00)
	// - F3 = "000000"
	// - F4 = "000855100005"
	// - F7 = "1615125700"
	// - F11 = "00"
	// - F18 = "5045"
	rawHex := "02007238448108c0800002000000000855100005161512570000005045"
	rawBytes, _ := hex.DecodeString(rawHex)

	msg, err := Parse(rawBytes)
	if err != nil {
		t.Fatalf("Failed to parse BCD message (F2 present): %v", err)
	}

	if msg.MTI != "0200" {
		t.Errorf("Expected MTI '0200', got %q", msg.MTI)
	}
	if msg.Fields[2] != "00" {
		t.Errorf("Expected F2 '00', got %q", msg.Fields[2])
	}
	if msg.Fields[3] != "000000" {
		t.Errorf("Expected F3 '000000', got %q", msg.Fields[3])
	}
	if msg.Fields[4] != "085510000516" {
		t.Errorf("Expected F4 '085510000516', got %q", msg.Fields[4])
	}
	if msg.Fields[7] != "1512570000" {
		t.Errorf("Expected F7 '1512570000', got %q", msg.Fields[7])
	}
	if msg.Fields[11] != "00" {
		t.Errorf("Expected F11 '00', got %q", msg.Fields[11])
	}
	if msg.Fields[18] != "5045" {
		t.Errorf("Expected F18 '5045', got %q", msg.Fields[18])
	}

	// Test a BCD message where F2 is omitted (like TX-2026001)
	// 40 00 00 00 00 23 32 43 00 05 16 15 12 57 00 00 01 50 45
	// - F2 is omitted
	// - F3 = "400000"
	// - F4 = "000023324300"
	// - F7 = "0516151257"
	// - F11 = "000001"
	// - F18 = "5045"
	rawHexOmitted := "02007238448108c0800040000000002332430005161512570000015045"
	rawBytesOmitted, _ := hex.DecodeString(rawHexOmitted)

	msgOmitted, err := Parse(rawBytesOmitted)
	if err != nil {
		t.Fatalf("Failed to parse BCD message (F2 omitted): %v", err)
	}

	if msgOmitted.Fields[2] != "" {
		t.Errorf("Expected F2 to be empty, got %q", msgOmitted.Fields[2])
	}
	if msgOmitted.Fields[3] != "400000" {
		t.Errorf("Expected F3 '400000', got %q", msgOmitted.Fields[3])
	}
	if msgOmitted.Fields[4] != "000023324300" {
		t.Errorf("Expected F4 '000023324300', got %q", msgOmitted.Fields[4])
	}
	if msgOmitted.Fields[7] != "0516151257" {
		t.Errorf("Expected F7 '0516151257', got %q", msgOmitted.Fields[7])
	}
	if msgOmitted.Fields[11] != "000001" {
		t.Errorf("Expected F11 '000001', got %q", msgOmitted.Fields[11])
	}
	if msgOmitted.Fields[18] != "5045" {
		t.Errorf("Expected F18 '5045', got %q", msgOmitted.Fields[18])
	}
}
