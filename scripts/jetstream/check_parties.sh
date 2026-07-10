#!/usr/bin/env bash
set -euo pipefail

CONFIG="${CONFIG:-scripts/jetstream/parties.json}"
COUNT="${1:-10}"

SSH_KEY="$(python3 - "$CONFIG" <<'PY'
import json
import os
import sys

with open(sys.argv[1]) as f:
    cfg = json.load(f)

print(os.path.expanduser(cfg["ssh_key"]))
PY
)"

python3 - "$CONFIG" "$COUNT" > /tmp/tss_party_ips.txt <<'PY'
import json
import sys

config_path = sys.argv[1]
count = int(sys.argv[2])

with open(config_path) as f:
    cfg = json.load(f)

for p in cfg["parties"][:count]:
    print(p["id"], p["name"], p["internal_ip"], p["user"])
PY

echo "Using config: $CONFIG"
echo "Using SSH key: $SSH_KEY"
echo "Checking first $COUNT party VM(s)"
echo

while read -r id name ip user; do
  echo "== party $id: $name / $ip =="

  ssh -n -i "$SSH_KEY" \
    -o BatchMode=yes \
    -o ConnectTimeout=5 \
    -o StrictHostKeyChecking=accept-new \
    "$user@$ip" "
      set -e
      echo hostname=\$(hostname)
      echo user=\$(whoami)
      echo internal_ip=\$(hostname -I | tr ' ' '\n' | grep '^10\.1\.20\.' | head -n 1)
      echo disk=\$(df -h / | tail -n 1 | tr -s ' ')
      echo go=\$(go version 2>/dev/null || echo missing)
    "

  echo
done < /tmp/tss_party_ips.txt

echo "Party SSH check completed."
