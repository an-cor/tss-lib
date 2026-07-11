#!/usr/bin/env bash
set -euo pipefail

TRIALS="${TRIALS:-2}"
ROOT="${ROOT:-$HOME/socioty-results/tss-lib/repeat-trials}"
RUN_TAG="${RUN_TAG:-rss-repeat-$(date +%Y%m%d-%H%M%S)}"
OUT_DIR="$ROOT/$RUN_TAG"

mkdir -p "$OUT_DIR"

echo "RUN_TAG=$RUN_TAG"
echo "TRIALS=$TRIALS"
echo "OUT_DIR=$OUT_DIR"
echo

for trial in $(seq 1 "$TRIALS"); do
  printf -v trial_id "%02d" "$trial"

  echo
  echo "============================================================"
  echo "RSS repeat trial $trial_id / $TRIALS: base matrix"
  echo "============================================================"

  BASE_LOG="$OUT_DIR/base_trial_${trial_id}.log"
  RUN_TAG="${RUN_TAG}-trial${trial_id}-base" \
    ./scripts/jetstream/run_tss_base_matrix.sh | tee "$BASE_LOG"

  BASE_OUT="$(grep 'BASE_MATRIX_OK OUT_DIR=' "$BASE_LOG" | tail -n 1 | sed 's/^BASE_MATRIX_OK OUT_DIR=//')"

  if [ -z "$BASE_OUT" ]; then
    echo "ERROR: missing BASE_MATRIX_OK for trial $trial_id"
    exit 1
  fi

  python3 scripts/analysis/add_rss_columns.py "$BASE_OUT/base_matrix_summary.csv" \
    > "$BASE_OUT/base_matrix_summary.with_rss.csv"

  echo "$BASE_OUT" >> "$OUT_DIR/base_matrix_dirs.txt"

  echo
  echo "============================================================"
  echo "RSS repeat trial $trial_id / $TRIALS: fixed multisign matrix"
  echo "============================================================"

  FIXED_LOG="$OUT_DIR/fixed_trial_${trial_id}.log"
  MODE=fixed RUN_TAG="${RUN_TAG}-trial${trial_id}-fixed" \
    ./scripts/jetstream/run_tss_multisign_matrix.sh | tee "$FIXED_LOG"

  FIXED_OUT="$(grep 'MULTISIGN_MATRIX_OK OUT_DIR=' "$FIXED_LOG" | tail -n 1 | sed 's/^MULTISIGN_MATRIX_OK OUT_DIR=//')"

  if [ -z "$FIXED_OUT" ]; then
    echo "ERROR: missing MULTISIGN_MATRIX_OK fixed for trial $trial_id"
    exit 1
  fi

  python3 scripts/analysis/add_rss_columns.py "$FIXED_OUT/multisign_matrix_summary.csv" \
    > "$FIXED_OUT/multisign_matrix_summary.with_rss.csv"

  echo "$FIXED_OUT" >> "$OUT_DIR/fixed_multisign_matrix_dirs.txt"

  echo
  echo "============================================================"
  echo "RSS repeat trial $trial_id / $TRIALS: random multisign matrix"
  echo "============================================================"

  RANDOM_LOG="$OUT_DIR/random_trial_${trial_id}.log"
  MODE=random RUN_TAG="${RUN_TAG}-trial${trial_id}-random" \
    ./scripts/jetstream/run_tss_multisign_matrix.sh | tee "$RANDOM_LOG"

  RANDOM_OUT="$(grep 'MULTISIGN_MATRIX_OK OUT_DIR=' "$RANDOM_LOG" | tail -n 1 | sed 's/^MULTISIGN_MATRIX_OK OUT_DIR=//')"

  if [ -z "$RANDOM_OUT" ]; then
    echo "ERROR: missing MULTISIGN_MATRIX_OK random for trial $trial_id"
    exit 1
  fi

  python3 scripts/analysis/add_rss_columns.py "$RANDOM_OUT/multisign_matrix_summary.csv" \
    > "$RANDOM_OUT/multisign_matrix_summary.with_rss.csv"

  echo "$RANDOM_OUT" >> "$OUT_DIR/random_multisign_matrix_dirs.txt"
done

echo
echo "RSS_REPEAT_TRIALS_OK OUT_DIR=$OUT_DIR"
echo
echo "== Base matrix dirs =="
cat "$OUT_DIR/base_matrix_dirs.txt"
echo
echo "== Fixed multisign matrix dirs =="
cat "$OUT_DIR/fixed_multisign_matrix_dirs.txt"
echo
echo "== Random multisign matrix dirs =="
cat "$OUT_DIR/random_multisign_matrix_dirs.txt"
