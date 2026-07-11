#!/usr/bin/env bash
set -euo pipefail

CONFIG="${CONFIG:-scripts/jetstream/parties.json}"
N="${1:-3}"
T="${2:-2}"
KEYGEN_RUN="${KEYGEN_RUN:-}"
RELAY_PORT="${RELAY_PORT:-19104}"
TIMEOUT="${TIMEOUT:-180s}"
START_DELAY="${START_DELAY:-10s}"
MSG="${MSG:-42}"
RUN_ID="${RUN_ID:-vm-sign-$(date +%Y%m%d-%H%M%S)}"
RESULT_ROOT="${RESULT_ROOT:-$HOME/socioty-results/tss-lib/sign-vm}"
OUT_DIR="$RESULT_ROOT/$RUN_ID"
REMOTE_ROOT="/home/exouser/socioty-results/tss-lib/sign-vm/$RUN_ID"

mkdir -p "$OUT_DIR"

if [ -z "$KEYGEN_RUN" ]; then
  KEYGEN_RUN="$(basename "$(ls -td "$HOME"/socioty-results/tss-lib/keygen-vm/vm-keygen-* | head -n 1)")"
fi

REMOTE_KEYGEN_ROOT="/home/exouser/socioty-results/tss-lib/keygen-vm/$KEYGEN_RUN"

get_controller_internal_ip() {
  python3 -c 'import json,sys; cfg=json.load(open(sys.argv[1])); print(cfg["controller"]["internal_ip"])' "$CONFIG"
}

get_ssh_key() {
  python3 -c 'import json,os,sys; cfg=json.load(open(sys.argv[1])); print(os.path.expanduser(cfg["ssh_key"]))' "$CONFIG"
}

get_party_rows() {
  python3 -c '
import json, sys
cfg = json.load(open(sys.argv[1]))
n = int(sys.argv[2])
for p in cfg["parties"][:n]:
    print("{} {} {} {}".format(p["id"], p["internal_ip"], p["user"], p["name"]))
' "$CONFIG" "$N"
}

RELAY_IP="$(get_controller_internal_ip)"
SSH_KEY="$(get_ssh_key)"
RELAY="${RELAY_IP}:${RELAY_PORT}"

echo "RUN_ID=$RUN_ID"
echo "OUT_DIR=$OUT_DIR"
echo "CONFIG=$CONFIG"
echo "N=$N"
echo "T=$T"
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
get_party_rows | tee "$OUT_DIR/parties.txt"
echo

echo "== copying tss_party and checking keygen shares =="
while read -r id ip user name; do
  save_path="$REMOTE_KEYGEN_ROOT/party${id}/keygen_save_party_$(printf "%02d" "$id").json"

  echo "-- party_id=$id name=$name ip=$ip"
  scp -q -i "$SSH_KEY" ./tss_party "$user@$ip:/home/$user/tss_party"

  ssh -n -i "$SSH_KEY" "$user@$ip" "
    chmod +x ~/tss_party
    test -f '$save_path'
    mkdir -p '$REMOTE_ROOT/party${id}'
    echo save_ok=$save_path
    df -h / | tail -n 1
  "
done < "$OUT_DIR/parties.txt"
echo

echo "== cleaning old processes =="
pkill -f "[t]ss_relay.*${RELAY_PORT}" 2>/dev/null || true

while read -r id ip user name; do
  ssh -n -i "$SSH_KEY" "$user@$ip" "pkill -f [t]ss_party 2>/dev/null || true"
done < "$OUT_DIR/parties.txt"

sleep 1
echo

echo "== starting relay =="
./tss_relay -listen "0.0.0.0:${RELAY_PORT}" -run "$RUN_ID" > "$OUT_DIR/relay.log" 2>&1 &
RELAY_PID=$!

sleep 1

if ! ss -ltnp | grep -q "$RELAY_PORT"; then
  echo "ERROR: relay did not start on port $RELAY_PORT"
  cat "$OUT_DIR/relay.log" || true
  exit 1
fi

echo "RELAY_PID=$RELAY_PID"
ss -ltnp | grep "$RELAY_PORT" || true
echo

echo "== starting signing parties =="
party_pids=()

while read -r id ip user name; do
  save_path="$REMOTE_KEYGEN_ROOT/party${id}/keygen_save_party_$(printf "%02d" "$id").json"

  echo "-- start sign party_id=$id ip=$ip"
  ssh -n -i "$SSH_KEY" "$user@$ip" \
    "mkdir -p '$REMOTE_ROOT/party${id}' && ~/tss_party -mode sign -id $id -n $N -t $T -relay '$RELAY' -run '$RUN_ID' -keygen-save '$save_path' -msg '$MSG' -out-dir '$REMOTE_ROOT/party${id}' -timeout '$TIMEOUT' -start-delay '$START_DELAY'" \
    > "$OUT_DIR/party${id}.log" 2>&1 &
  party_pids+=("$!")
done < "$OUT_DIR/parties.txt"

status=0
for pid in "${party_pids[@]}"; do
  if ! wait "$pid"; then
    status=1
  fi
done

kill "$RELAY_PID" 2>/dev/null || true

echo
echo "== relay tail =="
tail -n 120 "$OUT_DIR/relay.log"
echo

echo "== party logs =="
while read -r id ip user name; do
  echo "== party $id tail =="
  tail -n 100 "$OUT_DIR/party${id}.log"
  echo

  if ! grep -q "SIGN_OK id=${id}" "$OUT_DIR/party${id}.log"; then
    echo "ERROR: missing SIGN_OK for party $id"
    status=1
  fi
done < "$OUT_DIR/parties.txt"

if ! grep -R '"verify_ok": true' "$OUT_DIR" >/dev/null 2>&1; then
  echo "WARNING: verify_ok=true not found in local controller logs before artifact collection"
fi

echo "== collecting remote artifacts =="
mkdir -p "$OUT_DIR/artifacts"

while read -r id ip user name; do
  scp -q -r -i "$SSH_KEY" "$user@$ip:$REMOTE_ROOT/party${id}" "$OUT_DIR/artifacts/"
done < "$OUT_DIR/parties.txt"

echo "== collected files =="
find "$OUT_DIR" -maxdepth 4 -type f | sort

while read -r id ip user name; do
  if ! grep -q '"verify_ok": true' "$OUT_DIR/artifacts/party${id}/signature_summary_party_$(printf "%02d" "$id").json"; then
    echo "ERROR: verify_ok=true missing for party $id"
    status=1
  fi
done < "$OUT_DIR/parties.txt"

if [ "$status" -ne 0 ]; then
  echo "SIGN_VM_FAILED OUT_DIR=$OUT_DIR"
  exit "$status"
fi

echo "SIGN_VM_OK OUT_DIR=$OUT_DIR"
