// internal/enrichment/engine.go — Connex deterministic rules engine
//
// Evaluates rules.yaml in order. Each matched rule annotates the enrichment log
// with: field name, value, source (rule ID), confidence (1.0 for rules), and
// the rule description. The log is the complete provenance record for the event.

package enrichment

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed rules.yaml
var rulesYAML []byte

// ── Data structures ───────────────────────────────────────────────────────────

// FieldEntry is one entry in the enrichment log.
type FieldEntry struct {
	Value          string  `json:"value"`
	Source         string  `json:"source"`          // rule ID or "DEFERRED_AI_CLASSIFIER_V0_2"
	Description    string  `json:"description"`
	Confidence     float64 `json:"confidence"`
	LocalInstrument string `json:"local_instrument,omitempty"`
}

// Log is the complete enrichment record for one coordination event.
type Log map[string]FieldEntry

// Input is the parsed ISO 8583 fields passed to the engine.
type Input struct {
	AmountKES              float64
	CurrencyISO4217        string // "404"=KES, "840"=USD, etc.
	ProcessingCode         string // 6-digit F3
	MerchantType           string // 4-digit F18
	AcquiringInstitution   string // F32
	ForwardingInstitution  string // F33
	CrossInstitutional     bool   // ForwardingInstitution != AcquiringInstitution
}

// Result is the enriched payment fields ready for pacs.008 assembly.
type Result struct {
	RegulatoryCode          string
	PurposeCode             string
	SettlementMethod        string
	LocalInstrument         string
	InstructedAmountCurrency string
	ChargeBearer            string
	ServiceLevel            string
	Log                     Log
}

// ── Rule loading ──────────────────────────────────────────────────────────────

type ruleCondition struct {
	AmountMinKES              float64  `yaml:"amount_min_kes"`
	AmountMaxKES              float64  `yaml:"amount_max_kes"`
	CurrencyISO4217           string   `yaml:"currency_iso4217"`
	CurrencyISO4217Not        string   `yaml:"currency_iso4217_not"`
	ProcessingCode            string   `yaml:"processing_code"`
	ProcessingCodePrefix      string   `yaml:"processing_code_prefix"`
	MerchantType              string   `yaml:"merchant_type"`
	MerchantTypeIn            []string `yaml:"merchant_type_in"`
	CrossInstitutional        bool     `yaml:"cross_institutional"`
	AcquiringInstitutionPrefix string  `yaml:"acquiring_institution_prefix"`
	PurposeCodeUnset          bool     `yaml:"purpose_code_unset"`
	Always                    bool     `yaml:"always"`
}

type ruleOutcome struct {
	Field           string  `yaml:"field"`
	Value           string  `yaml:"value"`
	Source          string  `yaml:"source"`
	LocalInstrument string  `yaml:"local_instrument"`
	Confidence      float64 `yaml:"confidence"` // 0.0 if absent → engine uses rule-level default
}

type rule struct {
	ID          string        `yaml:"id"`
	Description string        `yaml:"description"`
	Condition   ruleCondition `yaml:"condition"`
	Outcome     ruleOutcome   `yaml:"outcome"`
	Confidence  float64       `yaml:"confidence"`
}

type rulesFile struct {
	Version string `yaml:"version"`
	Rules   []rule `yaml:"rules"`
}

var loadedRules []rule

func init() {
	var rf rulesFile
	if err := yaml.Unmarshal(rulesYAML, &rf); err != nil {
		panic(fmt.Sprintf("enrichment: failed to load rules.yaml: %v", err))
	}
	loadedRules = rf.Rules
}

// ── Condition evaluation ──────────────────────────────────────────────────────

