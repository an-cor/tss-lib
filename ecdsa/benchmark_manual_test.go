package ecdsa

import (
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"
	"time"
	"os"

	"github.com/ipfs/go-log"
	"github.com/stretchr/testify/require"

	"github.com/bnb-chain/tss-lib/v2/common"
	kg "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	sg "github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	testutil "github.com/bnb-chain/tss-lib/v2/test"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

type BenchResult struct {
	N            int
	T            int
	KeygenTime    time.Duration
	SignTime      time.Duration
	KeygenMsg     int64
	SignMsg       int64
	KeygenBytes   int64
	SignBytes     int64
}

func setBenchLogLevel(level string) {
	if err := log.SetLogLevel("tss-lib", level); err != nil {
		panic(err)
	}
}

func runManualKeygen(t *testing.T, n, threshold int) ([]kg.LocalPartySaveData, tss.SortedPartyIDs, time.Duration, int64, int64) {
	pIDs := tss.GenerateTestPartyIDs(n)
	p2pCtx := tss.NewPeerContext(pIDs)

	parties := make([]*kg.LocalParty, 0, len(pIDs))
	errCh := make(chan *tss.Error, len(pIDs))
	outCh := make(chan tss.Message, n*10)
	endCh := make(chan *kg.LocalPartySaveData, len(pIDs))

	start := time.Now()
	var msgCount int64
	var byteCount int64
	var ended int32

	updater := testutil.SharedPartyUpdater
	saves := make([]kg.LocalPartySaveData, len(pIDs))

	for i := 0; i < len(pIDs); i++ {
		params := tss.NewParameters(tss.S256(), p2pCtx, pIDs[i], len(pIDs), threshold)

		// keep same fast test behavior as existing keygen test
		params.SetNoProofMod()
		params.SetNoProofFac()

		P := kg.NewLocalParty(params, outCh, endCh).(*kg.LocalParty)
		parties = append(parties, P)

		go func(P *kg.LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}

	for {
		select {
		case err := <-errCh:
			require.FailNow(t, err.Error())

		case msg := <-outCh:
			atomic.AddInt64(&msgCount, 1)
			if bz, _, err := msg.WireBytes(); err == nil {
				atomic.AddInt64(&byteCount, int64(len(bz)))
			}

			dest := msg.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh)
				}
			} else {
				require.NotEqual(t, dest[0].Index, msg.GetFrom().Index)
				go updater(parties[dest[0].Index], msg, errCh)
			}

		case save := <-endCh:
			idx, err := save.OriginalIndex()
			require.NoError(t, err)
			saves[idx] = *save

			atomic.AddInt32(&ended, 1)
			if atomic.LoadInt32(&ended) == int32(len(pIDs)) {
				return saves, pIDs, time.Since(start), msgCount, byteCount
			}
		}
	}
}

func runManualSigning(t *testing.T, keys []kg.LocalPartySaveData, allPIDs tss.SortedPartyIDs, threshold int) (time.Duration, int64, int64) {
	// sign with exactly t+1 participants
	signPIDs := allPIDs[:threshold+1]
	signKeys := keys[:threshold+1]

	p2pCtx := tss.NewPeerContext(signPIDs)
	parties := make([]*sg.LocalParty, 0, len(signPIDs))

	errCh := make(chan *tss.Error, len(signPIDs))
	outCh := make(chan tss.Message, len(signPIDs)*10)
	endCh := make(chan *common.SignatureData, len(signPIDs))

	start := time.Now()
	var msgCount int64
	var byteCount int64
	var ended int32

	updater := testutil.SharedPartyUpdater
	msg := big.NewInt(42)

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

		case msg := <-outCh:
			atomic.AddInt64(&msgCount, 1)
			if bz, _, err := msg.WireBytes(); err == nil {
				atomic.AddInt64(&byteCount, int64(len(bz)))
			}

			dest := msg.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh)
				}
			} else {
				require.NotEqual(t, dest[0].Index, msg.GetFrom().Index)
				go updater(parties[dest[0].Index], msg, errCh)
			}

		case <-endCh:
			atomic.AddInt32(&ended, 1)
			if atomic.LoadInt32(&ended) == int32(len(signPIDs)) {
				return time.Since(start), msgCount, byteCount
			}
		}
	}
}

func TestManualBenchConfigs(t *testing.T) {
	setBenchLogLevel("info")

	n := 3
	tVal := 2

	if v := os.Getenv("N"); v != "" {
		fmt.Sscanf(v, "%d", &n)
	}
	if v := os.Getenv("T"); v != "" {
		fmt.Sscanf(v, "%d", &tVal)
	}

	configs := []struct {
		n int
		t int
	}{
		{n, tVal},
	}

	fmt.Println("n,t,keygen_s,sign_s,total_s,keygen_msg,sign_msg,keygen_bytes,sign_bytes")

	for _, cfg := range configs {
		keys, pIDs, kgTime, kgMsg, kgBytes := runManualKeygen(t, cfg.n, cfg.t)
		sgTime, sgMsg, sgBytes := runManualSigning(t, keys, pIDs, cfg.t)

		total := kgTime + sgTime

		fmt.Printf(
			"%d,%d,%.6f,%.6f,%.6f,%d,%d,%d,%d\n",
			cfg.n,
			cfg.t,
			kgTime.Seconds(),
			sgTime.Seconds(),
			total.Seconds(),
			kgMsg,
			sgMsg,
			kgBytes,
			sgBytes,
		)
	}
}