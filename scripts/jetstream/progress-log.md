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

## Reusable cross-VM smoke script update

Completed:
- Added `scripts/jetstream/run_tss_smoke.sh`.
- Script builds relay/party binaries, copies party binary to selected VMs, starts controller relay, runs party smoke clients, collects logs, and checks `PARTY_OK`.
- Verified reusable smoke script for `n=3`.

Next:
- Start replacing hello payloads with serialized tss-lib protocol messages.
- Begin networked keygen implementation for `n=3,t=2`.

## Reusable cross-VM smoke script correction

Correction:
- The first reusable smoke script run failed because the embedded Python party-row printer used an invalid escaped f-string.
- The script was patched to use `.format(...)` instead.
- The reusable smoke script has now been verified for `n=3`.

Next:
- Begin the real networked tss-lib bridge:
  - wrap `LocalParty`
  - call `Start()`
  - forward `outCh` messages through the relay
  - feed inbound relay messages into `UpdateFromBytes(...)`

## Reusable smoke script cleanup fix

Correction:
- The first reusable smoke script did not complete after the cleanup phase.
- The cleanup command used `pkill -f tss_party`, which could match its own remote SSH command.
- Updated cleanup patterns to `[t]ss_party` and `[t]ss_relay`.
- Re-ran the reusable cross-VM smoke script successfully for `n=3`.

Next:
- Begin the real networked tss-lib bridge using `LocalParty`, `Start()`, `outCh`, and `UpdateFromBytes(...)`.

## Local multi-process keygen update

Completed:
- Added `--mode keygen` to `cmd/tssbench/party`.
- Verified local multi-process keygen with one controller relay and three separate party processes.
- Confirmed all three parties reached `KEYGEN_OK`.
- Confirmed keygen save and summary JSON files are written per party.

Next:
- Run the same networked keygen flow across the first 3 Jetstream party VMs.

## Cross-VM keygen update

Completed:
- Ran networked Binance tss-lib keygen across the first 3 Jetstream party VMs.
- Configuration: n=3, t=2.
- Controller ran `tss_relay`; party VMs ran `tss_party --mode keygen`.
- Confirmed all three parties reached `KEYGEN_OK`.
- Collected per-party keygen save and summary JSON files back to the controller.

Next:
- Convert the manual cross-VM keygen command sequence into a reusable script.
- Add basic metrics extraction for keygen runtime, messages, and bytes.

## Cross-VM keygen update

Completed:
- Ran networked Binance tss-lib keygen across the first 3 Jetstream party VMs.
- Configuration: n=3, t=2.
- Controller ran `tss_relay`; party VMs ran `tss_party --mode keygen`.
- Confirmed all three parties reached `KEYGEN_OK`.
- Collected per-party keygen save and summary JSON files back to the controller.

Next:
- Convert the manual cross-VM keygen command sequence into a reusable script.
- Add basic metrics extraction for keygen runtime, messages, and bytes.

## Reusable keygen script wait fix

Correction:
- The first reusable keygen script run hung after starting party jobs.
- Cause: the script waited on all background jobs, including the long-running relay process.
- Fixed the script to track and wait only for party SSH job PIDs.
- Re-ran the reusable keygen script successfully for n=3, t=2.

Next:
- Add basic metrics extraction for keygen runs.
- Then add signing mode using saved keygen shares.

## Keygen metrics update

Completed:
- Added per-party keygen elapsed timing to `tss_party --mode keygen`.
- Added `scripts/analysis/export_tss_keygen_csv.py`.
- Verified CSV export for a reusable n=3, t=2 cross-VM keygen run.

Next:
- Add signing mode using saved keygen shares.

## Local multi-process signing update

Completed:
- Added `--mode sign` to `cmd/tssbench/party`.
- Loaded saved keygen shares from prior cross-VM keygen artifacts.
- Verified local multi-process signing with one relay and three separate party processes.
- Confirmed all three parties reached `SIGN_OK`.
- Confirmed each signature verifies with `verify_ok=true`.

Next:
- Run the same signing flow across the first 3 Jetstream party VMs.

## Cross-VM signing update

Completed:
- Ran Binance tss-lib signing across the first 3 Jetstream party VMs.
- Reused keygen shares from the prior cross-VM keygen run.
- Configuration: n=3, t=2.
- Controller ran `tss_relay`; party VMs ran `tss_party --mode sign`.
- Confirmed all three parties reached `SIGN_OK`.
- Confirmed all signatures verified with `verify_ok=true`.
- Collected per-party signature JSON artifacts back to the controller.

Next:
- Convert the manual cross-VM signing command sequence into a reusable script.
- Add signing CSV export.

## Reusable cross-VM signing script update

Completed:
- Added `scripts/jetstream/run_tss_sign_round.sh`.
- Script builds binaries, copies `tss_party` to selected VMs, checks keygen shares, starts controller relay, runs networked signing parties, checks `SIGN_OK`, verifies `verify_ok=true`, and collects signature artifacts.
- Verified reusable script for n=3, t=2.

Next:
- Add signing CSV exporter.
- Then add combined keygen+sign script for comparison runs.

## Signing metrics update

Completed:
- Removed confusing pre-artifact verify warning from `run_tss_sign_round.sh`.
- Added `scripts/analysis/export_tss_sign_csv.py`.
- Verified CSV export for a reusable n=3, t=2 cross-VM signing run.

Next:
- Add combined keygen+sign script for complete base comparison runs.
