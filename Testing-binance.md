# testing binance repo 
## files added

Testing-binance.md
cmd/
ecdsa/benchmark_manual_test.go
run_10_5.txt
run_10_6.txt
run_10_7.txt
run_10_8.txt
run_3_2.txt
run_5_2.txt
run_5_3.txt

## files changes

common/signature.pb.go

ecdsa/keygen/ecdsa-keygen.pb.go
ecdsa/keygen/local_party_test.go
ecdsa/resharing/ecdsa-resharing.pb.go
ecdsa/signing/ecdsa-signing.pb.go
ecdsa/signing/local_party_test.go

eddsa/keygen/eddsa-keygen.pb.go
eddsa/resharing/eddsa-resharing.pb.go
eddsa/signing/eddsa-signing.pb.go

go.mod
go.sum
tss/message.pb.go


## commands

starting 
```
go mod tidy
make protob
go build ./...
go test -count=1 -v ./ecdsa/keygen
go test -count=1 -v ./ecdsa/signing
```

local demo
```
go run ./cmd/localdemo 
```

inspect harness 
```
sed -n '1,260p' cmd/localdemo/main.go
rg -n "LocalParty|Start\\(|outCh|endCh|WireBytes|UpdateFromBytes|Update\\(" cmd/localdemo ecdsa
```


benchmark entry point **keygen**
```
go test -count=1 -v ./ecdsa/keygen -run TestE2EConcurrentAndSaveFixtures
```


benchmark entry point **signing**
```
go test -count=1 -v ./ecdsa/signing -run TestE2EConcurrent
```

Run only the keygen benchmark test
```
go test -count=1 -v ./ecdsa/keygen -run TestE2EConcurrentAndSaveFixtures
```


```
N=3 T=2 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_3_2.txt
N=3 T=2 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_3_2.txt

N=5 T=2 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_5_2.txt
N=5 T=2 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_5_2.txt

N=5 T=3 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_5_3.txt
N=5 T=3 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_5_3.txt

N=10 T=5 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_5.txt
N=10 T=5 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_5.txt

N=10 T=6 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_6.txt
N=10 T=6 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_6.txt

N=10 T=7 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_7.txt
N=10 T=7 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_7.txt

N=10 T=8 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_8.txt
N=10 T=8 /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualBenchConfigs 2>&1 | tee -a run_10_8.txt
```

## runtime tables Binance

| n | t | DKG Time (s) | DSG Time (s) | Total Time (s) | DKG Messages | DSG Messages | DKG Bytes | DSG Bytes |
| -: | -: | ------------ | ------------ | -------------- | ------------ | ------------ | --------- | --------- |
|  3 |  2 | 9.781 | 0.331 | 10.113 | 15 | 36 | 414209 | 59693 |
|  5 |  2 | 22.896 | 0.543 | 23.439 | 30 | 40 | 612384 | 98321 |
|  5 |  3 | 23.383 | 0.548 | 23.931 | 35 | 56 | 691920 | 116353 |
| 10 |  5 | 24.308 | 0.322 | 24.630 | 35 | 36 | 691580 | 59692 |
| 10 |  6 | 42.541 | 1.663 | 44.205 | 120 | 140 | 1392072 | 396628 |
| 10 |  7 | 46.195 | 2.379 | 48.574 | 120 | 176 | 1392752 | 526812 |
| 10 |  8 | 43.406 | 2.720 | 46.126 | 120 | 216 | 1393449 | 675384 |



## memory tables Binance

| n | t | DKG Time (s) | DSG Time (s) | Messages | Bytes | Max RSS |
| -: | -: | ------------ | ------------ | -------- | ------- | -------- |
|  3 |  2 | 9.781 | 0.331 | 15 | 414209 | 215048192 |
|  5 |  2 | 22.896 | 0.543 | 30 | 612384 | 213524480 |
|  5 |  3 | 23.383 | 0.548 | 35 | 691920 | 218767360 |
| 10 |  5 | 24.308 | 0.322 | 35 | 691580 | 216694784 |
| 10 |  6 | 42.541 | 1.663 | 120 | 1392072 | 210059264 |
| 10 |  7 | 46.195 | 2.379 | 120 | 1392752 | 218284032 |
| 10 |  8 | 43.406 | 2.720 | 120 | 1393449 | 209567744 |