func (c *ruleCondition) matches(in *Input, partial *Result) bool {
	if c.Always {
		return true
	}
	if c.AmountMinKES > 0 && in.AmountKES < c.AmountMinKES {
		return false
	}
	if c.AmountMaxKES > 0 && in.AmountKES > c.AmountMaxKES {
		return false
	}
	if c.CurrencyISO4217 != "" && in.CurrencyISO4217 != c.CurrencyISO4217 {
		return false
	}
	if c.CurrencyISO4217Not != "" && in.CurrencyISO4217 == c.CurrencyISO4217Not {
		return false
	}
	if c.ProcessingCode != "" && in.ProcessingCode != c.ProcessingCode {
		return false
	}
	if c.ProcessingCodePrefix != "" && !strings.HasPrefix(in.ProcessingCode, c.ProcessingCodePrefix) {
		return false
	}
	if c.MerchantType != "" && in.MerchantType != c.MerchantType {
		return false
	}
	if len(c.MerchantTypeIn) > 0 {
		found := false
		for _, mt := range c.MerchantTypeIn {
			if in.MerchantType == mt {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if c.CrossInstitutional && !in.CrossInstitutional {
		return false
	}
	if c.AcquiringInstitutionPrefix != "" && !strings.HasPrefix(in.AcquiringInstitution, c.AcquiringInstitutionPrefix) {
		return false
	}
	if c.PurposeCodeUnset && partial.PurposeCode != "" {
		return false // purpose already set by a prior rule
	}
	return true
}

// ── Engine ────────────────────────────────────────────────────────────────────

// Enrich runs the rules engine against the input and returns enriched fields
// plus a fully annotated enrichment log.
func Enrich(in *Input) (*Result, error) {
	result := &Result{Log: make(Log)}

	for _, r := range loadedRules {
		if !r.Condition.matches(in, result) {
			continue
		}

		conf := r.Confidence
		if conf == 0 && r.Outcome.Confidence != 0 {
			conf = r.Outcome.Confidence
		}

		src := r.ID
		if r.Outcome.Source != "" {
			src = r.Outcome.Source
		}

		entry := FieldEntry{
			Value:           r.Outcome.Value,
			Source:          src,
			Description:     r.Description,
			Confidence:      conf,
			LocalInstrument: r.Outcome.LocalInstrument,
		}

		switch r.Outcome.Field {
		case "regulatory_reporting_code":
			if result.RegulatoryCode == "" {
				result.RegulatoryCode = r.Outcome.Value
				result.Log["regulatory_reporting_code"] = entry
			}
		case "purpose_code":
			if result.PurposeCode == "" {
				result.PurposeCode = r.Outcome.Value
				result.Log["purpose_code"] = entry
			}
		case "settlement_method":
			if result.SettlementMethod == "" {
				result.SettlementMethod = r.Outcome.Value
				result.LocalInstrument = r.Outcome.LocalInstrument
				result.Log["settlement_method"] = entry
			}
		case "instructed_amount_currency":
			if result.InstructedAmountCurrency == "" {
				result.InstructedAmountCurrency = r.Outcome.Value
				result.Log["instructed_amount_currency"] = entry
			}
		case "charge_bearer":
			if result.ChargeBearer == "" {
				result.ChargeBearer = r.Outcome.Value
				result.Log["charge_bearer"] = entry
			}
		case "service_level":
			if result.ServiceLevel == "" {
				result.ServiceLevel = r.Outcome.Value
				result.Log["service_level"] = entry
			}
		}
	}

	// Apply defaults for any unset fields
	if result.RegulatoryCode == "" {
		result.RegulatoryCode = "100"
		result.Log["regulatory_reporting_code"] = FieldEntry{Value: "100", Source: "DEFAULT", Description: "No matching CBK rule", Confidence: 1.0}
	}
	if result.InstructedAmountCurrency == "" {
		result.InstructedAmountCurrency = isoToCurrencyCode(in.CurrencyISO4217)
		result.Log["instructed_amount_currency"] = FieldEntry{Value: result.InstructedAmountCurrency, Source: "DEFAULT_CURRENCY_MAP", Confidence: 1.0}
	}
	if result.ChargeBearer == "" {
		result.ChargeBearer = "SHAR"
		result.Log["charge_bearer"] = FieldEntry{Value: "SHAR", Source: "DEFAULT", Confidence: 1.0}
	}
	if result.ServiceLevel == "" {
		result.ServiceLevel = "NURG"
		result.Log["service_level"] = FieldEntry{Value: "NURG", Source: "DEFAULT", Confidence: 1.0}
	}

	return result, nil
}

// LogJSON returns the enrichment log serialised as compact JSON.
func (l Log) JSON() (string, error) {
	b, err := json.Marshal(l)
	return string(b), err
}

// isoToCurrencyCode maps ISO 4217 numeric codes to alpha codes.
func isoToCurrencyCode(numeric string) string {
	m := map[string]string{
		"404": "KES", "840": "USD", "978": "EUR",
		"826": "GBP", "756": "CHF", "392": "JPY",
		"036": "AUD", "124": "CAD", "710": "ZAR",
		"800": "UGX", "834": "TZS", "646": "RWF",
	}
	if s, ok := m[numeric]; ok {
		return s
	}
	return numeric
}

// FormatAmount converts a 12-digit right-justified ISO 8583 F4 string to decimal.
func FormatAmount(raw string) string {
	raw = strings.TrimLeft(raw, " 0")
	if raw == "" {
		return "0.00"
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return "0.00"
	}
	whole := n / 100
	cents := n % 100
	return fmt.Sprintf("%d.%02d", whole, cents)
}
