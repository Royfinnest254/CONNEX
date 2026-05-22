# Connex Makefile
# Targets: build, test, demo, clean, witness-test

GOFLAGS := -trimpath
BINDIR  := bin
DATA    := data
KEYS    := keys

.PHONY: all build test demo clean witness-test bench

all: build

## build — compile all binaries
build:
	@echo "==> Building binaries..."
	@mkdir -p $(BINDIR)
	go build $(GOFLAGS) -o $(BINDIR)/witness  ./cmd/witness
	go build $(GOFLAGS) -o $(BINDIR)/gateway  ./cmd/gateway
	go build $(GOFLAGS) -o $(BINDIR)/bench    ./cmd/bench
	@echo "    OK: $(BINDIR)/witness, $(BINDIR)/gateway, $(BINDIR)/bench"

## test — run all unit tests
test:
	go test ./...

## witness-test — start witnesses, sign a test hash, verify with Python
witness-test: build
	@echo "==> Witness integration test..."
	@mkdir -p $(KEYS)/alpha $(KEYS)/beta $(KEYS)/gamma $(DATA)
	@$(BINDIR)/witness --port=8091 --keypath=$(KEYS)/alpha/witness.key --name=alpha &
	@$(BINDIR)/witness --port=8092 --keypath=$(KEYS)/beta/witness.key  --name=beta  &
	@$(BINDIR)/witness --port=8093 --keypath=$(KEYS)/gamma/witness.key --name=gamma &
	@sleep 1
	@echo "==> Fetching public keys..."
	@curl -sf http://localhost:8091/v1/pubkey | python3 -m json.tool
	@curl -sf http://localhost:8092/v1/pubkey | python3 -m json.tool
	@curl -sf http://localhost:8093/v1/pubkey | python3 -m json.tool
	@echo "==> Signing test hash (32 zero bytes in base64)..."
	@curl -sf -X POST http://localhost:8091/v1/sign \
	     -H 'Content-Type: application/json' \
	     -d '{"hash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}' | python3 -m json.tool
	@pkill -f "witness --port=809" || true
	@echo "==> Witness test PASSED"

## demo — run the full end-to-end demonstration
demo: build
	@bash scripts/demo.sh

## bench — run the benchmark harness (100 requests, burst mode)
bench: build
	@mkdir -p bench/results
	$(BINDIR)/bench --mode=burst --count=100 --gateway=http://localhost:8080 \
	                --corpus=corpus/v1.0/transactions.jsonl

## clean — remove build artifacts (NOT keys or database)
clean:
	rm -rf $(BINDIR)
	go clean -cache

## init-db — initialise an empty SQLite database
init-db:
	@mkdir -p $(DATA)
	@sqlite3 $(DATA)/connex.db < internal/storage/schema.sql
	@echo "Database initialised at $(DATA)/connex.db"
