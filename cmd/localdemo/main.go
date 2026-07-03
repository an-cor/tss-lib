package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"sync/atomic"

	log "github.com/ipfs/go-log"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/test"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func setUp(level string) {
	if err := log.SetLogLevel("tss-lib", level); err != nil {
		panic(err)
	}
}

func main() {
	setUp("info")

	n := test.TestParticipants
	threshold := test.TestThreshold
	signers := threshold + 1
	msg := big.NewInt(42).Bytes()

	fmt.Printf("== tss-lib localdemo ==\n")
	fmt.Printf("N=%d, threshold=%d, signers=%d\n\n", n, threshold, signers)

	// -------------------------
	// PHASE 1: KEYGEN
	// -------------------------
	fmt.Println("== PHASE: KEYGEN ==")

	fixtures, pIDs, err := keygen.LoadKeygenTestFixtures(n)
	if err != nil {
		common.Logger.Info("No keygen fixtures found; generating safe primes from scratch (may take a while)...")
		pIDs = tss.GenerateTestPartyIDs(n)
	}

	p2pCtx := tss.NewPeerContext(pIDs)
	parties := make([]*keygen.LocalParty, 0, len(pIDs))

	errCh := make(chan *tss.Error, len(pIDs))
	outCh := make(chan tss.Message, len(pIDs))
	endCh := make(chan *keygen.LocalPartySaveData, len(pIDs))

	updater := test.SharedPartyUpdater
	startGR := runtime.NumGoroutine()

	for i := 0; i < len(pIDs); i++ {
		params := tss.NewParameters(tss.S256(), p2pCtx, pIDs[i], len(pIDs), threshold)

		// NOT for untrusted settings
		params.SetNoProofMod()
		params.SetNoProofFac()

		var P *keygen.LocalParty
		if i < len(fixtures) {
			P = keygen.NewLocalParty(params, outCh, endCh, fixtures[i].LocalPreParams).(*keygen.LocalParty)
		} else {
			P = keygen.NewLocalParty(params, outCh, endCh).(*keygen.LocalParty)
		}
		parties = append(parties, P)

		go func(P *keygen.LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}

	var ended int32
	var saves []*keygen.LocalPartySaveData

keygenLoop:
	for {
		select {
		case err := <-errCh:
			fmt.Printf("KEYGEN ERROR: %v\n", err)
			os.Exit(1)

		case msg := <-outCh:
			dest := msg.GetTo()
			if dest == nil { // broadcast
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh)
				}
			} else { // p2p
				go updater(parties[dest[0].Index], msg, errCh)
			}

		case save := <-endCh:
			saves = append(saves, save)
			atomic.AddInt32(&ended, 1)
			if int(atomic.LoadInt32(&ended)) == len(pIDs) {
				break keygenLoop
			}
		}
	}

	save0 := saves[0]
	pkX, pkY := save0.ECDSAPub.X(), save0.ECDSAPub.Y()
	pk := ecdsa.PublicKey{Curve: tss.EC(), X: pkX, Y: pkY}

	fmt.Printf("Keygen complete. Public key:\nX=%s\nY=%s\n", pkX.String(), pkY.String())
	fmt.Printf("Public key on curve: %v\n", pk.IsOnCurve(pkX, pkY))
	fmt.Printf("Goroutines: start=%d end=%d\n\n", startGR, runtime.NumGoroutine())

	// -------------------------
	// PHASE 2: SIGNING
	// -------------------------
	fmt.Println("== PHASE: SIGNING ==")

	keys, signPIDs, err := keygen.LoadKeygenTestFixturesRandomSet(signers, n)
	if err != nil {
		fmt.Println("Could not load signing fixtures. Run:")
		fmt.Println("  go test -count=1 -v ./ecdsa/keygen -run TestE2EConcurrentAndSaveFixtures")
		os.Exit(1)
	}

	p2pCtx2 := tss.NewPeerContext(signPIDs)
	signParties := make([]*signing.LocalParty, 0, len(signPIDs))

	signErrCh := make(chan *tss.Error, len(signPIDs))
	signOutCh := make(chan tss.Message, len(signPIDs))
	signEndCh := make(chan *common.SignatureData, len(signPIDs))

	for i := 0; i < len(signPIDs); i++ {
		params := tss.NewParameters(tss.S256(), p2pCtx2, signPIDs[i], len(signPIDs), threshold)
		P := signing.NewLocalParty(big.NewInt(42), params, keys[i], signOutCh, signEndCh).(*signing.LocalParty)
		signParties = append(signParties, P)

		go func(P *signing.LocalParty) {
			if err := P.Start(); err != nil {
				signErrCh <- err
			}
		}(P)
	}

	var signEnded int32

signLoop:
	for {
		select {
		case err := <-signErrCh:
			fmt.Printf("SIGN ERROR: %v\n", err)
			os.Exit(1)

		case msg := <-signOutCh:
			dest := msg.GetTo()
			if dest == nil { // broadcast
				for _, P := range signParties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, signErrCh)
				}
			} else { // p2p
				go updater(signParties[dest[0].Index], msg, signErrCh)
			}

		case sd := <-signEndCh:
			// Print once
			fmt.Printf("SignatureData: %#v\n", sd)

			// Parse r,s
			r := new(big.Int).SetBytes(sd.R)
			s := new(big.Int).SetBytes(sd.S)

			// Verify against the message bytes in sd.M (or your local msg, they match)
			ok := ecdsa.Verify(&pk, sd.M, r, s)

			fmt.Printf("\nVERIFY:\nmsg=%x\nr=%s\ns=%s\nok=%v\n\n", sd.M, r.String(), s.String(), ok)

			atomic.AddInt32(&signEnded, 1)
			if int(atomic.LoadInt32(&signEnded)) == len(signPIDs) {
				break signLoop
			}
		}
	}

	fmt.Println("\nSigning completed (all parties ended).")
	fmt.Println("Next: extract (r,s) from SignatureData and verify with ecdsa.Verify(pk, msg, r, s).")
	_ = msg
}
