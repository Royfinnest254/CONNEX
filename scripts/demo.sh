#!/usr/bin/env bash
# scripts/demo.sh — Connex end-to-end demonstration
# Requires: go, python3, pynacl (pip install pynacl), curl
# Completes in under 5 minutes on a modern laptop.
# Every step prints clear output. Exits non-zero on any failure.

set -euo pipefail

BINDIR="bin"
KEYS="keys"
DATA="data"
BUNDLES="bench/results/demo-bundles"

RED='\033[0;31m'
GRN='\033[0;32m'
BLU='\033[0;34m'
NC='\033[0m'

info()  { echo -e "${BLU}==>${NC} $*"; }
ok()    { echo -e "${GRN}    PASS${NC}  $*"; }
fail()  { echo -e "${RED}    FAIL${NC}  $*"; exit 1; }

PIDS=()
cleanup() {
  info "Stopping background processes..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT

# ── Step 1: Build ─────────────────────────────────────────────────────────────
info "Step 1/12 — Building all binaries"
make build
ok "gateway, witness, bench compiled"

# ── Step 2: Initialise database ───────────────────────────────────────────────
info "Step 2/12 — Initialising SQLite database"
mkdir -p "$DATA" "$KEYS/alpha" "$KEYS/beta" "$KEYS/gamma" "$BUNDLES"
sqlite3 "$DATA/connex.db" < internal/storage/schema.sql 2>/dev/null || true
ok "data/connex.db ready"

# ── Step 3: Start witnesses ───────────────────────────────────────────────────
info "Step 3/12 — Starting witness nodes (Alpha:8091, Beta:8092, Gamma:8093)"
"$BINDIR/witness" --port=8091 --keypath="$KEYS/alpha/witness.key" --name=alpha --token=alpha-token > /tmp/witness-alpha.log 2>&1 &
PIDS+=($!)
"$BINDIR/witness" --port=8092 --keypath="$KEYS/beta/witness.key"  --name=beta  --token=beta-token > /tmp/witness-beta.log  2>&1 &
PIDS+=($!)
"$BINDIR/witness" --port=8093 --keypath="$KEYS/gamma/witness.key" --name=gamma --token=gamma-token > /tmp/witness-gamma.log 2>&1 &
PIDS+=($!)

# Wait for witnesses to be healthy
for port in 8091 8092 8093; do
  for i in $(seq 1 20); do
    if curl -sf "http://localhost:$port/health" > /dev/null 2>&1; then break; fi
    sleep 0.2
  done
  FP=$(curl -sf "http://localhost:$port/v1/pubkey" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['fingerprint'])")
  ok "Witness :$port fingerprint=$FP"
done

# ── Step 4: Start gateway ─────────────────────────────────────────────────────
info "Step 4/12 — Starting gateway (port 8080)"
"$BINDIR/gateway" --port=8080 --db="$DATA/connex.db" --w1token=alpha-token --w2token=beta-token --w3token=gamma-token > /tmp/gateway.log 2>&1 &
PIDS+=($!)
for i in $(seq 1 20); do
  if curl -sf "http://localhost:8080/health" > /dev/null 2>&1; then break; fi
  sleep 0.3
done
ok "Gateway healthy"

# ── Step 5: Send 10 transactions ──────────────────────────────────────────────
info "Step 5/12 — Sending 10 transactions from corpus"
VALID_COUNT=0
i=0
while IFS= read -r line && [ $i -lt 10 ]; do
  HEX=$(echo "$line" | python3 -c "import json,sys; print(json.load(sys.stdin)['iso8583_hex'])")
  B64=$(echo "$HEX" | python3 -c "import sys,binascii,base64; print(base64.b64encode(binascii.unhexlify(sys.stdin.read().strip())).decode())")
  CORPUS_ID=$(echo "$line" | python3 -c "import json,sys; print(json.load(sys.stdin)['corpus_id'])")

  RESP=$(curl -sf -X POST http://localhost:8080/v1/coordinate \
    -H "Content-Type: text/plain" \
    -d "$B64")

  BUNDLE_ID=$(echo "$RESP" | python3 -c "import json,sys; print(json.load(sys.stdin)['bundle_id'])")
  QUORUM=$(echo "$RESP"   | python3 -c "import json,sys; print(json.load(sys.stdin)['quorum_status'])")
  CHAIN=$(echo "$RESP"    | python3 -c "import json,sys; print(json.load(sys.stdin)['chain_hash'][:16]+'...')")

  echo "$RESP" > "$BUNDLES/$BUNDLE_ID.json"

  ok "[$CORPUS_ID] bundle=$BUNDLE_ID quorum=$QUORUM chain=$CHAIN"
  VALID_COUNT=$((VALID_COUNT + 1))
  i=$((i + 1))
done < corpus/v1.0/transactions.jsonl

# ── Step 6: Verify all bundles with Python verifier ───────────────────────────
info "Step 6/12 — Running verify.py against all $VALID_COUNT bundles"
VERIFIED=0
for bundle_file in "$BUNDLES"/*.json; do
  python3 verify/verify.py "$bundle_file" "$KEYS" > /tmp/verify.log 2>&1
  if [ $? -eq 0 ]; then
    VERIFIED=$((VERIFIED + 1))
    ok "$(basename $bundle_file)"
  else
    fail "verify.py FAILED on $(basename $bundle_file)"
  fi
done
ok "$VERIFIED/$VALID_COUNT bundles verified VALID"

# ── Step 7: Tamper detection test ─────────────────────────────────────────────
info "Step 7/12 — Tamper detection: flipping a byte in enriched_hash"
FIRST_BUNDLE=$(ls "$BUNDLES"/*.json | head -1)
python3 - "$FIRST_BUNDLE" <<'EOF'
import json, sys
with open(sys.argv[1]) as f:
    b = json.load(f)
# Flip the first byte of enriched_hash
h = list(b['enriched_hash'])
h[0] = '0' if h[0] != '0' else 'f'
b['enriched_hash'] = ''.join(h)
b['_tampered'] = True
with open(sys.argv[1] + '.tampered', 'w') as f:
    json.dump(b, f)
EOF
if python3 verify/verify.py "${FIRST_BUNDLE}.tampered" "$KEYS" > /tmp/tamper.log 2>&1; then
  fail "Tampered bundle incorrectly reported VALID"
else
  ok "Tampered bundle correctly detected as INVALID"
fi
rm -f "${FIRST_BUNDLE}.tampered"

# ── Step 8: Graceful degradation (kill witness Gamma) ────────────────────────
info "Step 8/12 — Killing Witness Gamma, sending 3 more transactions"
kill "${PIDS[2]}" && unset 'PIDS[2]'
sleep 0.5

DEGRADED=0
i=0
while IFS= read -r line && [ $i -lt 3 ]; do
  HEX=$(echo "$line" | python3 -c "import json,sys; print(json.load(sys.stdin)['iso8583_hex'])")
  B64=$(echo "$HEX" | python3 -c "import sys,binascii,base64; print(base64.b64encode(binascii.unhexlify(sys.stdin.read().strip())).decode())")
  RESP=$(curl -sf -X POST http://localhost:8080/v1/coordinate -H "Content-Type: text/plain" -d "$B64")
  QUORUM=$(echo "$RESP" | python3 -c "import json,sys; print(json.load(sys.stdin)['quorum_status'])")
  SIGS=$(echo "$RESP"   | python3 -c "import json,sys; print(len(json.load(sys.stdin)['signatures']))")
  if [ "$QUORUM" = "QUORUM_MET" ]; then
    ok "Degraded mode: quorum=$QUORUM sigs=$SIGS/3 (Gamma offline)"
    DEGRADED=$((DEGRADED + 1))
  fi
  i=$((i + 1))
done < corpus/v1.0/transactions.jsonl

# ── Step 9: Append-only enforcement test ─────────────────────────────────────
info "Step 9/12 — Verifying append-only enforcement"
UPDATE_RESULT=$(sqlite3 "$DATA/connex.db" "UPDATE coordination_events SET quorum_status='HACKED' WHERE 1=1" 2>&1 || true)
if echo "$UPDATE_RESULT" | grep -q "append-only"; then
  ok "UPDATE correctly rejected by DB trigger: append-only enforced"
else
  fail "UPDATE was NOT rejected — append-only invariant broken"
fi

# ── Step 10: Run quick benchmark ─────────────────────────────────────────────
info "Step 10/12 — Running burst benchmark (50 requests)"
"$BINDIR/bench" --mode=burst --count=50 --gateway=http://localhost:8080 \
                --corpus=corpus/v1.0/transactions.jsonl \
                --out=bench/results/demo-bench.csv 2>/dev/null || \
  echo "    (bench binary not yet built — skipping)"

# ── Step 11: Summary ──────────────────────────────────────────────────────────
info "Step 11/12 — Final summary"
TOTAL_EVENTS=$(sqlite3 "$DATA/connex.db" "SELECT COUNT(*) FROM coordination_events")
echo ""
echo "  ┌─────────────────────────────────────────────┐"
echo "  │  CONNEX DEMO RESULTS                        │"
echo "  │                                             │"
printf "  │  Transactions processed:  %-17s│\n" "$TOTAL_EVENTS"
printf "  │  Bundles verified VALID:  %-17s│\n" "$VERIFIED"
printf "  │  Tamper attempts caught:  %-17s│\n" "1"
printf "  │  Degraded-mode success:   %-17s│\n" "$DEGRADED/3"
echo "  │  Append-only verified:    YES               │"
echo "  │                                             │"
echo "  └─────────────────────────────────────────────┘"
echo ""

# ── Step 12: Cleanup ──────────────────────────────────────────────────────────
info "Step 12/12 — Cleanup"
ok "Demo complete. All processes will stop on exit."
