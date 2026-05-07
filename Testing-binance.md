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