#!/usr/bin/env bash
set -euo pipefail

N="${1:-3}"
T="${2:-2}"
SIGNERS="${SIGNERS:-$((T + 1))}"
SIGNER_IDS="${SIGNER_IDS:-}"
KEYGEN_RUN="${KEYGEN_RUN:-}"
RUN_ID="${RUN_ID:-vm-sign-$(date +%Y%m%d-%H%M%S)}"
CONFIG="${CONFIG:-scripts/jetstream/parties.json}"
RELAY_PORT="${RELAY_PORT:-19104}"
TIMEOUT="${TIMEOUT:-180s}"
START_DELAY="${START_DELAY:-10s}"
MSG="${MSG:-42}"

OUT_ROOT="${OUT_ROOT:-$HOME/socioty-results/tss-lib/sign-vm}"
OUT_DIR="$OUT_ROOT/$RUN_ID"
REMOTE_ROOT="/home/exouser/socioty-results/tss-lib/sign-vm/$RUN_ID"

mkdir -p "$OUT_DIR"

if [ -z "$SIGNER_IDS" ]; then
  SIGNER_IDS="$(python3 - "$SIGNERS" <<'PY'
import sys
count = int(sys.argv[1])
print(",".join(str(i) for i in range(1, count + 1)))
PY
)"
else
  SIGNERS="$(python3 - "$SIGNER_IDS" <<'PY'
import sys
ids = [x.strip() for x in sys.argv[1].split(",") if x.strip()]
print(len(ids))
PY
)"
fi

echo "$SIGNER_IDS" > "$OUT_DIR/signer_ids.txt"

if [ -z "$KEYGEN_RUN" ]; then
  KEYGEN_RUN="$(ls -td "$HOME"/socioty-results/tss-lib/keygen-vm/vm-keygen-* 2>/dev/null | head -n 1 | xargs -r basename)"
fi

if [ -z "$KEYGEN_RUN" ]; then
  echo "ERROR: KEYGEN_RUN is empty and no prior keygen run was found"
  exit 1
fi

REMOTE_KEYGEN_ROOT="/home/exouser/socioty-results/tss-lib/keygen-vm/$KEYGEN_RUN"

RELAY_IP="$(python3 - "$CONFIG" <<'PY'
import json, sys
cfg = json.load(open(sys.argv[1]))
print(cfg["controller"]["internal_ip"])
PY
)"

SSH_KEY="$(python3 - "$CONFIG" <<'PY'
import json, os, sys
cfg = json.load(open(sys.argv[1]))
print(os.path.expanduser(cfg.get("ssh_key") or cfg.get("controller", {}).get("ssh_key") or cfg.get("controller", {}).get("ssh_private_key") or "~/.ssh/socioty_controller_key"))
PY
)"

RELAY="$RELAY_IP:$RELAY_PORT"

get_party_rows() {
  local ids_csv="$1"
  python3 -c '
import json, sys
cfg = json.load(open(sys.argv[1]))
ids = [int(x.strip()) for x in sys.argv[2].split(",") if x.strip()]
by_id = {int(p["id"]): p for p in cfg["parties"]}
for party_id in ids:
    if party_id not in by_id:
        raise SystemExit(f"party id {party_id} not found in config")
    p = by_id[party_id]
    print("{} {} {} {}".format(p["id"], p["internal_ip"], p["user"], p["name"]))
' "$CONFIG" "$ids_csv"
}

echo "RUN_ID=$RUN_ID"
echo "OUT_DIR=$OUT_DIR"
echo "CONFIG=$CONFIG"
echo "N=$N"
echo "T=$T"
echo "SIGNERS=$SIGNERS"
echo "SIGNER_IDS=$SIGNER_IDS"
echo "KEYGEN_RUN=$KEYGEN_RUN"
echo "RELAY=$RELAY"
echo "TIMEOUT=$TIMEOUT"
echo "START_DELAY=$START_DELAY"
echo "MSG=$MSG"
echo

echo "== building binaries on controller =="
go build -o tss_relay ./cmd/tssbench/relay
go build -o tss_party ./cmd/tssbench/party
ls -lh tss_relay tss_party

echo
echo "== selected parties =="
get_party_rows "$SIGNER_IDS" | tee "$OUT_DIR/parties.txt"

