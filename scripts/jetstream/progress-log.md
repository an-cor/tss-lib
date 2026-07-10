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

## SSH verification update

Completed:
- Confirmed controller-to-party SSH works when using ~/.ssh/socioty_controller_key.
- Verified all 10 party VM internal IPs.
- Recovered full 10-party inventory from old DKLS controller files.
- Created clean Binance-owned inventory at scripts/jetstream/parties.json.
- Added reusable SSH checker at scripts/jetstream/check_parties.sh.

Next:
- Sync Binance repo/environment to party VMs.
- Create the first minimal Binance relay scaffold.
- Create the first minimal Binance party scaffold.

## Party VM sync update

Completed:
- Synced current tss-lib working tree from controller to first 3 party VMs.
- Confirmed first 3 party VMs can build tss-lib locally with `go build ./...`.

Next:
- Add minimal tss-lib relay scaffold under `cmd/tssbench/relay`.
- Add minimal tss-lib party scaffold under `cmd/tssbench/party`.
- Run first network smoke test before implementing full keygen.

## Party VM disk cleanup update

Completed:
- Cleaned old DKLS result folders, DKLS export artifacts, DKLS build targets, and temporary caches from party VMs.
- Preserved Binance `~/tss-lib` working copies.
- Rechecked party VM disk levels with `scripts/jetstream/check_parties.sh 10`.

Next:
- Continue Binance network-layer work.
- Add minimal relay under `cmd/tssbench/relay`.
- Add minimal party client under `cmd/tssbench/party`.

## Cross-VM smoke update

Completed:
- Built minimal `tss_relay` and `tss_party` binaries.
- Verified local controller-only relay smoke test.
- Verified cross-VM smoke test with controller relay and party clients on the first 3 party VMs.

Next:
- Convert the manual cross-VM smoke commands into a reusable script.
- Begin replacing hello payloads with serialized tss-lib messages.
- Start implementing networked keygen for n=3,t=2.

## Cross-VM smoke correction and verification

Correction:
- The first cross-VM smoke attempt did not pass because port 9100 was already occupied by an old DKLS relay process.
- Party clients timed out because the new `tss_relay` failed to bind.

Completed:
- Re-ran the cross-VM smoke test on port 19100.
- Verified controller relay can route a broadcast hello from party 1 to parties 2 and 3 across Jetstream internal IPs.

Next:
- Convert this manual cross-VM smoke test into a reusable script.
- Then begin replacing hello payloads with serialized tss-lib protocol messages.
