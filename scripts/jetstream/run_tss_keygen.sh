#!/usr/bin/env bash
set -euo pipefail

CONFIG="${CONFIG:-scripts/jetstream/parties.json}"
N="${1:-3}"
T="${2:-2}"
RELAY_PORT="${RELAY_PORT:-19102}"
TIMEOUT="${TIMEOUT:-300s}"
START_DELAY="${START_DELAY:-10s}"
RUN_ID="${RUN_ID:-vm-keygen-$(date +%Y%m%d-%H%M%S)}"
RESULT_ROOT="${RESULT_ROOT:-$HOME/socioty-results/tss-lib/keygen-vm}"
OUT_DIR="$RESULT_ROOT/$RUN_ID"
REMOTE_ROOT="/home/exouser/socioty-results/tss-lib/keygen-vm/$RUN_ID"

mkdir -p "$OUT_DIR"

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
echo "RELAY=$RELAY"
echo "TIMEOUT=$TIMEOUT"
echo "START_DELAY=$START_DELAY"
echo

echo "== building binaries on controller =="
go build -o tss_relay ./cmd/tssbench/relay
go build -o tss_party ./cmd/tssbench/party
ls -lh tss_relay tss_party
echo

echo "== selected parties =="
get_party_rows | tee "$OUT_DIR/parties.txt"
echo

echo "== copying tss_party to selected parties =="
while read -r id ip user name; do
  echo "-- copy party_id=$id name=$name ip=$ip"
  scp -q -i "$SSH_KEY" ./tss_party "$user@$ip:/home/$user/tss_party"
  ssh -n -i "$SSH_KEY" "$user@$ip" "chmod +x ~/tss_party && mkdir -p '$REMOTE_ROOT/party${id}' && df -h / | tail -n 1"
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

echo "== starting keygen parties =="
party_pids=()

while read -r id ip user name; do
  echo "-- start keygen party_id=$id ip=$ip"
  ssh -n -i "$SSH_KEY" "$user@$ip" \
    "mkdir -p '$REMOTE_ROOT/party${id}' && ~/tss_party -mode keygen -id $id -n $N -t $T -relay '$RELAY' -run '$RUN_ID' -out-dir '$REMOTE_ROOT/party${id}' -timeout '$TIMEOUT' -start-delay '$START_DELAY'" \
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

  if ! grep -q "KEYGEN_OK id=${id}" "$OUT_DIR/party${id}.log"; then
    echo "ERROR: missing KEYGEN_OK for party $id"
    status=1
  fi
done < "$OUT_DIR/parties.txt"

echo "== collecting remote artifacts =="
mkdir -p "$OUT_DIR/artifacts"

while read -r id ip user name; do
  scp -q -r -i "$SSH_KEY" "$user@$ip:$REMOTE_ROOT/party${id}" "$OUT_DIR/artifacts/"
done < "$OUT_DIR/parties.txt"

echo "== collected files =="
find "$OUT_DIR" -maxdepth 4 -type f | sort

if [ "$status" -ne 0 ]; then
  echo "KEYGEN_VM_FAILED OUT_DIR=$OUT_DIR"
  exit "$status"
fi

echo "KEYGEN_VM_OK OUT_DIR=$OUT_DIR"
