# scripts/demo.py — Connex end-to-end demonstration (Windows/Cross-platform)
import os
import sys
import time
import json
import base64
import sqlite3
import binascii
import subprocess
import urllib.request
import urllib.error

# ANSI Color codes
RED = '\033[91m'
GRN = '\033[92m'
BLU = '\033[94m'
CYN = '\033[96m'
YEL = '\033[93m'
NC = '\033[0m'

def info(msg):
    print(f"{BLU}==>{NC} {msg}")

def success(msg):
    print(f"{GRN}    PASS{NC}  {msg}")

def warn(msg):
    print(f"{YEL}    WARN{NC}  {msg}")

def fail(msg):
    print(f"{RED}    FAIL{NC}  {msg}")
    sys.exit(1)

def http_get(url):
    req = urllib.request.Request(url)
    with urllib.request.urlopen(req) as r:
        return json.loads(r.read().decode('utf-8'))

def http_post(url, data_str, headers):
    req = urllib.request.Request(url, data=data_str.encode('utf-8'), headers=headers, method="POST")
    with urllib.request.urlopen(req) as r:
        return json.loads(r.read().decode('utf-8'))

def decode_hex_lenient(hex_str):
    hex_chars = []
    for c in hex_str:
        if c in '0123456789abcdefABCDEF':
            hex_chars.append(c)
        else:
            break
    cleaned = "".join(hex_chars)
    if len(cleaned) % 2 != 0:
        cleaned = cleaned[:-1]
    return binascii.unhexlify(cleaned)

