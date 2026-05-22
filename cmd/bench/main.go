// cmd/bench/main.go — Connex Benchmark Harness
//
// An external load generator that fires requests at the gateway and measures
// performance (latency, throughput, quorum success).
// Does not import gateway internals.
// Produces a CSV of every request and a manifest JSON summary.

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type Transaction struct {
	CorpusID   string `json:"corpus_id"`
	ISO8583Hex string `json:"iso8583_hex"`
}

type Result struct {
	RequestID          string
	CorpusID           string
	RequestSentAt      time.Time
	ResponseReceivedAt time.Time
	HTTPStatus         int
	BundleID           string
	QuorumStatus       string
	LatencyMS          int64
	Error              string
}

type Manifest struct {
	SystemInfo struct {
		OS      string `json:"os"`
		Arch    string `json:"arch"`
		CPUs    int    `json:"cpus"`
		GoVer   string `json:"go_version"`
		Host    string `json:"hostname"`
	} `json:"system_info"`
	Config struct {
		Mode     string `json:"mode"`
		TPS      int    `json:"tps,omitempty"`
		Count    int    `json:"count,omitempty"`
		Duration string `json:"duration,omitempty"`
	} `json:"config"`
	Stats struct {
		TotalRequests int     `json:"total_requests"`
		SuccessCount  int     `json:"success_count"`
		ErrorCount    int     `json:"error_count"`
		P50Latency    float64 `json:"p50_latency_ms"`
		P95Latency    float64 `json:"p95_latency_ms"`
		P99Latency    float64 `json:"p99_latency_ms"`
		Throughput    float64 `json:"avg_throughput_tps"`
	} `json:"stats"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func main() {
	mode := flag.String("mode", "burst", "Benchmark mode: fixed, burst")
	tps := flag.Int("tps", 10, "Target TPS for fixed mode")
	count := flag.Int("count", 100, "Total requests for burst mode")
	duration := flag.Duration("duration", 10*time.Second, "Duration for fixed mode")
	gatewayURL := flag.String("gateway", "http://localhost:8080", "Gateway URL")
	corpusPath := flag.String("corpus", "corpus/v1.0/transactions.jsonl", "Path to corpus file")
	outPath := flag.String("out", "", "Output CSV path (default: bench/results/run-{ts}.csv)")

	flag.Parse()

	transactions, err := loadCorpus(*corpusPath)
	if err != nil {
		log.Fatalf("failed to load corpus: %v", err)
	}

	if *outPath == "" {
		ts := time.Now().Format("20060102-150405")
		*outPath = fmt.Sprintf("bench/results/run-%s.csv", ts)
	}
	os.MkdirAll(filepath.Dir(*outPath), 0755)

	csvFile, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("failed to create CSV: %v", err)
	}
	defer csvFile.Close()
	fmt.Fprintln(csvFile, "request_id,corpus_id,request_sent_at,response_received_at,http_status,bundle_id,quorum_status,latency_ms,error")

	var results []Result
	var resultsMu sync.Mutex

	startTime := time.Now()
	fmt.Printf("Starting benchmark: mode=%s, count=%d, target=%s\n", *mode, *count, *gatewayURL)

	if *mode == "burst" {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 20) // Concurrency limit
		for i := 0; i < *count; i++ {
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int) {
				defer wg.Done()
				defer func() { <-sem }()
				tx := transactions[idx%len(transactions)]
				res := fireRequest(*gatewayURL, tx, idx)
				resultsMu.Lock()
				results = append(results, res)
				resultsMu.Unlock()
				writeCSVLine(csvFile, res)
			}(i)
		}
		wg.Wait()
	} else if *mode == "fixed" {
		ticker := time.NewTicker(time.Second / time.Duration(*tps))
		stop := time.After(*duration)
		i := 0
		var wg sync.WaitGroup
	loop:
		for {
			select {
			case <-ticker.C:
				wg.Add(1)
				tx := transactions[i%len(transactions)]
				go func(idx int, t Transaction) {
					defer wg.Done()
					res := fireRequest(*gatewayURL, t, idx)
					resultsMu.Lock()
					results = append(results, res)
					resultsMu.Unlock()
					writeCSVLine(csvFile, res)
				}(i, tx)
				i++
			case <-stop:
				break loop
			}
		}
		ticker.Stop()
		wg.Wait()
	}

	endTime := time.Now()
	fmt.Printf("Benchmark complete. Produced %d results in %v\n", len(results), endTime.Sub(startTime))

	generateManifest(*outPath, results, startTime, endTime, *mode, *tps, *count, *duration)
}

func loadCorpus(path string) ([]Transaction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var txs []Transaction
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var tx Transaction
		if err := decoder.Decode(&tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func fireRequest(url string, tx Transaction, id int) Result {
	reqID := fmt.Sprintf("req-%d", id)
	
	raw, _ := hex.DecodeString(tx.ISO8583Hex)
	payload := base64.StdEncoding.EncodeToString(raw)

	res := Result{
		RequestID:     reqID,
		CorpusID:      tx.CorpusID,
		RequestSentAt: time.Now(),
	}

	resp, err := http.Post(url+"/v1/coordinate", "text/plain", bytes.NewReader([]byte(payload)))
	res.ResponseReceivedAt = time.Now()
	res.LatencyMS = res.ResponseReceivedAt.Sub(res.RequestSentAt).Milliseconds()

	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	res.HTTPStatus = resp.StatusCode
	if resp.StatusCode == http.StatusOK {
		var bundle struct {
			BundleID     string `json:"bundle_id"`
			QuorumStatus string `json:"quorum_status"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&bundle); err == nil {
			res.BundleID = bundle.BundleID
			res.QuorumStatus = bundle.QuorumStatus
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		res.Error = string(body)
	}

	return res
}