echo
echo "== copying tss_party and checking keygen shares =="
while read -r id ip user name; do
  printf -v id2 "%02d" "$id"
  save_path="$REMOTE_KEYGEN_ROOT/party${id}/keygen_save_party_${id2}.json"

  echo "-- party_id=$id name=$name ip=$ip"
  scp -q -i "$SSH_KEY" -o StrictHostKeyChecking=no ./tss_party "$user@$ip:~/tss_party"

  ssh -n -i "$SSH_KEY" -o StrictHostKeyChecking=no "$user@$ip" \
    "test -f '$save_path' && echo save_ok='$save_path' && df -h / | tail -n 1"
done < "$OUT_DIR/parties.txt"

echo
echo "== cleaning old processes =="
pkill -f "[t]ss_relay.*${RELAY_PORT}" || true

while read -r id ip user name; do
  ssh -n -i "$SSH_KEY" -o StrictHostKeyChecking=no "$user@$ip" \
    "pkill -f '[t]ss_party' || true"
done < "$OUT_DIR/parties.txt"

sleep 1

echo
echo "== starting relay =="
./tss_relay -listen "0.0.0.0:$RELAY_PORT" -run "$RUN_ID" > "$OUT_DIR/relay.log" 2>&1 &
RELAY_PID=$!
echo "RELAY_PID=$RELAY_PID"

sleep 1
ss -ltnp | grep ":$RELAY_PORT" || true

cleanup() {
  pkill -f "[t]ss_relay.*${RELAY_PORT}" || true
}
trap cleanup EXIT

echo
echo "== starting signing parties =="

party_pids=()

while read -r id ip user name; do
  printf -v id2 "%02d" "$id"
  save_path="$REMOTE_KEYGEN_ROOT/party${id}/keygen_save_party_${id2}.json"
  remote_out="$REMOTE_ROOT/party${id}"

  echo "-- start sign party_id=$id ip=$ip"

  ssh -n -i "$SSH_KEY" -o StrictHostKeyChecking=no "$user@$ip" "
    mkdir -p '$remote_out'
    ~/tss_party \
      -mode sign \
      -id '$id' \
      -n '$N' \
      -t '$T' \
      -signers '$SIGNERS' \
      -signer-ids '$SIGNER_IDS' \
      -relay '$RELAY' \
      -run '$RUN_ID' \
      -keygen-save '$save_path' \
      -msg '$MSG' \
      -out-dir '$remote_out' \
      -timeout '$TIMEOUT' \
      -start-delay '$START_DELAY'
  " > "$OUT_DIR/party${id}.log" 2>&1 &

  party_pids+=("$!")
done < "$OUT_DIR/parties.txt"

status=0
for pid in "${party_pids[@]}"; do
  if ! wait "$pid"; then
    status=1
  fi
done

echo
echo "== relay tail =="
tail -n 80 "$OUT_DIR/relay.log" || true

echo
echo "== party logs =="
while read -r id ip user name; do
  echo "== party $id tail =="
  tail -n 80 "$OUT_DIR/party${id}.log" || true
  echo
done < "$OUT_DIR/parties.txt"

echo "== collecting remote artifacts =="
mkdir -p "$OUT_DIR/artifacts"

while read -r id ip user name; do
  mkdir -p "$OUT_DIR/artifacts/party${id}"
  scp -q -i "$SSH_KEY" -o StrictHostKeyChecking=no -r \
    "$user@$ip:$REMOTE_ROOT/party${id}/." \
    "$OUT_DIR/artifacts/party${id}/" || true
done < "$OUT_DIR/parties.txt"

echo "== collected files =="
find "$OUT_DIR" -maxdepth 4 -type f | sort

if [ "$status" -ne 0 ]; then
  echo "ERROR: one or more signing parties failed"
  exit "$status"
fi

ok_count="$(grep -h "SIGN_OK" "$OUT_DIR"/party*.log | wc -l | tr -d ' ')"

if [ "$ok_count" != "$SIGNERS" ]; then
  echo "ERROR: expected $SIGNERS SIGN_OK lines, found $ok_count"
  exit 1
fi

if ! grep -R '"verify_ok": true' "$OUT_DIR/artifacts" >/dev/null 2>&1; then
  echo "ERROR: verify_ok=true not found in collected signature summaries"
  exit 1
fi

echo "SIGN_VM_OK OUT_DIR=$OUT_DIR"
