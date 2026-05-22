// cmd/gateway/main.go — Connex Coordination Gateway
//
// Single binary. No web framework. Accepts POST /v1/coordinate with a
// base64-encoded ISO 8583 message, runs the full pipeline, and returns
// a cryptographically-sealed proof bundle. Never blocks the HTTP response
// on the SQLite write — that happens in a background goroutine.

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/connextech/connex/internal/enrichment"
	"github.com/connextech/connex/internal/iso20022"
	"github.com/connextech/connex/internal/iso8583"
	"github.com/connextech/connex/internal/storage"
)

// ── Proof bundle ──────────────────────────────────────────────────────────────

type SignatureEntry struct {
	Witness     string `json:"witness"`
	Fingerprint string `json:"fingerprint"`
	Signature   string `json:"signature"`
	Timestamp   string `json:"timestamp"`
}

type Bundle struct {
	BundleID      string           `json:"bundle_id"`
	Timestamp     string           `json:"timestamp"`
	OriginalHash  string           `json:"original_hash"`  // hex
	EnrichedHash  string           `json:"enriched_hash"`  // hex
	PrevChainHash string           `json:"prev_chain_hash"` // hex
	ChainHash     string           `json:"chain_hash"`     // hex
	Signatures    []SignatureEntry  `json:"signatures"`
	QuorumStatus  string           `json:"quorum_status"`
	EnrichmentLog json.RawMessage  `json:"enrichment_log"`
}

// ── Witness client ────────────────────────────────────────────────────────────

type witnessAddr struct{ url string }

func requestSignature(addr string, token string, hashBytes []byte, timeout time.Duration) (*SignatureEntry, error) {
	body, _ := json.Marshal(map[string]string{
		"hash": base64.StdEncoding.EncodeToString(hashBytes),
	})
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("POST", addr+"/v1/sign", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("witness returned %d", resp.StatusCode)
	}
	var sig SignatureEntry
	if err := json.NewDecoder(resp.Body).Decode(&sig); err != nil {
		return nil, fmt.Errorf("decode signature response: %w", err)
	}
	return &sig, nil
}

// collectSignatures fires parallel requests to all witnesses and returns
// whatever signatures arrive within the timeout. Requires ≥2 for quorum.
func collectSignatures(witnesses []string, tokens []string, hashBytes []byte, timeout time.Duration) []SignatureEntry {
	type result struct {
		sig *SignatureEntry
		err error
	}
	ch := make(chan result, len(witnesses))
	for i, w := range witnesses {
		w := w
		var token string
		if i < len(tokens) {
			token = tokens[i]
		}
		go func() {
			sig, err := requestSignature(w, token, hashBytes, timeout)
			ch <- result{sig, err}
		}()
	}

	var sigs []SignatureEntry
	deadline := time.After(timeout)
	for range witnesses {
		select {
		case r := <-ch:
			if r.err != nil {
				slog.Warn("witness error", "err", r.err)
			} else {
				sigs = append(sigs, *r.sig)
			}
		case <-deadline:
			slog.Warn("witness timeout reached", "collected", len(sigs))
			return sigs
		}
	}
	return sigs
}

// ── Hash computation ──────────────────────────────────────────────────────────

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func coordinationHash(originalHash, enrichedHash, prevHash string) ([]byte, string) {
	origBytes, _ := hex.DecodeString(originalHash)
	enrichBytes, _ := hex.DecodeString(enrichedHash)
	prevBytes, _ := hex.DecodeString(prevHash)
	raw := append(origBytes, append(enrichBytes, prevBytes...)...)
	h := sha256.Sum256(raw)
	return h[:], hex.EncodeToString(h[:])
}

// ── Gateway handler ───────────────────────────────────────────────────────────

type gateway struct {
	witnesses []string
	tokens    []string
	db        *storage.DB
	mu        sync.Mutex // protects coordination chain updates
}

