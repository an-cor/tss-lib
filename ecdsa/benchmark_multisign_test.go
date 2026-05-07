package ecdsa

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bnb-chain/tss-lib/v2/common"
	kg "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	sg "github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	testutil "github.com/bnb-chain/tss-lib/v2/test"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

type MultiSignMode string

const (
	ModeFixed  MultiSignMode = "fixed"
	ModeRandom MultiSignMode = "random"
)

func makeFixedDigest() *big.Int {
	// Approximate "bitcoin transaction sized" payload:
	// a fixed 250-byte payload, then hashed to 32 bytes for signing.
	payload := make([]byte, 250)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	sum := sha256.Sum256(payload)
	return new(big.Int).SetBytes(sum[:])
}

func makeRandomDigest(t *testing.T) *big.Int {
	// Random 250-byte payload, then hashed to 32 bytes.
	payload := make([]byte, 250)
	_, err := rand.Read(payload)
	require.NoError(t, err)

	sum := sha256.Sum256(payload)
	return new(big.Int).SetBytes(sum[:])
}

func makeMessageForMode(t *testing.T, mode MultiSignMode) *big.Int {
	switch mode {
	case ModeFixed:
		return makeFixedDigest()
	case ModeRandom:
		return makeRandomDigest(t)
	default:
		t.Fatalf("unknown mode: %s", mode)
		return nil
	}
}

func runOneSigningSession(
	t *testing.T,
	keys []kg.LocalPartySaveData,
	allPIDs tss.SortedPartyIDs,
	threshold int,
	msg *big.Int,
) time.Duration {
	signPIDs := allPIDs[:threshold+1]
	signKeys := keys[:threshold+1]

	p2pCtx := tss.NewPeerContext(signPIDs)
	parties := make([]*sg.LocalParty, 0, len(signPIDs))

	errCh := make(chan *tss.Error, len(signPIDs))
	outCh := make(chan tss.Message, len(signPIDs)*10)
	endCh := make(chan *common.SignatureData, len(signPIDs))

	var ended int32
	updater := testutil.SharedPartyUpdater

	start := time.Now()

	for i := 0; i < len(signPIDs); i++ {
		params := tss.NewParameters(tss.S256(), p2pCtx, signPIDs[i], len(signPIDs), threshold)
		P := sg.NewLocalParty(msg, params, signKeys[i], outCh, endCh).(*sg.LocalParty)
		parties = append(parties, P)

		go func(P *sg.LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}

	for {
		select {
		case err := <-errCh:
			require.FailNow(t, err.Error())

		case msgOut := <-outCh:
			dest := msgOut.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index == msgOut.GetFrom().Index {
						continue
					}
					go updater(P, msgOut, errCh)
				}
			} else {
				require.NotEqual(t, dest[0].Index, msgOut.GetFrom().Index)
				go updater(parties[dest[0].Index], msgOut, errCh)
			}

		case sig := <-endCh:
			require.NotNil(t, sig)
			atomic.AddInt32(&ended, 1)

			if atomic.LoadInt32(&ended) == int32(len(signPIDs)) {
				return time.Since(start)
			}
		}
	}
}

func runMultiSign(
	t *testing.T,
	n int,
	threshold int,
	sigCount int,
	mode MultiSignMode,
) {
	// Reuse your existing manual keygen helper from earlier work.
	keys, pIDs, keygenDur, _, _ := runManualKeygen(t, n, threshold)

	totalSignDur := time.Duration(0)

	for i := 0; i < sigCount; i++ {
		msg := makeMessageForMode(t, mode)
		oneSignDur := runOneSigningSession(t, keys, pIDs, threshold, msg)
		totalSignDur += oneSignDur
	}

	avgSignDur := totalSignDur / time.Duration(sigCount)

	fmt.Println("n,t,sigs,mode,dkg_time_s,total_sign_time_s,avg_sign_time_s")
	fmt.Printf(
		"%d,%d,%d,%s,%.6f,%.6f,%.6f\n",
		n,
		threshold,
		sigCount,
		strings.Title(string(mode)),
		keygenDur.Seconds(),
		totalSignDur.Seconds(),
		avgSignDur.Seconds(),
	)
}

func TestManualMultiSignHarness(t *testing.T) {
	setBenchLogLevel("info")

	n := 3
	tVal := 2
	sigs := 1
	mode := ModeFixed

	if v := os.Getenv("N"); v != "" {
		fmt.Sscanf(v, "%d", &n)
	}
	if v := os.Getenv("T"); v != "" {
		fmt.Sscanf(v, "%d", &tVal)
	}
	if v := os.Getenv("SIGS"); v != "" {
		fmt.Sscanf(v, "%d", &sigs)
	}
	if v := os.Getenv("MODE"); v != "" {
		switch strings.ToLower(v) {
		case "fixed":
			mode = ModeFixed
		case "random":
			mode = ModeRandom
		default:
			t.Fatalf("unsupported MODE=%s (use fixed or random)", v)
		}
	}

	runMultiSign(t, n, tVal, sigs, mode)
}