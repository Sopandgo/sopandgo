#!/usr/bin/env python3
"""Check a SOPandGO audit evidence bundle without SOPandGO.

Usage:
    python3 -I verify_anchors.py sopandgo-audit-evidence-YYYYMMDD-HHMMSS.zip
    python3 -I verify_anchors.py --self-test

The bundle comes from Admin -> Integrity -> Download evidence bundle. This
script uses only the Python standard library. It:

  1. recomputes every audit row hash and checks each row links to the one
     before it (the algorithm is in docs/ops/verifying-anchors.md);
  2. checks every receipt's .json names a row that still has the anchored hash,
     with no unexplained broken row at or before it;
  3. runs `openssl ts -verify` on every receipt when OpenSSL is installed,
     otherwise prints the commands to run.

Get the TSA's CA certificate from the TSA itself and compare it with
tsa-ca.pem before trusting step 3.

Exit status: 0 when everything checks out, 1 otherwise.
"""

import hashlib
import json
import os
import shutil
import struct
import subprocess
import sys
import tempfile
import zipfile

GENESIS = "0" * 64
V2_DOMAIN = b"sopandgo-audit-v2"


def row_hash(row):
    """Return the row hash for the row's hash_version."""
    version = row.get("hash_version", 1)
    if version == 1:
        data = (row["prev_hash"] + row["event_type"] + row["entity_id"]
                + row["payload"] + row["created_at"])
        return hashlib.sha256(data.encode("utf-8")).hexdigest()
    if version == 2:
        h = hashlib.sha256(V2_DOMAIN)
        fields = [
            row["prev_hash"], row["event_type"], row["entity_type"],
            row["entity_id"], row.get("actor_user_id") or "",
            row["payload"], row["created_at"],
        ]
        for field in fields:
            b = field.encode("utf-8")
            h.update(struct.pack(">I", len(b)))
            h.update(b)
        return h.hexdigest()
    raise ValueError("unknown hash_version %r" % version)


def verify_chain(rows):
    """Return (failures, known_broken) like SOPandGO's integrity scan."""
    failures = {}
    known = {}
    expected_prev = GENESIS
    for row in rows:
        if row["prev_hash"] != expected_prev:
            failures[row["id"]] = "broken_chain_link"
        elif row_hash(row) != row["hash"]:
            failures[row["id"]] = "hash_mismatch"
        # Resync on the stored hash so one edit is reported once.
        expected_prev = row["hash"]
        if row["event_type"] == "integrity_baseline":
            try:
                for f in json.loads(row["payload"]).get("known_broken", []):
                    known[f["event_id"]] = f["error"]
            except (ValueError, KeyError, TypeError):
                pass
    return failures, known


def openssl_verify(tsr_path, tip_hash, ca_path):
    cmd = ["openssl", "ts", "-verify", "-digest", tip_hash,
           "-in", tsr_path, "-CAfile", ca_path, "-partial_chain"]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    out = (proc.stdout + proc.stderr).strip()
    return proc.returncode == 0 and "Verification: OK" in out, out


def check_bundle(path):
    ok = True
    with zipfile.ZipFile(path) as z:
        names = set(z.namelist())
        rows = [json.loads(line) for line in
                z.read("audit_events.jsonl").decode("utf-8").splitlines() if line.strip()]
        sidecars = sorted(n for n in names if n.startswith("anchors/") and n.endswith(".json"))
        tmp = tempfile.mkdtemp(prefix="sopandgo-verify-")
        try:
            z.extractall(tmp, [n for n in names if n.startswith("anchors/") or n == "tsa-ca.pem"])
            ok = report(rows, sidecars, z, tmp) and ok
        finally:
            shutil.rmtree(tmp, ignore_errors=True)
    return ok


def report(rows, sidecars, z, tmp):
    ok = True
    by_id = {r["id"]: r for r in rows}
    failures, known = verify_chain(rows)
    unexpected = {i: e for i, e in failures.items() if i not in known}

    print("Audit rows: %d" % len(rows))
    for i in sorted(failures):
        tag = "known (accepted when anchoring was switched on)" if i in known else "PROBLEM"
        print("  row %d: %s - %s" % (i, failures[i], tag))
    if unexpected:
        ok = False

    have_openssl = shutil.which("openssl") is not None
    ca_path = os.path.join(tmp, "tsa-ca.pem")
    print("Receipts: %d" % len(sidecars))
    for name in sidecars:
        side = json.loads(z.read(name).decode("utf-8"))
        event_id, tip = side["event_id"], side["tip_hash"]
        label = "row %d at %s" % (event_id, side.get("gen_time", "?"))
        row = by_id.get(event_id)
        if row is None:
            print("  %s: PROBLEM - row is missing from the audit log" % label)
            ok = False
            continue
        if row["hash"] != tip:
            print("  %s: PROBLEM - row hash differs from the anchored hash" % label)
            ok = False
            continue
        bad = [i for i in unexpected if i <= event_id]
        if bad:
            print("  %s: PROBLEM - covered row %d fails verification" % (label, min(bad)))
            ok = False
            continue
        tsr_path = os.path.join(tmp, name[:-len(".json")] + ".tsr")
        if have_openssl:
            good, out = openssl_verify(tsr_path, tip, ca_path)
            if good:
                print("  %s: OK (chain matches, openssl verified the receipt)" % label)
            else:
                print("  %s: PROBLEM - openssl: %s" % (label, out.splitlines()[-1] if out else "failed"))
                ok = False
        else:
            print("  %s: chain matches; verify the receipt with:" % label)
            print("    openssl ts -verify -digest %s -in %s -CAfile tsa-ca.pem -partial_chain"
                  % (tip, name[:-len(".json")] + ".tsr"))

    if sidecars:
        newest = max(json.loads(z.read(n).decode("utf-8"))["event_id"] for n in sidecars)
        later = sum(1 for r in rows if r["id"] > newest)
        if later:
            print("Note: %d row(s) after row %d are not covered by any receipt yet." % (later, newest))
    print("RESULT: %s" % ("OK" if ok else "PROBLEMS FOUND"))
    return ok


def self_test():
    """Print the reference vectors from backend/internal/audit/hash_test.go."""
    base = {
        "prev_hash": GENESIS, "event_type": "user_created", "entity_type": "user",
        "entity_id": "1", "payload": '{"name":"Alice"}',
        "created_at": "2026-11-18T10:00:00.123456789Z",
    }
    cases = [
        (dict(base, hash_version=1, actor_user_id="user-123"),
         "5138f3d19128405dad31fba13596f2479534d4cf0b185076fc06442d5ec985f8"),
        (dict(base, hash_version=2, actor_user_id="user-123"),
         "ec4f0361affac4711a49da38208bf68bdd854b484fd9f36c9ee5e925b075cf2c"),
        (dict(base, hash_version=2, actor_user_id=None),
         "bc62abd7a0130c6135a863783c0d29c55a8030a7efd782ce22fa60aa407376dc"),
    ]
    good = True
    for row, want in cases:
        got = row_hash(row)
        print("%s v%d: %s" % ("ok  " if got == want else "FAIL", row["hash_version"], got))
        good = good and got == want
    return good


def main(argv):
    if len(argv) != 2:
        print(__doc__.strip())
        return 2
    if argv[1] == "--self-test":
        return 0 if self_test() else 1
    return 0 if check_bundle(argv[1]) else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