def main():
    processes = []
    
    try:
        # ── Step 1: Build confirmation ─────────────────────────────────────────
        info("Step 1/12 — Confirming Compiled Binaries")
        bin_dir = "bin"
        witness_exe = os.path.join(bin_dir, "witness.exe" if os.name == "nt" else "witness")
        gateway_exe = os.path.join(bin_dir, "gateway.exe" if os.name == "nt" else "gateway")
        bench_exe = os.path.join(bin_dir, "bench.exe" if os.name == "nt" else "bench")
        
        for exe in [witness_exe, gateway_exe, bench_exe]:
            if not os.path.exists(exe):
                fail(f"Required binary not found: {exe}. Please compile first.")
        success("gateway, witness, bench binaries confirmed")
        
        # ── Step 2: Initialise database ──────────────────────────────────────────
        info("Step 2/12 — Initialising SQLite Database")
        data_dir = "data"
        keys_dir = "keys"
        bundles_dir = os.path.join("bench", "results", "demo-bundles")
        
        os.makedirs(data_dir, exist_ok=True)
        os.makedirs(os.path.join(keys_dir, "alpha"), exist_ok=True)
        os.makedirs(os.path.join(keys_dir, "beta"), exist_ok=True)
        os.makedirs(os.path.join(keys_dir, "gamma"), exist_ok=True)
        os.makedirs(bundles_dir, exist_ok=True)
        for f in os.listdir(bundles_dir):
            if f.endswith(".json") or f.endswith(".tampered"):
                try:
                    os.remove(os.path.join(bundles_dir, f))
                except OSError:
                    pass
        
        db_path = os.path.join(data_dir, "connex.db")
        if os.path.exists(db_path):
            try:
                os.remove(db_path)
            except OSError:
                pass
                
        conn = sqlite3.connect(db_path)
        with open(os.path.join("internal", "storage", "schema.sql")) as f:
            conn.executescript(f.read())
        conn.close()
        success(f"{db_path} ready with triggers")
        
        # ── Step 3: Start witnesses ──────────────────────────────────────────────
        info("Step 3/12 — Starting Witness Nodes (Alpha:8091, Beta:8092, Gamma:8093)")
        witness_configs = [
            {"port": 8091, "keypath": os.path.join(keys_dir, "alpha", "witness.key"), "name": "alpha", "token": "alpha-token"},
            {"port": 8092, "keypath": os.path.join(keys_dir, "beta", "witness.key"), "name": "beta", "token": "beta-token"},
            {"port": 8093, "keypath": os.path.join(keys_dir, "gamma", "witness.key"), "name": "gamma", "token": "gamma-token"},
        ]
        
        for wc in witness_configs:
            log_file = open(os.path.join(data_dir, f"witness-{wc['name']}.log"), "w")
            p = subprocess.Popen([
                witness_exe,
                f"--port={wc['port']}",
                f"--keypath={wc['keypath']}",
                f"--name={wc['name']}",
                f"--token={wc['token']}"
            ], stdout=log_file, stderr=subprocess.STDOUT)
            processes.append(p)
            
        # Wait for witnesses to start
        for wc in witness_configs:
            url = f"http://localhost:{wc['port']}/health"
            healthy = False
            for _ in range(30):
                try:
                    res = http_get(url)
                    if res.get("status") == "ok":
                        healthy = True
                        break
                except Exception:
                    pass
                time.sleep(0.2)
            if not healthy:
                fail(f"Witness {wc['name']} failed to become healthy")
                
            # Fetch fingerprint
            pubkey_res = http_get(f"http://localhost:{wc['port']}/v1/pubkey")
            success(f"Witness {wc['name']} active on :{wc['port']}, fingerprint={pubkey_res['fingerprint']}")
            
        # ── Step 4: Start gateway ────────────────────────────────────────────────
        info("Step 4/12 — Starting Gateway (port 8080)")
        gw_log = open(os.path.join(data_dir, "gateway.log"), "w")
        gw_process = subprocess.Popen([
            gateway_exe,
            "--port=8080",
            f"--db={db_path}",
            "--w1token=alpha-token",
            "--w2token=beta-token",
            "--w3token=gamma-token"
        ], stdout=gw_log, stderr=subprocess.STDOUT)
        processes.append(gw_process)
        
        # Wait for gateway health
        gw_healthy = False
        for _ in range(30):
            try:
                res = http_get("http://localhost:8080/health")
                if res.get("status") == "ok":
                    gw_healthy = True
                    break
            except Exception:
                pass
            time.sleep(0.3)
        if not gw_healthy:
            fail("Gateway failed to become healthy")
        success("Gateway healthy")
        
        # ── Step 5: Send 10 transactions ──────────────────────────────────────────
        info("Step 5/12 — Coordinating 10 Transactions From Corpus")
        transactions = []
        with open(os.path.join("corpus", "v1.0", "transactions.jsonl")) as f:
            for line in f:
                transactions.append(json.loads(line))
                
        valid_count = 0
        for i in range(min(10, len(transactions))):
            tx = transactions[i]
            iso_hex = tx["iso8583_hex"]
            # Convert to base64
            raw_bytes = decode_hex_lenient(iso_hex.strip())
            b64_data = base64.b64encode(raw_bytes).decode('utf-8')
            
            try:
                resp = http_post("http://localhost:8080/v1/coordinate", b64_data, {"Content-Type": "text/plain"})
                bundle_id = resp["bundle_id"]
                quorum = resp["quorum_status"]
                chain_hash_short = resp["chain_hash"][:16] + "..."
                
                # Save response
                bundle_file = os.path.join(bundles_dir, f"{bundle_id}.json")
                with open(bundle_file, "w") as bf:
                    json.dump(resp, bf, indent=2)
                
                success(f"[{tx['corpus_id']}] bundle={bundle_id} quorum={quorum} chain={chain_hash_short}")
                valid_count += 1
            except Exception as e:
                fail(f"Failed to coordinate transaction {i}: {e}")
                
        # ── Step 6: Verify all bundles with python verifier ────────────────────────
        info(f"Step 6/12 — Running verify.py against all {valid_count} bundles")
        verified_count = 0
        for fname in os.listdir(bundles_dir):
            if not fname.endswith(".json"):
                continue
            bpath = os.path.join(bundles_dir, fname)
            
            # Execute verify.py as a subprocess
            res = subprocess.run([
                sys.executable,
                os.path.join("verify", "verify.py"),
                bpath,
                keys_dir
            ], capture_output=True, text=True)
            
            if res.returncode == 0:
                verified_count += 1
                success(f"{fname} verified VALID")
            else:
                print(res.stdout)
                print(res.stderr)
                fail(f"verify.py FAILED on {fname}")
                
        success(f"{verified_count}/{valid_count} bundles verified VALID")
        
        # ── Step 7: Tamper detection test ──────────────────────────────────────────
        info("Step 7/12 — Tamper Detection: Flipping a byte in enriched_hash")
        first_bundle_name = [f for f in os.listdir(bundles_dir) if f.endswith(".json")][0]
        first_bundle_path = os.path.join(bundles_dir, first_bundle_name)
        
        with open(first_bundle_path) as f:
            bdata = json.load(f)
            
        # Tamper: flip first char of enriched_hash
        eh = list(bdata["enriched_hash"])
        eh[0] = '0' if eh[0] != '0' else 'f'
        bdata["enriched_hash"] = ''.join(eh)
        
        tampered_path = first_bundle_path + ".tampered"
        with open(tampered_path, "w") as f:
            json.dump(bdata, f)
            
        res = subprocess.run([
            sys.executable,
            os.path.join("verify", "verify.py"),
            tampered_path,
            keys_dir
        ], capture_output=True)
        
        try:
            os.remove(tampered_path)
        except OSError:
            pass
            
        if res.returncode == 0:
            fail("Tampered bundle incorrectly reported VALID")
        else:
            success("Tampered bundle correctly detected as INVALID")
            
        # ── Step 8: Graceful degradation (kill witness Gamma) ─────────────────────
        info("Step 8/12 — Killing Witness Gamma (8093), Sending 3 More Transactions")
        # Find and terminate Gamma process
        gamma_p = processes[2]
        gamma_p.terminate()
        gamma_p.wait()
        
        degraded_success = 0
        for i in range(10, min(13, len(transactions))):
            tx = transactions[i]
            iso_hex = tx["iso8583_hex"]
            raw_bytes = decode_hex_lenient(iso_hex.strip())
            b64_data = base64.b64encode(raw_bytes).decode('utf-8')
            
            try:
                resp = http_post("http://localhost:8080/v1/coordinate", b64_data, {"Content-Type": "text/plain"})
                quorum = resp["quorum_status"]
                sigs_count = len(resp["signatures"])
                
                if quorum == "QUORUM_MET" and sigs_count == 2:
                    success(f"Degraded mode: quorum={quorum} sigs={sigs_count}/3 (Gamma offline)")
                    degraded_success += 1
                else:
                    fail(f"Unexpected degraded response: quorum={quorum} sigs={sigs_count}")
            except Exception as e:
                fail(f"Failed to coordinate in degraded mode: {e}")
                
        # ── Step 9: Append-only enforcement test ──────────────────────────────────
        info("Step 9/12 — Verifying DB Append-Only Enforcement")
        db_conn = sqlite3.connect(db_path)
        cursor = db_conn.cursor()
        
        update_rejected = False
        try:
            cursor.execute("UPDATE coordination_events SET quorum_status='HACKED' WHERE 1=1")
            db_conn.commit()
        except sqlite3.Error as e:
            if "append-only" in str(e):
                update_rejected = True
                
        db_conn.close()
        
        if update_rejected:
            success("UPDATE correctly rejected by DB trigger: append-only enforced")
        else:
            fail("UPDATE was NOT rejected — append-only invariant broken")
            
        # ── Step 10: Run quick benchmark ──────────────────────────────────────────
        info("Step 10/12 — Running Burst Benchmark (50 requests)")
        bench_out = os.path.join("bench", "results", "demo-bench.csv")
        if os.path.exists(bench_out):
            try:
                os.remove(bench_out)
            except OSError:
                pass
                
        res = subprocess.run([
            bench_exe,
            "--mode=burst",
            "--count=50",
            "--gateway=http://localhost:8080",
            "--corpus=corpus/v1.0/transactions.jsonl",
            f"--out={bench_out}"
        ], capture_output=True, text=True)
        
        if res.returncode == 0:
            success("Burst benchmark ran successfully")
        else:
            warn(f"Benchmark execution failed or returned warning:\n{res.stdout}\n{res.stderr}")
            
        # ── Step 11: Final Summary ───────────────────────────────────────────────
        info("Step 11/12 — Final Summary")
        db_conn = sqlite3.connect(db_path)
        cursor = db_conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM coordination_events")
        total_events = cursor.fetchone()[0]
        db_conn.close()
        
        print("\n"
              "  +---------------------------------------------+\n"
              "  |  CONNEX DEMO RESULTS                        |\n"
              "  |                                             |\n"
             f"  |  Transactions processed:  {total_events:<18}|\n"
             f"  |  Bundles verified VALID:  {verified_count:<18}|\n"
             f"  |  Tamper attempts caught:  1                 |\n"
             f"  |  Degraded-mode success:   {f'{degraded_success}/3':<18}|\n"
              "  |  Append-only verified:    YES               |\n"
              "  |                                             |\n"
              "  +---------------------------------------------+\n")
              
        # ── Step 12: Cleanup ─────────────────────────────────────────────────────
        info("Step 12/12 — Cleanup")
        success("Demo complete. Stopping background processes.")
        
    finally:
        # Terminate all processes
        for p in processes:
            try:
                p.terminate()
                p.wait(timeout=1.0)
            except Exception:
                try:
                    p.kill()
                except Exception:
                    pass
                    
if __name__ == "__main__":
    # Support ANSI colors on Windows Terminal
    if os.name == 'nt':
        import ctypes
        kernel32 = ctypes.windll.kernel32
        kernel32.SetConsoleMode(kernel32.GetStdHandle(-11), 7)
        
    main()
