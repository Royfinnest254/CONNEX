// internal/iso20022/assembler.go — pacs.008.001.08 XML assembler
//
// Accepts parsed ISO 8583 fields + enrichment results.
// Renders the pacs008.xml.tmpl template and validates the output
// against the bundled XSD using exec("xmllint") — no CGO required.

package iso20022

import (
	_ "embed"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/connextech/connex/internal/enrichment"
	"github.com/connextech/connex/internal/iso8583"
)

//go:embed templates/pacs008.xml.tmpl
var pacs008Tmpl string

var tmpl = template.Must(template.New("pacs008").Parse(pacs008Tmpl))

// TemplateData is the data passed to the pacs008 template.
// TemplateData is the data passed to the pacs008 template.
type TemplateData struct {
	MsgID                string
	BundleID             string
	CreationDateTime     string
	SettlementDate       string
	EndToEndID           string
	TransactionID        string
	Amount               string
	Currency             string
	SettlementMethod     string
	LocalInstrument      string
	ServiceLevel         string
	PurposeCode          string
	ChargeBearer         string
	RegulatoryCode       string
	AcquiringInstitution string
	ForwardingInstitution string
	DebtorAccount        string
	CreditorAccount      string
}

// XMLDocument represents the minimal pacs.008 schema structure needed for native validation.
type XMLDocument struct {
	XMLName xml.Name `xml:"Document"`
	FIToFICstmrCdtTrf struct {
		GrpHdr struct {
			MsgId    string `xml:"MsgId"`
			CreDtTm  string `xml:"CreDtTm"`
			SttlmInf struct {
				SttlmMtd string `xml:"SttlmMtd"`
				LclInstrm struct {
					Prtry string `xml:"Prtry"`
				} `xml:"LclInstrm"`
			} `xml:"SttlmInf"`
		} `xml:"GrpHdr"`
		CdtTrfTxInf struct {
			PmtId struct {
				EndToEndId string `xml:"EndToEndId"`
				TxId       string `xml:"TxId"`
			} `xml:"PmtId"`
			PmtTpInf struct {
				SvcLvl struct {
					Cd string `xml:"Cd"`
				} `xml:"SvcLvl"`
				CtgyPurp struct {
					Cd string `xml:"Cd"`
				} `xml:"CtgyPurp"`
			} `xml:"PmtTpInf"`
			IntrBkSttlmAmt struct {
				Ccy    string `xml:",attr"`
				Amount string `xml:",chardata"`
			} `xml:"IntrBkSttlmAmt"`
			IntrBkSttlmDt string `xml:"IntrBkSttlmDt"`
			ChrgBr        string `xml:"ChrgBr"`
			InstgAgt struct {
				FinInstnId struct {
					Othr struct {
						Id string `xml:"Id"`
					} `xml:"Othr"`
				} `xml:"FinInstnId"`
			} `xml:"InstgAgt"`
			InstdAgt struct {
				FinInstnId struct {
					Othr struct {
						Id string `xml:"Id"`
					} `xml:"Othr"`
				} `xml:"FinInstnId"`
			} `xml:"InstdAgt"`
			Dbtr struct {
				Nm string `xml:"Nm"`
			} `xml:"Dbtr"`
			DbtrAcct struct {
				Id struct {
					Othr struct {
						Id string `xml:"Id"`
					} `xml:"Othr"`
				} `xml:"Id"`
			} `xml:"DbtrAcct"`
			Cdtr struct {
				Nm string `xml:"Nm"`
			} `xml:"Cdtr"`
			CdtrAcct struct {
				Id struct {
					Othr struct {
						Id string `xml:"Id"`
					} `xml:"Othr"`
				} `xml:"Id"`
			} `xml:"CdtrAcct"`
			RgltryRptg struct {
				Authrty struct {
					Nm   string `xml:"Nm"`
					Ctry string `xml:"Ctry"`
				} `xml:"Authrty"`
				Dtls struct {
					Cd string `xml:"Cd"`
				} `xml:"Dtls"`
			} `xml:"RgltryRptg"`
		} `xml:"CdtTrfTxInf"`
	} `xml:"FIToFICstmrCdtTrf"`
}

