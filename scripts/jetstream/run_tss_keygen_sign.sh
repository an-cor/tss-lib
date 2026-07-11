#!/usr/bin/env bash
set -euo pipefail

N="${1:-3}"
T="${2:-2}"
SIGNERS="${SIGNERS:-$((T + 1))}"
RUN_TAG="${RUN_TAG:-$(date +%Y%m%d-%H%M%S)}"
KEYGEN_PORT="${KEYGEN_PORT:-19112}"
SIGN_PORT="${SIGN_PORT:-19114}"
MSG="${MSG:-42}"

COMBINED_ROOT="${COMBINED_ROOT:-$HOME/socioty-results/tss-lib/keygen-sign-vm}"
COMBINED_OUT_DIR="$COMBINED_ROOT/vm-keygen-sign-$RUN_TAG"

mkdir -p "$COMBINED_OUT_DIR"

KEYGEN_LOG="$COMBINED_OUT_DIR/keygen_driver.log"
SIGN_LOG="$COMBINED_OUT_DIR/sign_driver.log"

echo "RUN_TAG=$RUN_TAG"
echo "N=$N"
echo "T=$T"
echo "SIGNERS=$SIGNERS"
echo "MSG=$MSG"
echo "COMBINED_OUT_DIR=$COMBINED_OUT_DIR"
echo "KEYGEN_PORT=$KEYGEN_PORT"
echo "SIGN_PORT=$SIGN_PORT"
echo

echo "== running keygen =="
RUN_ID="vm-keygen-${RUN_TAG}" \
RELAY_PORT="$KEYGEN_PORT" \
./scripts/jetstream/run_tss_keygen.sh "$N" "$T" | tee "$KEYGEN_LOG"

KEYGEN_OUT_DIR="$(grep 'KEYGEN_VM_OK OUT_DIR=' "$KEYGEN_LOG" | tail -n 1 | sed 's/^KEYGEN_VM_OK OUT_DIR=//')"

if [ -z "$KEYGEN_OUT_DIR" ]; then
  echo "ERROR: could not determine KEYGEN_OUT_DIR"
  exit 1
fi

KEYGEN_RUN="$(basename "$KEYGEN_OUT_DIR")"

echo
echo "KEYGEN_RUN=$KEYGEN_RUN"
echo "KEYGEN_OUT_DIR=$KEYGEN_OUT_DIR"

python3 scripts/analysis/export_tss_keygen_csv.py "$KEYGEN_OUT_DIR" | tee "$KEYGEN_OUT_DIR/keygen_summary.csv"
cp "$KEYGEN_OUT_DIR/keygen_summary.csv" "$COMBINED_OUT_DIR/keygen_summary.csv"

echo
echo "== running signing =="
KEYGEN_RUN="$KEYGEN_RUN" \
RUN_ID="vm-sign-${RUN_TAG}" \
SIGNERS="$SIGNERS" \
RELAY_PORT="$SIGN_PORT" \
MSG="$MSG" \
./scripts/jetstream/run_tss_sign_round.sh "$N" "$T" | tee "$SIGN_LOG"

SIGN_OUT_DIR="$(grep 'SIGN_VM_OK OUT_DIR=' "$SIGN_LOG" | tail -n 1 | sed 's/^SIGN_VM_OK OUT_DIR=//')"

if [ -z "$SIGN_OUT_DIR" ]; then
  echo "ERROR: could not determine SIGN_OUT_DIR"
  exit 1
fi

echo
echo "SIGN_OUT_DIR=$SIGN_OUT_DIR"

python3 scripts/analysis/export_tss_sign_csv.py "$SIGN_OUT_DIR" | tee "$SIGN_OUT_DIR/sign_summary.csv"
cp "$SIGN_OUT_DIR/sign_summary.csv" "$COMBINED_OUT_DIR/sign_summary.csv"

echo
echo "== writing combined summary =="
python3 - "$COMBINED_OUT_DIR" "$KEYGEN_OUT_DIR" "$SIGN_OUT_DIR" <<'PY'
import csv
import sys
from pathlib import Path

combined_dir = Path(sys.argv[1])
keygen_dir = Path(sys.argv[2])
sign_dir = Path(sys.argv[3])

def read_one(path):
    with open(path, newline="") as f:
        rows = list(csv.DictReader(f))
    if len(rows) != 1:
        raise SystemExit(f"expected one row in {path}, got {len(rows)}")
    return rows[0]

k = read_one(combined_dir / "keygen_summary.csv")
s = read_one(combined_dir / "sign_summary.csv")

keygen_ms = int(k["keygen_time_ms"])
sign_ms = int(s["sign_time_ms"])

row = {
    "protocol": "tss-lib",
    "n": k["n"],
    "t": k["t"],
    "signer_count": s.get("party_count", ""),
    "keygen_run_id": k["run_id"],
    "sign_run_id": s["run_id"],
    "keygen_time_ms": keygen_ms,
    "sign_time_ms": sign_ms,
    "total_keygen_plus_sign_ms": keygen_ms + sign_ms,
    "keygen_sent_messages": k["total_sent_messages"],
    "sign_sent_messages": s["total_sent_messages"],
    "keygen_sent_bytes": k["total_sent_bytes"],
    "sign_sent_bytes": s["total_sent_bytes"],
    "sign_verify_all": s["verify_all"],
    "keygen_result_dir": str(keygen_dir),
    "sign_result_dir": str(sign_dir),
    "combined_result_dir": str(combined_dir),
}

out_path = combined_dir / "keygen_sign_summary.csv"
with open(out_path, "w", newline="") as f:
    writer = csv.DictWriter(f, fieldnames=list(row.keys()))
    writer.writeheader()
    writer.writerow(row)

print(out_path)
PY

cat "$COMBINED_OUT_DIR/keygen_sign_summary.csv"

echo
echo "KEYGEN_SIGN_VM_OK OUT_DIR=$COMBINED_OUT_DIR"
