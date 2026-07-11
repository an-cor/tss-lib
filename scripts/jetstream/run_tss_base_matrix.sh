#!/usr/bin/env bash
set -euo pipefail

MATRIX_ROOT="${MATRIX_ROOT:-$HOME/socioty-results/tss-lib/base-matrix}"
RUN_TAG="${RUN_TAG:-$(date +%Y%m%d-%H%M%S)}"
MATRIX_OUT_DIR="$MATRIX_ROOT/base-matrix-$RUN_TAG"
SUMMARY="$MATRIX_OUT_DIR/base_matrix_summary.csv"

mkdir -p "$MATRIX_OUT_DIR"

CONFIGS=(
  "3 2"
  "5 2"
  "5 3"
  "10 5"
  "10 6"
  "10 7"
  "10 8"
)

echo "RUN_TAG=$RUN_TAG"
echo "MATRIX_OUT_DIR=$MATRIX_OUT_DIR"
echo

first=1

for cfg in "${CONFIGS[@]}"; do
  read -r N T <<< "$cfg"

  echo
  echo "============================================================"
  echo "Running tss-lib base config n=$N t=$T"
  echo "============================================================"

  CFG_TAG="${RUN_TAG}-n${N}-t${T}"
  LOG="$MATRIX_OUT_DIR/run_n${N}_t${T}.log"

  RUN_TAG="$CFG_TAG" ./scripts/jetstream/run_tss_keygen_sign.sh "$N" "$T" | tee "$LOG"

  COMBINED_OUT_DIR="$(grep 'KEYGEN_SIGN_VM_OK OUT_DIR=' "$LOG" | tail -n 1 | sed 's/^KEYGEN_SIGN_VM_OK OUT_DIR=//')"

  if [ -z "$COMBINED_OUT_DIR" ]; then
    echo "ERROR: could not find combined output dir for n=$N t=$T"
    exit 1
  fi

  ROW_FILE="$COMBINED_OUT_DIR/keygen_sign_summary.csv"

  if [ ! -f "$ROW_FILE" ]; then
    echo "ERROR: missing $ROW_FILE"
    exit 1
  fi

  if [ "$first" -eq 1 ]; then
    cat "$ROW_FILE" > "$SUMMARY"
    first=0
  else
    tail -n +2 "$ROW_FILE" >> "$SUMMARY"
  fi

  echo "Recorded row from $ROW_FILE"
done

echo
echo "BASE_MATRIX_OK OUT_DIR=$MATRIX_OUT_DIR"
echo "SUMMARY=$SUMMARY"
cat "$SUMMARY"