// Assemble builds a pacs.008.001.08 XML document from the parsed 8583 message
// and the enrichment result. Returns the validated XML bytes.
func Assemble(msg *iso8583.Message, enr *enrichment.Result, bundleID string) ([]byte, error) {
	now := time.Now().UTC()

	// Build template data with strict XML entity sanitization
	td := TemplateData{
		MsgID:            sanitizeXML(bundleID),
		BundleID:         sanitizeXML(bundleID),
		CreationDateTime: now.Format("2006-01-02T15:04:05"),
		SettlementDate:   now.Format("2006-01-02"),
		EndToEndID:       sanitizeXML(msg.Fields[37]),
		TransactionID:    sanitizeXML(msg.Fields[11]),
		Amount:           sanitizeXML(enrichment.FormatAmount(msg.Fields[4])),
		Currency:         sanitizeXML(enr.InstructedAmountCurrency),
		SettlementMethod: sanitizeXML(enr.SettlementMethod),
		LocalInstrument:  sanitizeXML(enr.LocalInstrument),
		ServiceLevel:     sanitizeXML(enr.ServiceLevel),
		PurposeCode:      sanitizeXML(enr.PurposeCode),
		ChargeBearer:     sanitizeXML(enr.ChargeBearer),
		RegulatoryCode:   sanitizeXML(enr.RegulatoryCode),
		AcquiringInstitution:  sanitizeXML(msg.Fields[32]),
		ForwardingInstitution: sanitizeXML(msg.Fields[33]),
		DebtorAccount:         sanitizeXML(msg.Fields[102]),
		CreditorAccount:       sanitizeXML(msg.Fields[103]),
	}

	// Apply defaults for empty fields
	if td.EndToEndID == "" {
		td.EndToEndID = td.BundleID
	}
	if td.TransactionID == "" {
		td.TransactionID = td.BundleID
	}
	if td.AcquiringInstitution == "" {
		td.AcquiringInstitution = "UNKNOWN"
	}
	if td.ForwardingInstitution == "" {
		td.ForwardingInstitution = "UNKNOWN"
	}
	if td.DebtorAccount == "" {
		td.DebtorAccount = sanitizeXML(msg.Fields[2]) // PAN fallback
	}
	if td.DebtorAccount == "" {
		td.DebtorAccount = "UNKNOWN"
	}
	if td.CreditorAccount == "" {
		td.CreditorAccount = "DEFERRED"
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return nil, fmt.Errorf("template execute: %w", err)
	}

	xmlBytes := buf.Bytes()

	// Validate well-formedness at minimum (always available)
	if err := validateWellFormed(xmlBytes); err != nil {
		return nil, fmt.Errorf("XML not well-formed: %w", err)
	}

	// Programmatic native schema checks to prevent silent bypass of structural constraints
	if err := validateNativeISO(xmlBytes); err != nil {
		return nil, fmt.Errorf("native XML validation: %w", err)
	}

	// XSD validation via xmllint (if available on PATH)
	if xsdPath := os.Getenv("CONNEX_XSD_PATH"); xsdPath != "" {
		if err := validateXSD(xmlBytes, xsdPath); err != nil {
			return nil, fmt.Errorf("XSD validation failed: %w", err)
		}
	}

	return xmlBytes, nil
}

// validateWellFormed checks that the XML is syntactically valid using stdlib xml.
func validateWellFormed(data []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		_, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
	}
}

// validateNativeISO validates basic constraints in the output XML.
func validateNativeISO(xmlData []byte) error {
	var doc XMLDocument
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		return fmt.Errorf("unmarshal for native validation: %w", err)
	}

	header := doc.FIToFICstmrCdtTrf.GrpHdr
	txInfo := doc.FIToFICstmrCdtTrf.CdtTrfTxInf

	if header.MsgId == "" {
		return fmt.Errorf("missing GrpHdr.MsgId")
	}

	if _, err := time.Parse("2006-01-02T15:04:05", header.CreDtTm); err != nil {
		return fmt.Errorf("invalid CreDtTm format: %q", header.CreDtTm)
	}

	method := header.SttlmInf.SttlmMtd
	if method != "CLRG" && method != "INDA" {
		return fmt.Errorf("invalid SttlmMtd: %q (expected CLRG or INDA)", method)
	}

	ccy := txInfo.IntrBkSttlmAmt.Ccy
	if len(ccy) != 3 {
		return fmt.Errorf("invalid Currency: %q (must be exactly 3 characters)", ccy)
	}
	for i := 0; i < len(ccy); i++ {
		if ccy[i] < 'A' || ccy[i] > 'Z' {
			return fmt.Errorf("invalid Currency character in %q", ccy)
		}
	}

	amtStr := strings.TrimSpace(txInfo.IntrBkSttlmAmt.Amount)
	if amtStr == "" {
		return fmt.Errorf("missing instructed amount")
	}
	if strings.Contains(amtStr, "-") {
		return fmt.Errorf("negative amount not allowed: %q", amtStr)
	}

	pc := txInfo.PmtTpInf.CtgyPurp.Cd
	if pc != "" {
		if len(pc) != 4 {
			return fmt.Errorf("invalid purpose code: %q (must be exactly 4 alphanumeric characters)", pc)
		}
		for i := 0; i < len(pc); i++ {
			c := pc[i]
			isAlphaNum := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
			if !isAlphaNum {
				return fmt.Errorf("invalid character %q in purpose code", c)
			}
		}
	}

	cb := txInfo.ChrgBr
	if cb != "SHAR" && cb != "DEBT" && cb != "CRED" && cb != "SLEV" {
		return fmt.Errorf("invalid ChrgBr: %q (expected SHAR, DEBT, CRED, or SLEV)", cb)
	}

	sl := txInfo.PmtTpInf.SvcLvl.Cd
	if sl != "URGP" && sl != "NURG" {
		return fmt.Errorf("invalid SvcLvl: %q (expected URGP or NURG)", sl)
	}

	if txInfo.DbtrAcct.Id.Othr.Id == "" {
		return fmt.Errorf("missing DebtorAccount")
	}
	if txInfo.CdtrAcct.Id.Othr.Id == "" {
		return fmt.Errorf("missing CreditorAccount")
	}

	return nil
}

// validateXSD runs xmllint --schema for full XSD conformance.
// Requires xmllint to be installed.
func validateXSD(xmlData []byte, xsdPath string) error {
	cmd := exec.Command("xmllint", "--schema", xsdPath, "--noout", "-")
	cmd.Stdin = bytes.NewReader(xmlData)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xmllint: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// sanitizeXML escapes a string for safe inclusion in XML.
func sanitizeXML(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
