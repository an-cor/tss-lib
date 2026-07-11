#!/usr/bin/env bash
set -euo pipefail

N="${1:-3}"
T="${2:-2}"
SIGS="${3:-5}"
MODE="${MODE:-fixed}"
SIGNERS="${SIGNERS:-$((T + 1))}"
RUN_TAG="${RUN_TAG:-$(date +%Y%m%d-%H%M%S)}"
KEYGEN_PORT="${KEYGEN_PORT:-19212}"
SIGN_PORT="${SIGN_PORT:-19214}"
MSG_BASE="${MSG_BASE:-42}"

if [ "$MODE" != "fixed" ]; then
  echo "ERROR: only MODE=fixed is supported by this script right now"
  exit 2
fi

ROOT="${ROOT:-$HOME/socioty-results/tss-lib/multisign-vm}"
OUT_DIR="$ROOT/multisign-${MODE}-${RUN_TAG}-n${N}-t${T}-sigs${SIGS}"
mkdir -p "$OUT_DIR"

KEYGEN_LOG="$OUT_DIR/keygen_driver.log"

echo "RUN_TAG=$RUN_TAG"
echo "N=$N"
echo "T=$T"
echo "SIGNERS=$SIGNERS"
echo "SIGS=$SIGS"
echo "MODE=$MODE"
echo "OUT_DIR=$OUT_DIR"
echo

echo "== running one keygen =="
RUN_ID="vm-multisign-keygen-${RUN_TAG}-n${N}-t${T}" \
RELAY_PORT="$KEYGEN_PORT" \
./scripts/jetstream/run_tss_keygen.sh "$N" "$T" | tee "$KEYGEN_LOG"

KEYGEN_OUT_DIR="$(grep 'KEYGEN_VM_OK OUT_DIR=' "$KEYGEN_LOG" | tail -n 1 | sed 's/^KEYGEN_VM_OK OUT_DIR=//')"
KEYGEN_RUN="$(basename "$KEYGEN_OUT_DIR")"

if [ -z "$KEYGEN_RUN" ]; then
  echo "ERROR: could not determine KEYGEN_RUN"
  exit 1
fi

python3 scripts/analysis/export_tss_keygen_csv.py "$KEYGEN_OUT_DIR" | tee "$KEYGEN_OUT_DIR/keygen_summary.csv"
cp "$KEYGEN_OUT_DIR/keygen_summary.csv" "$OUT_DIR/keygen_summary.csv"

echo
echo "KEYGEN_RUN=$KEYGEN_RUN"
echo "KEYGEN_OUT_DIR=$KEYGEN_OUT_DIR"
echo

SIGN_SUMMARY_ALL="$OUT_DIR/sign_rounds_summary.csv"
first=1

for i in $(seq 1 "$SIGS"); do
  ROUND="$(printf "%02d" "$i")"
  MSG="$((MSG_BASE + i))"
  SIGN_LOG="$OUT_DIR/sign_round_${ROUND}.log"

  echo
  echo "== signing round $ROUND / $SIGS msg=$MSG =="

  KEYGEN_RUN="$KEYGEN_RUN" \
  RUN_ID="vm-multisign-sign-${RUN_TAG}-n${N}-t${T}-round${ROUND}" \
  SIGNERS="$SIGNERS" \
  RELAY_PORT="$SIGN_PORT" \
  MSG="$MSG" \
  ./scripts/jetstream/run_tss_sign_round.sh "$N" "$T" | tee "$SIGN_LOG"

  SIGN_OUT_DIR="$(grep 'SIGN_VM_OK OUT_DIR=' "$SIGN_LOG" | tail -n 1 | sed 's/^SIGN_VM_OK OUT_DIR=//')"

  if [ -z "$SIGN_OUT_DIR" ]; then
    echo "ERROR: could not determine SIGN_OUT_DIR for round $ROUND"
    exit 1
  fi

  python3 scripts/analysis/export_tss_sign_csv.py "$SIGN_OUT_DIR" | tee "$SIGN_OUT_DIR/sign_summary.csv"

  if [ "$first" -eq 1 ]; then
    cat "$SIGN_OUT_DIR/sign_summary.csv" > "$SIGN_SUMMARY_ALL"
    first=0
  else
    tail -n +2 "$SIGN_OUT_DIR/sign_summary.csv" >> "$SIGN_SUMMARY_ALL"
  fi
done

echo
echo "== writing multisign summary =="
python3 - "$OUT_DIR" "$KEYGEN_OUT_DIR" "$SIGN_SUMMARY_ALL" "$MODE" "$SIGS" <<'PY'
import csv
import sys
from pathlib import Path

out_dir = Path(sys.argv[1])
keygen_dir = Path(sys.argv[2])
sign_summary_all = Path(sys.argv[3])
mode = sys.argv[4]
sigs = int(sys.argv[5])

def read_one(path):
    with open(path, newline="") as f:
        rows = list(csv.DictReader(f))
    if len(rows) != 1:
        raise SystemExit(f"expected one row in {path}, got {len(rows)}")
    return rows[0]

k = read_one(out_dir / "keygen_summary.csv")

with open(sign_summary_all, newline="") as f:
    sign_rows = list(csv.DictReader(f))

if len(sign_rows) != sigs:
    raise SystemExit(f"expected {sigs} sign rows, got {len(sign_rows)}")

sign_times = [int(r["sign_time_ms"]) for r in sign_rows]
sign_messages = [int(r["total_sent_messages"]) for r in sign_rows]
sign_bytes = [int(r["total_sent_bytes"]) for r in sign_rows]
verify_all = all(r["verify_all"] == "True" for r in sign_rows)

row = {
    "protocol": "tss-lib",
    "n": k["n"],
    "t": k["t"],
    "signer_count": sign_rows[0]["party_count"],
    "mode": mode,
    "signatures_per_keygen": sigs,
    "keygen_time_ms": k["keygen_time_ms"],
    "total_sign_time_ms": sum(sign_times),
    "avg_sign_time_ms": round(sum(sign_times) / len(sign_times), 3),
    "keygen_messages": k["total_sent_messages"],
    "sign_messages": sum(sign_messages),
    "keygen_bytes": k["total_sent_bytes"],
    "sign_bytes": sum(sign_bytes),
    "sign_verify_all": verify_all,
    "keygen_result_dir": str(keygen_dir),
    "multisign_result_dir": str(out_dir),
}

out_path = out_dir / "multisign_summary.csv"
with open(out_path, "w", newline="") as f:
    writer = csv.DictWriter(f, fieldnames=list(row.keys()))
    writer.writeheader()
    writer.writerow(row)

print(out_path)
PY

cat "$OUT_DIR/multisign_summary.csv"

echo
echo "MULTISIGN_VM_OK OUT_DIR=$OUT_DIR"