func (g *gateway) handleCoordinate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	// 1. Read and decode base64 ISO 8583 body
	rawBody, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	msgBytes, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(rawBody)))
	if err != nil {
		http.Error(w, fmt.Sprintf("base64 decode: %v", err), http.StatusBadRequest)
		return
	}

	// 2. Parse ISO 8583
	msg, err := iso8583.Parse(msgBytes)
	if err != nil {
		slog.Error("iso8583 parse failed", "err", err)
		http.Error(w, fmt.Sprintf("ISO 8583 parse error: %v", err), http.StatusBadRequest)
		return
	}

	// 3. Run enrichment engine
	in := &enrichment.Input{
		AmountKES:            msg.AmountKES(),
		CurrencyISO4217:      msg.CurrencyCode(),
		ProcessingCode:       msg.Fields[3],
		MerchantType:         msg.Fields[18],
		AcquiringInstitution: msg.Fields[32],
		ForwardingInstitution: msg.Fields[33],
		CrossInstitutional:   msg.Fields[32] != "" && msg.Fields[33] != "" && msg.Fields[32] != msg.Fields[33],
	}
	enrResult, err := enrichment.Enrich(in)
	if err != nil {
		slog.Error("enrichment failed", "err", err)
		http.Error(w, "enrichment error", http.StatusInternalServerError)
		return
	}

	// 4. Generate bundle ID and assemble ISO 20022 XML
	randBytes := make([]byte, 4)
	if _, err := rand.Read(randBytes); err != nil {
		slog.Error("rand read failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	bundleID := fmt.Sprintf("CX-%s-%x", time.Now().UTC().Format("20060102150405.000000"), randBytes)
	xmlBytes, err := iso20022.Assemble(msg, enrResult, bundleID)
	if err != nil {
		slog.Error("xml assembly/validation failed", "err", err)
		http.Error(w, fmt.Sprintf("XML error: %v", err), http.StatusUnprocessableEntity)
		return
	}

	// 5. Compute hashes
	originalHash := sha256Hex(msgBytes)
	enrichedHash := sha256Hex(xmlBytes)

	// 6. Lock coordination sequence to prevent parallel hash chain branching
	g.mu.Lock()
	defer g.mu.Unlock()

	prevHash, err := g.db.LatestChainHash()
	if err != nil {
		slog.Error("latest chain hash", "err", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	// 7. Compute coordination hash
	chainHashBytes, chainHashHex := coordinationHash(originalHash, enrichedHash, prevHash)

	// 8. Collect witness signatures (150ms timeout)
	sigs := collectSignatures(g.witnesses, g.tokens, chainHashBytes, 150*time.Millisecond)
	quorumStatus := "QUORUM_FAILED"
	if len(sigs) >= 2 {
		quorumStatus = "QUORUM_MET"
	}

	// 9. Build enrichment log JSON
	logJSON, _ := enrResult.Log.JSON()

	// 10. Assemble proof bundle
	bundle := Bundle{
		BundleID:      bundleID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		OriginalHash:  originalHash,
		EnrichedHash:  enrichedHash,
		PrevChainHash: prevHash,
		ChainHash:     chainHashHex,
		Signatures:    sigs,
		QuorumStatus:  quorumStatus,
		EnrichmentLog: json.RawMessage(logJSON),
	}

	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		slog.Error("marshal bundle failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 11. Write to SQLite synchronously - must succeed before returning 200 OK
	enrLogStr, _ := enrResult.Log.JSON()
	err = g.db.Insert(&storage.Event{
		BundleID:      bundleID,
		Timestamp:     time.Now().UTC(),
		OriginalHash:  originalHash,
		EnrichedHash:  enrichedHash,
		ChainHash:     chainHashHex,
		PrevChainHash: prevHash,
		BundleJSON:    string(bundleJSON),
		EnrichedXML:   string(xmlBytes),
		EnrichmentLog: enrLogStr,
		QuorumStatus:  quorumStatus,
	})
	if err != nil {
		slog.Error("db write failed", "bundle", bundleID, "err", err)
		http.Error(w, "database write error", http.StatusInternalServerError)
		return
	}

	// 12. Return bundle
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bundleJSON)

	slog.Info("coordinated",
		"bundle", bundleID,
		"quorum", quorumStatus,
		"sigs", len(sigs),
		"original_hash", originalHash[:16]+"...",
	)
}

func (g *gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	count, _ := g.db.Count()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"events":  count,
		"version": "v0.1.0",
	})
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	port    := flag.Int("port", 8080, "Gateway listening port")
	dbPath  := flag.String("db", "data/connex.db", "SQLite database path")
	w1      := flag.String("witness1", "http://localhost:8091", "Witness Alpha URL")
	w2      := flag.String("witness2", "http://localhost:8092", "Witness Beta URL")
	w3      := flag.String("witness3", "http://localhost:8093", "Witness Gamma URL")
	w1tok   := flag.String("w1token", "", "Token secret for witness Alpha")
	w2tok   := flag.String("w2token", "", "Token secret for witness Beta")
	w3tok   := flag.String("w3token", "", "Token secret for witness Gamma")
	flag.Parse()

	db, err := storage.Open(*dbPath)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	gw := &gateway{
		witnesses: []string{*w1, *w2, *w3},
		tokens:    []string{*w1tok, *w2tok, *w3tok},
		db:        db,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/coordinate", gw.handleCoordinate)
	mux.HandleFunc("/health", gw.handleHealth)

	count, _ := db.Count()
	slog.Info("gateway ready",
		"port", *port,
		"db", *dbPath,
		"existing_events", count,
		"witnesses", []string{*w1, *w2, *w3},
	)

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("gateway error", "err", err)
		os.Exit(1)
	}
}
