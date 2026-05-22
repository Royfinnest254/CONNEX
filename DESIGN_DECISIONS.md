# Design Decisions

This document records the rationale for significant architectural and technology choices in the Connex rebuild.

## 1. Programming Language: Go
**Choice:** Go 1.22
**Rationale:** 
- **Concurrency:** The core problem (parallel witness signing) is perfectly handled by Go's goroutines and channels without the complexity of async/await or thread management.
- **Binary Distribution:** Compiles to a single static binary with zero runtime dependencies. This is critical for the "Senior Engineer Audit" requirement—no `npm install` or `pip install` required for the core runtime.
- **Standard Library:** Go's `crypto/ed25519` and `net/http` are battle-tested and efficient.
**Alternatives Considered:** Rust (too high development friction for MVP), Node.js (weak concurrency model for low-latency coordination).

## 2. Cryptography: Ed25519
**Choice:** Ed25519 (RFC 8032)
**Rationale:**
- **Performance:** Much faster signing and verification than RSA or ECDSA (P-256).
- **Security:** Deterministic signatures (no reliance on a high-quality RNG at signing time to prevent private key leakage).
- **Size:** Small keys (32 bytes) and signatures (64 bytes) keep the proof bundles compact and suitable for inclusion in payment metadata.
**Alternatives Considered:** RSA 4096 (too slow/bulky), ECDSA (non-deterministic nonce risk).

## 3. Storage: SQLite with Triggers
**Choice:** SQLite with `BEFORE UPDATE` and `BEFORE DELETE` triggers.
**Rationale:**
- **Zero Setup:** No database server to install or configure. `make build` is all you need.
- **Append-Only Enforcement:** Database-level triggers provide a secondary layer of protection for history. Even if the application logic has a bug, the database rejects modifications.
- **Auditability:** A single `.db` file is easy to copy, checksum, and audit.
**Alternatives Considered:** PostgreSQL (too much setup for a demo), LevelDB (no triggers/SQL for easy auditing).

## 4. Witness Protocol: Blind Signing
**Choice:** Witnesses sign a 32-byte hash without seeing the original data.
**Rationale:**
- **Privacy:** Witnesses do not see sensitive transaction details (amounts, account numbers).
- **Scalability:** Witnesses don't need to implement the complex ISO 8583 parsing or enrichment logic. They are simple "notaries."
- **Independence:** If a witness was required to validate the XML, it would need the exact same ruleset as the gateway, creating a "software monoculture" risk where a bug in the rules affects all nodes.
**Trade-off:** Witnesses can be tricked into signing "anything" that hashes to 32 bytes. We mitigate this by ensuring the Gateway output is verifiable by *any* third party using the public keys.

## 5. Pure Go SQLite (modernc.org)
**Choice:** `modernc.org/sqlite` instead of `github.com/mattn/go-sqlite3`.
**Rationale:**
- **No CGO:** `go-sqlite3` requires a C compiler (GCC) to build. On Windows, this is a major friction point. `modernc.org` is a pure Go port.
- **Portability:** Ensures the "Senior Engineer" can clone and build on *any* machine without installing a toolchain beyond Go itself.

## 6. Verification: Independent Python Script
**Choice:** `verify/verify.py` as a standalone script using only `pynacl`.
**Rationale:**
- **Credibility:** If the verifier were part of the Go codebase, a skeptic might suspect shared "tricks" or mocked logic. Using a different language (Python) and a different library (`pynacl`) proves that the cryptographic construction is standard and interoperable.
- **Audition:** 80 lines of Python is easier to audit in 2 minutes than a Go internal package.
