#!/usr/bin/env python3
import csv
import json
import re
import sys
from pathlib import Path

RSS_RE = re.compile(r"Maximum resident set size.*:\s*(\d+)")

def parse_time_v(path: Path):
    if not path.exists():
        return None
    text = path.read_text(errors="replace")
    m = RSS_RE.search(text)
    return int(m.group(1)) if m else None

def parse_json_rss(path: Path):
    try:
        data = json.loads(path.read_text())
    except Exception:
        return None

    for key in ("max_rss_kb", "max_rss_per_party_kb"):
        val = data.get(key)
        if val in ("", None):
            continue
        try:
            return int(val)
        except Exception:
            pass

    return None

def max_rss_under_result_dir(result_dir: str):
    if not result_dir:
        return None

    root = Path(result_dir)
    if not root.exists():
        return None

    vals = []

    for pattern in [
        "artifacts/party*/keygen_summary_party_*.json",
        "artifacts/party*/signature_summary_party_*.json",
        "party*/keygen_summary_party_*.json",
        "party*/signature_summary_party_*.json",
    ]:
        for path in root.glob(pattern):
            v = parse_json_rss(path)
            if v is not None:
                vals.append(v)

    for pattern in [
        "artifacts/party*/time_v.txt",
        "party*/time_v.txt",
    ]:
        for path in root.glob(pattern):
            v = parse_time_v(path)
            if v is not None:
                vals.append(v)

    return max(vals) if vals else None

def max_sign_rss_from_multisign_dir(multisign_dir: str):
    if not multisign_dir:
        return None

    root = Path(multisign_dir)
    sign_rounds = root / "sign_rounds_summary.csv"

    if not sign_rounds.exists():
        return None

    vals = []
    with open(sign_rounds, newline="") as f:
        for row in csv.DictReader(f):
            v = max_rss_under_result_dir(row.get("result_dir", ""))
            if v is not None:
                vals.append(v)

    return max(vals) if vals else None

def add_rss(row):
    if "result_dir" in row:
        v = max_rss_under_result_dir(row.get("result_dir", ""))
        row["max_rss_per_party_kb"] = "" if v is None else str(v)

    if "keygen_result_dir" in row:
        kg = max_rss_under_result_dir(row.get("keygen_result_dir", ""))
        row["keygen_max_rss_per_party_kb"] = "" if kg is None else str(kg)

    if "sign_result_dir" in row:
        sg = max_rss_under_result_dir(row.get("sign_result_dir", ""))
        row["sign_max_rss_per_party_kb"] = "" if sg is None else str(sg)

    if "multisign_result_dir" in row:
        sg_multi = max_sign_rss_from_multisign_dir(row.get("multisign_result_dir", ""))
        row["sign_max_rss_per_party_kb"] = "" if sg_multi is None else str(sg_multi)

    candidates = []
    for key in (
        "max_rss_per_party_kb",
        "keygen_max_rss_per_party_kb",
        "sign_max_rss_per_party_kb",
    ):
        val = str(row.get(key, "")).strip()
        if val:
            try:
                candidates.append(int(val))
            except ValueError:
                pass

    row["overall_max_rss_per_party_kb"] = str(max(candidates)) if candidates else ""
    return row

def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: add_rss_columns.py SUMMARY.csv")

    path = Path(sys.argv[1])

    with open(path, newline="") as f:
        rows = list(csv.DictReader(f))

    if not rows:
        raise SystemExit(f"no rows in {path}")

    new_rows = [add_rss(dict(row)) for row in rows]

    fieldnames = []
    for row in new_rows:
        for key in row:
            if key not in fieldnames:
                fieldnames.append(key)

    writer = csv.DictWriter(sys.stdout, fieldnames=fieldnames)
    writer.writeheader()
    writer.writerows(new_rows)

if __name__ == "__main__":
    main()
