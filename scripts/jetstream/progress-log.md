# Binance tss-lib Jetstream VM Completion Progress Log

## 2026-07-03

Completed:
- Freed controller root disk from 99% to about 80%.
- Created branch `angel-jetstream-vm-benchmarks`.
- Created Binance-only folders under:
  - `scripts/jetstream`
  - `scripts/analysis`
  - `cmd/tssbench/relay`
  - `cmd/tssbench/party`
- Fixed `cmd/localdemo/main.go`, which was missing a closing brace.
- Confirmed `go build ./...` passes.
- Confirmed local Binance baseline tests pass:
  - `go test -count=1 -v ./ecdsa/keygen`
  - `go test -count=1 -v ./ecdsa/signing`
  - `N=3 T=2 go test -count=1 -v ./ecdsa -run TestManualBenchConfigs`
  - `N=3 T=2 SIGS=5 MODE=fixed go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness`

Current result:
- Binance local/single-process baseline is healthy.
- Next step is controller-to-party SSH verification.

Next:
- Confirm controller can SSH into party VMs using internal IPs.
- Create a clean `parties.json`.
- Start Binance-only relay/party scaffold.
