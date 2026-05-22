// cmd/witness/main.go — Connex Witness Node
//
// Generates a real Ed25519 keypair on first start (persisted to --keypath).
// Exposes two HTTP endpoints:
//   GET  /v1/pubkey  — returns public key (base64) and fingerprint (SHA-256[:16])
//   POST /v1/sign    — signs a 32-byte coordination hash with Ed25519
//
// Three identical binaries run on ports 8091, 8092, 8093 with different keypaths.
// No shared state. No knowledge of other witnesses.

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ── Keypair management ────────────────────────────────────────────────────────

// loadOrGenerate loads an Ed25519 keypair from disk or generates a new one.
// Private key is stored with mode 0600. Public key with mode 0644.
func loadOrGenerate(keyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	privPath := keyPath
	pubPath := keyPath + ".pub"

	privBytes, err := os.ReadFile(privPath)
	if err == nil {
		pubBytes, err2 := os.ReadFile(pubPath)
		if err2 == nil && len(privBytes) == ed25519.PrivateKeySize && len(pubBytes) == ed25519.PublicKeySize {
			slog.Info("keypair loaded from disk", "path", keyPath)
			return ed25519.PublicKey(pubBytes), ed25519.PrivateKey(privBytes), nil
		}
	}

	// Generate fresh keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate keypair: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, nil, fmt.Errorf("create key dir: %w", err)
	}
	if err := os.WriteFile(privPath, priv, 0600); err != nil {
		return nil, nil, fmt.Errorf("write private key: %w", err)
	}
	if err := os.WriteFile(pubPath, pub, 0644); err != nil {
		return nil, nil, fmt.Errorf("write public key: %w", err)
	}

	slog.Info("new keypair generated", "path", keyPath)
	return pub, priv, nil
}

// fingerprint returns the first 16 hex characters of SHA-256(pubkey).
func fingerprint(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:])[:16]
}

// ── HTTP server ───────────────────────────────────────────────────────────────

type witness struct {
	priv        ed25519.PrivateKey
	pub         ed25519.PublicKey
	fp          string
	witnessName string
	token       string
}

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

type signRequest struct {
	Hash string `json:"hash"` // base64-encoded 32 bytes
}

type signResponse struct {
	Witness     string `json:"witness"`
	Fingerprint string `json:"fingerprint"`
	Signature   string `json:"signature"` // base64 Ed25519 sig (64 bytes)
	Timestamp   string `json:"timestamp"`
}

func (w *witness) handleSign(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if w.token != "" {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + w.token
		if auth != expected {
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req signRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON body", http.StatusBadRequest)
		return
	}

	hashBytes, err := base64.StdEncoding.DecodeString(req.Hash)
	if err != nil || len(hashBytes) != 32 {
		http.Error(rw, "hash must be exactly 32 bytes (base64-encoded)", http.StatusBadRequest)
		return
	}

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)

	// Bind timestamp to signature: H_witness = SHA-256(H_coord || timestamp_bytes)
	h := sha256.New()
	h.Write(hashBytes)
	h.Write([]byte(timestamp))
	witnessHash := h.Sum(nil)

	// Real Ed25519 signature
	sig := ed25519.Sign(w.priv, witnessHash)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(signResponse{
		Witness:     w.witnessName,
		Fingerprint: w.fp,
		Signature:   base64.StdEncoding.EncodeToString(sig),
		Timestamp:   timestamp,
	})
}

func (w *witness) handleHealth(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]string{
		"status":      "ok",
		"witness":     w.witnessName,
		"fingerprint": w.fp,
	})
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	port    := flag.Int("port", 8091, "Port to listen on")
	keyPath := flag.String("keypath", "keys/witness.key", "Path for Ed25519 keypair storage")
	name    := flag.String("name", "witness", "Human-readable witness name (alpha/beta/gamma)")
	token   := flag.String("token", "", "Shared authentication token secret")
	flag.Parse()

	pub, priv, err := loadOrGenerate(*keyPath)
	if err != nil {
		slog.Error("keypair setup failed", "err", err)
		os.Exit(1)
	}

	fp := fingerprint(pub)
	w := &witness{priv: priv, pub: pub, fp: fp, witnessName: *name, token: *token}

	slog.Info("witness ready",
		"name", *name,
		"port", *port,
		"fingerprint", fp,
		"pubkey", base64.StdEncoding.EncodeToString(pub),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pubkey", w.handlePubkey)
	mux.HandleFunc("/v1/sign", w.handleSign)
	mux.HandleFunc("/health", w.handleHealth)

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
