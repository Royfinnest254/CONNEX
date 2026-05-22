# Connex Corpus v1.0

This directory contains the synthetic coordination corpus used for verifying the technical claim of the Connex MVP.

## 1. Composition
The corpus consists of **500 synthetic ISO 8583 messages** in `transactions.jsonl`. Each entry includes:
- `iso8583_hex`: The raw message as it would appear on a legacy bank wire.
- `labels`: The ground-truth enrichment values (regulatory codes, purpose codes) derived from manual analysis.

### Category Distribution
- **Salary (35%):** Payroll payments, typically credit-to-account (F3=40), amounts 10k-200k KES.
- **Supplier (25%):** B2B payments, typically larger values (>100k KES), cross-institutional.
- **Tax (15%):** Government payments (KRA), identified by Merchant Type (F18=9311).
- **Loan (12%):** Repayments, identified by specific processing code patterns.
- **Remittance (8%):** Cross-border flows, non-KES currencies.
- **Other (5%):** General retail and utility payments.

---

## 2. Methodology
**Honest Disclaimer:** This is a structurally realistic **synthetic corpus**. It does not contain real Kenyan bank data or M-Pesa traffic.

The messages were constructed by:
1.  Analyzing the **CBK Regulatory Codebook** for valid reporting categories.
2.  Reviewing **M-Pesa Daraja API** specifications for mobile money transaction structures.
3.  Consulting **ISO 20022 Example Messages** (iso20022.org) for target mapping.
4.  Mapping these patterns back to the **ISO 8583-1987** standard for field lengths and types.

---

## 3. Usage for Verification
The benchmark harness (`cmd/bench`) uses this corpus to verify that:
1.  The **Parser** handles all field types (Fixed, LLVAR, LLLVAR).
2.  The **Enrichment Engine** correctly classifies transactions according to the ground-truth labels.
3.  The **XSD Validator** accepts all generated XML outputs.

## 4. Manifest
- **Version:** 1.0
- **Count:** 500
- **Format:** JSONL
- **SHA-256:** `(calculated at build time)`
