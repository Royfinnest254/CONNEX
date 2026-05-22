#!/usr/bin/env python3
"""
verify.py — Independent Connex bundle verifier.

No dependency on the gateway binary. Uses only pynacl.
A third party can run this against any bundle using only the witness public keys.

Usage:
    python3 verify/verify.py <bundle.json> <keys_dir>

Exit codes:
    0 — VALID (≥2 signatures verify against the recomputed coordination hash)
    1 — INVALID (hash mismatch or fewer than 2 valid signatures)
    2 — Usage error or missing dependency
"""

import json
import sys
import hashlib
import base64
from pathlib import Path

try:
    import nacl.signing
    import nacl.exceptions
except ImportError:
    print("ERROR: pynacl required.  pip install pynacl")
    sys.exit(2)


def recompute_chain_hash(original_hash_hex: str,
                         enriched_hash_hex: str,
                         prev_hash_hex: str) -> bytes:
    """
    Recompute the coordination hash from its three components.
    Must match the gateway implementation exactly:
        chain_hash = SHA-256(original_hash || enriched_hash || prev_hash)
    All inputs are hex strings; returns raw bytes.
    """
    original = bytes.fromhex(original_hash_hex)
    enriched = bytes.fromhex(enriched_hash_hex)
    prev     = bytes.fromhex(prev_hash_hex)
    return hashlib.sha256(original + enriched + prev).digest()


def find_public_key(keys_dir: Path, fingerprint: str) -> bytes | None:
    """
    Find a witness public key by fingerprint (first 16 hex chars of SHA-256).
    Scans all .pub files in keys_dir.
    """
    for pub_file in keys_dir.rglob("*.pub"):
        raw = pub_file.read_bytes()
        if len(raw) != 32:          # Ed25519 public keys are exactly 32 bytes
            continue
        fp = hashlib.sha256(raw).hexdigest()[:16]
        if fp == fingerprint:
            return raw
    return None


def get_payload_hash(file_path: str, is_base64: bool = False) -> str:
    content = Path(file_path).read_bytes()
    if is_base64:
        try:
            decoded = base64.b64decode(content.strip(), validate=True)
            return hashlib.sha256(decoded).hexdigest()
        except Exception:
            pass
    return hashlib.sha256(content).hexdigest()


def verify_bundle(bundle_path: str, keys_dir: str, raw_iso_path: str = None, enriched_xml_path: str = None) -> bool:
    print(f"\n{'-'*60}")
    print(f"Bundle:   {bundle_path}")
    print(f"Keys dir: {keys_dir}")
    if raw_iso_path:
        print(f"Raw ISO:  {raw_iso_path}")
    if enriched_xml_path:
        print(f"XML:      {enriched_xml_path}")
    print(f"{'-'*60}")

    with open(bundle_path) as f:
        bundle = json.load(f)

    # Step 0: optional payload verification
    if raw_iso_path:
        iso_hash = get_payload_hash(raw_iso_path, is_base64=True)
        if iso_hash != bundle["original_hash"]:
            raw_iso_hash = get_payload_hash(raw_iso_path, is_base64=False)
            if raw_iso_hash == bundle["original_hash"]:
                iso_hash = raw_iso_hash
        if iso_hash != bundle["original_hash"]:
            print("INVALID  original payload hash mismatch")
            print(f"  computed: {iso_hash}")
            print(f"  expected: {bundle['original_hash']}")
            return False
        print("Payload ISO 8583 Hash OK")
    else:
        print("WARNING: No raw ISO 8583 payload path provided; payload content not verified.")

    if enriched_xml_path:
        xml_hash = get_payload_hash(enriched_xml_path, is_base64=False)
        if xml_hash != bundle["enriched_hash"]:
            print("INVALID  enriched payload hash mismatch")
            print(f"  computed: {xml_hash}")
            print(f"  expected: {bundle['enriched_hash']}")
            return False
        print("Payload Enriched XML Hash OK")
    else:
        print("WARNING: No enriched XML payload path provided; payload content not verified.")

    # Step 1: recompute coordination hash
    computed = recompute_chain_hash(
        bundle["original_hash"],
        bundle["enriched_hash"],
        bundle["prev_chain_hash"],
    )
    stored = bytes.fromhex(bundle["chain_hash"])

    if computed != stored:
        print("INVALID  hash mismatch — data has been tampered")
        print(f"  computed: {computed.hex()}")
        print(f"  stored:   {stored.hex()}")
        return False

    print(f"Hash OK  {computed.hex()[:32]}…")

    # Step 2: verify signatures
    signatures = bundle.get("signatures") or []
    valid_count = 0

    for entry in signatures:
        fp  = entry["fingerprint"]
        sig = base64.b64decode(entry["signature"])
        ts  = entry["timestamp"]

        pub_bytes = find_public_key(Path(keys_dir), fp)
        if pub_bytes is None:
            print(f"  SKIP  no public key found for fingerprint {fp}")
            continue

        # Bind timestamp to witness signature verify: SHA-256(H_coord || timestamp_bytes)
        witness_hash = hashlib.sha256(computed + ts.encode('utf-8')).digest()

        try:
            vk = nacl.signing.VerifyKey(pub_bytes)
            vk.verify(witness_hash, sig)
            valid_count += 1
            print(f"  SIG OK    witness={entry.get('witness','?')}  fp={fp}")
        except nacl.exceptions.BadSignatureError:
            print(f"  SIG FAIL  witness={entry.get('witness','?')}  fp={fp}")

    # Step 3: quorum check
    quorum_met = valid_count >= 2
    if quorum_met:
        print(f"\nVALID    {valid_count}/3 witnesses verified — quorum met")
    else:
        print(f"\nINVALID  {valid_count}/3 witnesses verified — quorum NOT met (need ≥2)")
    return quorum_met


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <bundle.json> <keys_dir> [raw_iso_path] [enriched_xml_path]")
        sys.exit(2)

    raw_iso = sys.argv[3] if len(sys.argv) > 3 else None
    enriched_xml = sys.argv[4] if len(sys.argv) > 4 else None

    ok = verify_bundle(sys.argv[1], sys.argv[2], raw_iso, enriched_xml)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
