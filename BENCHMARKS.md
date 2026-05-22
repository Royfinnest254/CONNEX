# Benchmark Methodology & Results

This document describes the performance characteristics of the Connex Coordination Gateway.

## 1. Methodology
Benchmarks are generated using `cmd/bench`, an external load generator that measures end-to-end latency from the perspective of a client.

- **Environment:** All processes (Gateway, 3 Witnesses, Bench Harness) run on the same physical host to isolate coordination latency from network jitter.
- **Data Source:** `corpus/v1.0/transactions.jsonl` (500 synthetic ISO 8583 transactions).
- **Target:** < 50ms P99 latency.

---

## 2. Baseline Performance (Warm State)

| Metric | Measurement |
| :--- | :--- |
| **Throughput (TPS)** | ~250-500 transactions/sec |
| **P50 Latency** | ~8-12 ms |
| **P95 Latency** | ~15-20 ms |
| **P99 Latency** | ~25-35 ms |

*Note: Latency includes ISO 8583 parsing, enrichment, XSD validation, 2-of-3 witness signing, and SQLite background write orchestration.*

---

## 3. Quorum Degradation Performance
Measurement of the system when one witness node (Gamma) is offline.

| Metric | Measurement |
| :--- | :--- |
| **Active Quorum** | 2 of 2 available (1 offline) |
| **P50 Latency** | ~10 ms |
| **Success Rate** | 100% |

**Observation:** The gateway handles witness failure gracefully. Because witness requests are fired in parallel with a 150ms timeout, the failure of a single node does not block the overall response.

---

## 4. Reproducibility
Every benchmark run produces a **Manifest JSON** (e.g., `bench/results/run-20260516.manifest.json`) containing:
- OS, Kernel, CPU model.
- Go version.
- Git SHA of the code.
- Calculated P50/P95/P99.
- SHA-256 of the raw results CSV.

To reproduce these results on your machine:
```bash
make build
# Start gateway and witnesses (see demo.sh)
make bench
```
Check the generated manifest in `bench/results/`.
