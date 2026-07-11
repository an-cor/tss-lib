#!/usr/bin/env python3
import csv
import json
import re
import sys
from datetime import datetime
from pathlib import Path

def parse_ts(line):
    m = re.match(r"^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})", line)
    if not m:
        return None
    return datetime.strptime(m.group(1), "%Y/%m/%d %H:%M:%S")

def main():
    if len(sys.argv) != 2:
        print("usage: export_tss_sign_csv.py RUN_DIR", file=sys.stderr)
        sys.exit(2)

    run_dir = Path(sys.argv[1]).expanduser().resolve()
    summaries = sorted(run_dir.glob("artifacts/party*/signature_summary_party_*.json"))

    if not summaries:
        summaries = sorted(run_dir.glob("party*/signature_summary_party_*.json"))

    if not summaries:
        print(f"no signature summary files found under {run_dir}", file=sys.stderr)
        sys.exit(1)

    rows = [json.loads(p.read_text()) for p in summaries]

    run_id = rows[0].get("run_id", run_dir.name)
    n = int(rows[0]["n"])
    t = int(rows[0]["t"])
    msg = rows[0].get("msg", "")
    party_count = len(rows)

    sign_ms_values = [int(r.get("sign_ms", 0)) for r in rows if r.get("sign_ms") is not None]
    sign_time_ms = max(sign_ms_values) if sign_ms_values else ""

    total_sent_messages = sum(int(r.get("sent_messages", 0)) for r in rows)
    total_sent_bytes = sum(int(r.get("sent_bytes", 0)) for r in rows)
    total_received_messages = sum(int(r.get("received_messages", 0)) for r in rows)
    total_received_bytes = sum(int(r.get("received_bytes", 0)) for r in rows)

    verify_all = all(bool(r.get("verify_ok")) for r in rows)

    relay_log = run_dir / "relay.log"
    relay_tss_messages = 0
    relay_payload_bytes_base64 = 0
    timestamps = []

    if relay_log.exists():
        for line in relay_log.read_text(errors="replace").splitlines():
            ts = parse_ts(line)
            if ts:
                timestamps.append(ts)
            if " type=tss " in line:
                relay_tss_messages += 1
                m = re.search(r"payload_bytes=(\d+)", line)
                if m:
                    relay_payload_bytes_base64 += int(m.group(1))

    controller_wall_ms = ""
    if len(timestamps) >= 2:
        controller_wall_ms = int((max(timestamps) - min(timestamps)).total_seconds() * 1000)

    out = csv.DictWriter(sys.stdout, fieldnames=[
        "run_id",
        "protocol",
        "n",
        "t",
        "party_count",
        "msg",
        "sign_time_ms",
        "verify_all",
        "total_sent_messages",
        "total_sent_bytes",
        "total_received_messages",
        "total_received_bytes",
        "relay_tss_messages",
        "relay_payload_bytes_base64",
        "controller_wall_ms",
        "result_dir",
    ])
    out.writeheader()
    out.writerow({
        "run_id": run_id,
        "protocol": "tss-lib",
        "n": n,
        "t": t,
        "party_count": party_count,
        "msg": msg,
        "sign_time_ms": sign_time_ms,
        "verify_all": verify_all,
        "total_sent_messages": total_sent_messages,
        "total_sent_bytes": total_sent_bytes,
        "total_received_messages": total_received_messages,
        "total_received_bytes": total_received_bytes,
        "relay_tss_messages": relay_tss_messages,
        "relay_payload_bytes_base64": relay_payload_bytes_base64,
        "controller_wall_ms": controller_wall_ms,
        "result_dir": str(run_dir),
    })

if __name__ == "__main__":
    main()