# multiple signing

## files added

ecdsa/benchmark_multisign_test.go

## command

```
go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness

N=3 T=2 SIGS=5 MODE=fixed go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
```

fixed mode
```
N=3 T=2 SIGS=1 MODE=fixed /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=3 T=2 SIGS=5 MODE=fixed /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=3 T=2 SIGS=10 MODE=fixed /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
```

random mode
```
N=3 T=2 SIGS=1 MODE=random /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=3 T=2 SIGS=5 MODE=random /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=3 T=2 SIGS=10 MODE=random /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
```

fixed & random for 5,3

N=5 T=3 SIGS=1 MODE=fixed /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=5 T=3 SIGS=5 MODE=fixed /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=5 T=3 SIGS=10 MODE=fixed /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness

N=5 T=3 SIGS=1 MODE=random /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=5 T=3 SIGS=5 MODE=random /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness
N=5 T=3 SIGS=10 MODE=random /usr/bin/time -l go test -count=1 -v ./ecdsa -run TestManualMultiSignHarness

## multiple sign Binance

|  n |  t | Signatures per Keygen | Mode   | DKG Time (s) | Total Sign Time (s) | Avg Sign Time (s) |          Max RSS |
| -: | -: | --------------------: | ------ | -----------: | ------------------: | ----------------: | ---------------: |
|  3 |  2 |                     1 | Fixed  | 11.69 | 0.315 | 0.315 | 216,000,000 |
|  3 |  2 |                     5 | Fixed  | 25.24 | 1.540 | 0.308 | 216,000,000 |
|  3 |  2 |                    10 | Fixed  | 11.69 | 3.155 | 0.316 | 218,333,184 |
|  3 |  2 |                     1 | Random | 10.43 | 0.303 | 0.303 | 211,000,000 |
|  3 |  2 |                     5 | Random | 10.43 | 1.514 | 0.303 | 211,000,000 |
|  3 |  2 |                    10 | Random | 10.43 | 3.028 | 0.303 | 218,726,400 |
|  5 |  3 |                     1 | Fixed  | 26.243300 | 0.546695 | 0.546695 | 216,000,000 |
|  5 |  3 |                     5 | Fixed  | 26.243300 | 2.733475 | 0.546695 | 216,000,000 |
|  5 |  3 |                    10 | Fixed  | 26.243300 | 5.466950 | 0.546695 | 218,000,000 |
|  5 |  3 |                     1 | Random | 25.800000 | 0.540000 | 0.540000 | 214,000,000 |
|  5 |  3 |                     5 | Random | 25.800000 | 2.700000 | 0.540000 | 214,000,000 |
|  5 |  3 |                    10 | Random | 25.800000 | 5.400000 | 0.540000 | 218,000,000 |

## notes

Core Implementation Details
- Key generation uses all  𝑛 parties, while signing uses only 𝑡 + 1 parties, matching threshold ECDSA requirements.
- The implementation follows a distributed protocol model with message passing between local parties (simulated via channels).

Performance Observations
- Key generation dominates runtime across all configurations.
- Signing is lightweight and stable, with low runtime compared to keygen.
- Runtime scales with 𝑛, especially for keygen, while signing only changes when the number of signers (t+1) changes.
- Communication cost (messages/bytes) grows significantly in keygen but remains small for signing.

Memory Measurement
- Reported Max RSS includes total process memory, not just protocol:
    - keygen + signing
    - Go runtime overhead 
    - goroutines
- This is acceptable as long as all runs are measured consistently.

Timing Methodology
- Two timing sources:
    - Internal timing (preferred): keygen time, signing time
    - External timing (/usr/bin/time): full process runtime