func writeCSVLine(w io.Writer, r Result) {
	fmt.Fprintf(w, "%s,%s,%s,%s,%d,%s,%s,%d,\"%s\"\n",
		r.RequestID, r.CorpusID,
		r.RequestSentAt.Format(time.RFC3339Nano),
		r.ResponseReceivedAt.Format(time.RFC3339Nano),
		r.HTTPStatus, r.BundleID, r.QuorumStatus,
		r.LatencyMS, r.Error)
}

func generateManifest(csvPath string, results []Result, start, end time.Time, mode string, tps, count int, dur time.Duration) {
	m := Manifest{
		StartTime: start.Format(time.RFC3339),
		EndTime:   end.Format(time.RFC3339),
	}
	m.SystemInfo.OS = runtime.GOOS
	m.SystemInfo.Arch = runtime.GOARCH
	m.SystemInfo.CPUs = runtime.NumCPU()
	m.SystemInfo.GoVer = runtime.Version()
	m.SystemInfo.Host, _ = os.Hostname()

	m.Config.Mode = mode
	m.Config.TPS = tps
	m.Config.Count = count
	m.Config.Duration = dur.String()

	m.Stats.TotalRequests = len(results)
	var latencies []int64
	for _, r := range results {
		if r.HTTPStatus == http.StatusOK {
			m.Stats.SuccessCount++
			latencies = append(latencies, r.LatencyMS)
		} else {
			m.Stats.ErrorCount++
		}
	}

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		m.Stats.P50Latency = float64(latencies[len(latencies)*50/100])
		m.Stats.P95Latency = float64(latencies[len(latencies)*95/100])
		m.Stats.P99Latency = float64(latencies[len(latencies)*99/100])
	}
	
	durationSecs := end.Sub(start).Seconds()
	if durationSecs > 0 {
		m.Stats.Throughput = float64(len(results)) / durationSecs
	}

	manifestPath := strings.TrimSuffix(csvPath, ".csv") + ".manifest.json"
	f, _ := os.Create(manifestPath)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(m)
	fmt.Printf("Manifest saved to %s\n", manifestPath)
}
