#!/usr/bin/env bash
set -euo pipefail

MODE="${MODE:-fixed}"
MATRIX_ROOT="${MATRIX_ROOT:-$HOME/socioty-results/tss-lib/multisign-matrix}"
RUN_TAG="${RUN_TAG:-$(date +%Y%m%d-%H%M%S)}"
MATRIX_OUT_DIR="$MATRIX_ROOT/multisign-matrix-${MODE}-${RUN_TAG}"
SUMMARY="$MATRIX_OUT_DIR/multisign_matrix_summary.csv"

mkdir -p "$MATRIX_OUT_DIR"

CONFIGS=(
  "3 2"
  "5 3"
)

SIG_COUNTS=(
  "1"
  "5"
  "10"
)

echo "RUN_TAG=$RUN_TAG"
echo "MODE=$MODE"
echo "MATRIX_OUT_DIR=$MATRIX_OUT_DIR"
echo

first=1

for cfg in "${CONFIGS[@]}"; do
  read -r N T <<< "$cfg"

  for SIGS in "${SIG_COUNTS[@]}"; do
    echo
    echo "============================================================"
    echo "Running tss-lib multisign config n=$N t=$T sigs=$SIGS mode=$MODE"
    echo "============================================================"

    CFG_TAG="${RUN_TAG}-n${N}-t${T}-sigs${SIGS}"
    LOG="$MATRIX_OUT_DIR/run_n${N}_t${T}_sigs${SIGS}_${MODE}.log"

    MODE="$MODE" RUN_TAG="$CFG_TAG" ./scripts/jetstream/run_tss_multisign.sh "$N" "$T" "$SIGS" | tee "$LOG"

    OUT_DIR="$(grep 'MULTISIGN_VM_OK OUT_DIR=' "$LOG" | tail -n 1 | sed 's/^MULTISIGN_VM_OK OUT_DIR=//')"

    if [ -z "$OUT_DIR" ]; then
      echo "ERROR: could not find multisign output dir for n=$N t=$T sigs=$SIGS"
      exit 1
    fi

    ROW_FILE="$OUT_DIR/multisign_summary.csv"

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
done

echo
echo "MULTISIGN_MATRIX_OK OUT_DIR=$MATRIX_OUT_DIR"
echo "SUMMARY=$SUMMARY"
cat "$SUMMARY"