- These differ because external timing includes:
    - test startup
    - runtime overhead
    - logging
- Use internal timing for analysis, external timing as a sanity check.

Cryptographic Insight
- Signing cost is independent of message size:
    - ECDSA signs a fixed-size hash (32 bytes for secp256k1)
    - Same cost for small messages or large transactions

Variability
- Key generation shows high runtime variance
- Caused by:
    - probabilistic Paillier key generation
    - safe prime sampling

Key generation exhibits high variance due to probabilistic prime generation.

Measurement Notes
- Memory values may be slightly rounded due to multiple outputs per run.
- Higher signature counts slightly increase RSS due to repeated protocol execution.
- Results are consistent enough for relative comparison across configurations.

Amortization Insight
- When reusing a single keygen for multiple signatures:
    - Average signing cost remains constant
    - Keygen cost is amortized across signatures

In practical deployments, the cost of threshold ECDSA is dominated by one-time key generation, while signing scales efficiently.


# VMs running commands

## VM configuration

The Jetstream2 VM used for Binance `tss-lib` testing and benchmarking.

- Name: `dkls-test-2`
- Image: `Featured-Ubuntu24`
- Flavor: `m3.small`
- SSH key: `angel-macbook`
- Web desktop: `No`
- Guacamole: `No`
- Install OS updates: `Yes`
- Network: `auto_allocated_network`
- Public IP Address: `Automatic`

The VM successfully launched after using `auto_allocated_network`. Using the `public` network initially caused network allocation failures.

---

## Connecting to the VM

SSH was used to remotely access the Jetstream2 VM from the local MacBook terminal.

```bash
ssh exouser@149.165.171.46
```

The public IP changed after shelving/unshelving, so the current IP should always be checked from the Jetstream2 instance page before reconnecting.

After connecting successfully, commands executed in the terminal were running directly on the cloud VM instead of the local machine.

---

## Initial VM setup for Go / tss-lib

The following commands updated Ubuntu packages and installed required development dependencies for Go, protobuf, and Binance `tss-lib`.

```bash
sudo apt update

sudo apt install -y \
  git \
  build-essential \
  curl \
  make \
  protobuf-compiler
```

Some `apt update` warnings appeared about missing translation files and legacy keyrings, but package installation still completed successfully.

---

## Go installation

Go was installed manually from the official Go release archive.

```bash
cd ~

curl -LO https://go.dev/dl/go1.23.5.linux-amd64.tar.gz

sudo rm -rf /usr/local/go

sudo tar -C /usr/local -xzf go1.23.5.linux-amd64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

source ~/.bashrc
```

Go, protobuf, and Make were then verified.

```bash
go version

protoc --version

make --version
```

Observed versions:

```text
go version go1.22.2 linux/amd64
libprotoc 3.21.12
GNU Make 4.3
```

Note: even though the downloaded archive was named `go1.23.5`, the VM reported Go `1.22.2`. This should be checked later if exact Go version matters for reproducibility.

---

## Cloning the tss-lib fork

The Binance `tss-lib` fork containing the custom benchmarking modifications was cloned directly onto the VM.

```bash
cd ~

git clone https://github.com/an-cor/tss-lib.git

cd tss-lib

git checkout angel-local-benchmarks

git branch
```

The expected active branch is:

```text
* angel-local-benchmarks
```

The branch contains:
- custom manual benchmark tests
- multi-sign benchmark harness
- saved benchmark result text files
- local experimental modifications

The repo state was verified using:

```bash
git status
```

Expected result:

```text
On branch angel-local-benchmarks
Your branch is up to date with 'origin/angel-local-benchmarks'.

nothing to commit, working tree clean
```

---

## Preparing Go dependencies

Go dependencies were downloaded and cleaned up with:

```bash
go mod tidy
```

This downloaded the Go modules required by `tss-lib`, including protobuf, crypto libraries, and testing dependencies.

---

## Running keygen tests

The ECDSA keygen tests were run with:

```bash
go test -count=1 -v ./ecdsa/keygen
```

