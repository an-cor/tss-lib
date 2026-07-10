#!/usr/bin/env bash
set -euo pipefail

# Cross-VM smoke test for Binance tss-lib Jetstream work.
#
# Usage:
#   ./scripts/jetstream/run_tss_smoke.sh
#   ./scripts/jetstream/run_tss_smoke.sh 3
#
# What it does:
# - Builds tss_relay and tss_party on the controller.
# - Copies tss_party to the first N party VMs.
# - Starts tss_relay on the controller.
# - Starts parties 2..N as receivers.
# - Starts party 1 as sender.
# - Verifies relay broadcast routing by checking PARTY_OK in logs.

CONFIG="${CONFIG:-scripts/jetstream/parties.json}"
N="${1:-3}"
RELAY_PORT="${RELAY_PORT:-19100}"
RUN_ID="${RUN_ID:-vm-smoke-$(date +%Y%m%d-%H%M%S)}"
RESULT_ROOT="${RESULT_ROOT:-$HOME/socioty-results/tss-lib/smoke}"
OUT_DIR="$RESULT_ROOT/$RUN_ID"

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
echo "RELAY=$RELAY"
echo "SSH_KEY=$SSH_KEY"
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
  ssh -n -i "$SSH_KEY" "$user@$ip" "chmod +x ~/tss_party && ls -lh ~/tss_party && df -h / | tail -n 1"
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

echo "RELAY_PID=$RELAY_PID"
sleep 1

if ! ss -ltnp | grep -q "$RELAY_PORT"; then
  echo "ERROR: relay did not start on port $RELAY_PORT"
  cat "$OUT_DIR/relay.log" || true
  exit 1
fi

ss -ltnp | grep "$RELAY_PORT" || true
echo

echo "== starting receiver parties =="
while read -r id ip user name; do
  if [ "$id" = "1" ]; then
    continue
  fi

  echo "-- start receiver party_id=$id ip=$ip"
  ssh -n -i "$SSH_KEY" "$user@$ip" \
    "nohup ~/tss_party -id $id -n $N -relay $RELAY -run $RUN_ID -expect 1 -timeout 30s > /tmp/${RUN_ID}-party${id}.log 2>&1 &"
done < "$OUT_DIR/parties.txt"

sleep 2
echo

echo "== starting sender party 1 =="
party1_line="$(head -n 1 "$OUT_DIR/parties.txt")"
party1_id="$(echo "$party1_line" | awk '{print $1}')"
party1_ip="$(echo "$party1_line" | awk '{print $2}')"
party1_user="$(echo "$party1_line" | awk '{print $3}')"

ssh -i "$SSH_KEY" "$party1_user@$party1_ip" \
  "~/tss_party -id $party1_id -n $N -relay $RELAY -run $RUN_ID -send -to 0 -expect 0 -timeout 30s > /tmp/${RUN_ID}-party${party1_id}.log 2>&1"

sleep 3
echo

echo "== collecting logs =="
while read -r id ip user name; do
  scp -q -i "$SSH_KEY" "$user@$ip:/tmp/${RUN_ID}-party${id}.log" "$OUT_DIR/party${id}.log"
done < "$OUT_DIR/parties.txt"

kill "$RELAY_PID" 2>/dev/null || true

echo "== relay log =="
cat "$OUT_DIR/relay.log"
echo

status=0

while read -r id ip user name; do
  echo "== party $id log =="
  cat "$OUT_DIR/party${id}.log"
  echo

  if ! grep -q "PARTY_OK id=${id}" "$OUT_DIR/party${id}.log"; then
    echo "ERROR: missing PARTY_OK for party $id"
    status=1
  fi
done < "$OUT_DIR/parties.txt"

if [ "$status" -ne 0 ]; then
  echo "SMOKE_FAILED OUT_DIR=$OUT_DIR"
  exit "$status"
fi

echo "SMOKE_OK OUT_DIR=$OUT_DIR"