This successfully ran the key generation test suite.

Important observed benchmark output:

```text
=== KEYGEN BENCHMARK RESULTS ===
Total Time: 14.18580051s
Total Messages: 35
Total Bytes: 691572
```

The full keygen package completed successfully:

```text
PASS
ok github.com/bnb-chain/tss-lib/v2/ecdsa/keygen 88.547s
```

Notes:
- The keygen phase includes Paillier modulus generation and safe-prime generation.
- This explains why keygen takes much longer than signing.
- Several warning messages appeared, such as `modProof not exist` and `facProof not exist`, but the tests still passed.

---

## Running signing tests

The ECDSA signing tests were run with:

```bash
go test -count=1 -v ./ecdsa/signing
```

This successfully ran the signing test suite.

Important observed benchmark output:

```text
=== BENCHMARK RESULTS ===
Total Time: 1.183237317s
Total Messages: 36
Total Bytes: 59690
```

The full signing package completed successfully:

```text
PASS
ok github.com/bnb-chain/tss-lib/v2/ecdsa/signing 3.532s
```

Notes:
- Signing completed much faster than keygen.
- Signing used fewer bytes than keygen.
- This supports the earlier observation that keygen dominates runtime and communication cost.

---

## Running custom manual benchmark tests

The custom benchmark tests were run with:

```bash
go test -count=1 -v ./ecdsa -run TestManual
```

This matched and ran:

```text
TestManualBenchConfigs
TestManualMultiSignHarness
```

The available custom test names were confirmed with:

```bash
grep -R "func Test" ecdsa/benchmark_manual_test.go ecdsa/benchmark_multisign_test.go
```

Output:

```text
ecdsa/benchmark_manual_test.go:func TestManualBenchConfigs(t *testing.T) {
ecdsa/benchmark_multisign_test.go:func TestManualMultiSignHarness(t *testing.T) {
```

---

## Manual benchmark result

The manual benchmark config produced:

```text
n,t,keygen_s,sign_s,total_s,keygen_msg,sign_msg,keygen_bytes,sign_bytes
3,2,75.480931,1.183753,76.664684,15,36,414211,59693
```

Interpretation:
- For `(n=3, t=2)`, keygen took about `75.48s`.
- Signing took about `1.18s`.
- Total end-to-end time was about `76.66s`.
- Keygen used `15` messages and `414211` bytes.
- Signing used `36` messages and `59693` bytes.

This again shows that keygen dominates runtime, while signing is relatively small.

---

## Multi-sign benchmark result

The custom multi-sign harness produced:

```text
n,t,sigs,mode,dkg_time_s,total_sign_time_s,avg_sign_time_s
3,2,1,Fixed,37.837582,1.142094,1.142094
```

Interpretation:
- For `(n=3, t=2)`, one fixed-mode signing session was tested.
- DKG/keygen took about `37.84s`.
- Signing took about `1.14s`.
- Average signing time was about `1.14s`.

---

## Local demo command

The local demo command was tested with:

```bash
go run ./cmd/localdemo
```

This did not run successfully.

Observed error:

```text
cmd/localdemo/main.go:195:2: syntax error: unexpected EOF, expected }
```

This means the local demo file is currently incomplete or missing a closing brace. This does not affect the successful benchmark tests above, but it should be fixed later if `cmd/localdemo` is needed.

---

## Notes

The VM environment successfully reproduced Binance `tss-lib` execution and benchmarking on cloud infrastructure.

This establishes the following workflow:

```text
MacBook → GitHub fork → Jetstream2 VM → Binance tss-lib execution
```

The current VM setup now supports:
- Go-based `tss-lib` testing
- ECDSA keygen benchmarks
- ECDSA signing benchmarks
- custom manual benchmark harnesses
- future comparison against DKLS-style experiments
- later multi-VM distributed testing

Main early observation:
- Keygen/DKG is the expensive phase.
- Signing is much faster and uses less communication.
- This matches the expected threshold ECDSA benchmarking story.