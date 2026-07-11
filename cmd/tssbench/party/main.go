package main

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	kg "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	sg "github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

type Envelope struct {
	Type            string `json:"type"`
	RunID           string `json:"run_id"`
	From            int    `json:"from"`
	To              int    `json:"to,omitempty"` // 0 means broadcast
	Payload         string `json:"payload,omitempty"`
	SentAtUnixNanos int64  `json:"sent_at_unix_nanos,omitempty"`
}

func main() {
	mode := flag.String("mode", "smoke", "mode: smoke, keygen, or sign")
	id := flag.Int("id", 0, "party id, 1-based")
	n := flag.Int("n", 0, "number of parties")
	threshold := flag.Int("t", 0, "threshold value used by tss-lib")
	signerCountFlag := flag.Int("signers", 0, "sign mode: number of signing participants; defaults to n")
	signerIDsFlag := flag.String("signer-ids", "", "sign mode: comma-separated actual party IDs to use, e.g. 1,3,4,5")
	keygenSavePath := flag.String("keygen-save", "", "sign mode: path to keygen LocalPartySaveData JSON")
	msgValue := flag.String("msg", "42", "sign mode: integer message to sign")
	relayAddr := flag.String("relay", "127.0.0.1:9100", "relay host:port")
	runID := flag.String("run", "manual", "run identifier")
	outDir := flag.String("out-dir", ".", "output directory for protocol artifacts")
	sendHello := flag.Bool("send", false, "smoke mode: send a hello broadcast after registration")
	to := flag.Int("to", 0, "smoke mode: destination party id; 0 means broadcast")
	expectHello := flag.Int("expect", 0, "smoke mode: number of hello messages expected before success")
	timeout := flag.Duration("timeout", 120*time.Second, "max time to wait")
	startDelay := flag.Duration("start-delay", 5*time.Second, "keygen mode: delay after registration before starting LocalParty")
	flag.Parse()

	if *id <= 0 {
		log.Fatalf("missing or invalid -id")
	}
	if *n <= 0 {
		log.Fatalf("missing or invalid -n")
	}

	switch *mode {
	case "smoke":
		runSmoke(*id, *n, *relayAddr, *runID, *sendHello, *to, *expectHello, *timeout)
	case "keygen":
		if *threshold <= 0 {
			log.Fatalf("keygen mode requires -t")
		}
		runKeygen(*id, *n, *threshold, *relayAddr, *runID, *outDir, *timeout, *startDelay)
	case "sign":
		if *threshold <= 0 {
			log.Fatalf("sign mode requires -t")
		}
		if *keygenSavePath == "" {
			log.Fatalf("sign mode requires -keygen-save")
		}
		runSign(*id, *n, *threshold, *signerCountFlag, *signerIDsFlag, *relayAddr, *runID, *outDir, *keygenSavePath, *msgValue, *timeout, *startDelay)
	default:
		log.Fatalf("unknown -mode %q; expected smoke, keygen, or sign", *mode)
	}
}

func connect(relayAddr string) (net.Conn, *json.Encoder, *json.Decoder) {
	conn, err := net.DialTimeout("tcp", relayAddr, 5*time.Second)
	if err != nil {
		log.Fatalf("connect relay=%s failed: %v", relayAddr, err)
	}
	return conn, json.NewEncoder(conn), json.NewDecoder(conn)
}

func register(enc *json.Encoder, runID string, id int) {
	err := enc.Encode(Envelope{
		Type:            "register",
		RunID:           runID,
		From:            id,
		Payload:         fmt.Sprintf("party-%d-register", id),
		SentAtUnixNanos: time.Now().UnixNano(),
	})
	if err != nil {
		log.Fatalf("party=%d register failed: %v", id, err)
	}
}

func startReader(dec *json.Decoder) (<-chan Envelope, <-chan error) {
	msgCh := make(chan Envelope, 32)
	errCh := make(chan error, 1)

	go func() {
		for {
			var msg Envelope
			if err := dec.Decode(&msg); err != nil {
				if err == io.EOF {
					errCh <- err
					return
				}
				errCh <- err
				return
			}
			msgCh <- msg
		}
	}()

	return msgCh, errCh
}

func runSmoke(id, n int, relayAddr, runID string, sendHello bool, to, expectHello int, timeout time.Duration) {
	conn, enc, dec := connect(relayAddr)
	defer conn.Close()

	register(enc, runID, id)

	msgCh, errCh := startReader(dec)
	deadline := time.After(timeout)

	registered := false
	helloSent := false
	helloReceived := 0

	for {
		select {
		case msg := <-msgCh:
			b, _ := json.Marshal(msg)
			fmt.Printf("RECV party=%d msg=%s\n", id, string(b))

			if msg.Type == "registered" {
				registered = true

				if sendHello && !helloSent {
					out := Envelope{
						Type:            "hello",
						RunID:           runID,
						From:            id,
						To:              to,
						Payload:         fmt.Sprintf("hello from party %d of %d", id, n),
						SentAtUnixNanos: time.Now().UnixNano(),
					}

					if err := enc.Encode(out); err != nil {
						log.Fatalf("party=%d send hello failed: %v", id, err)
					}

					fmt.Printf("SENT party=%d type=hello to=%d\n", id, to)
					helloSent = true
				}
			}

			if msg.Type == "hello" {
				helloReceived++
			}

			if registered && (!sendHello || helloSent) && helloReceived >= expectHello {
				fmt.Printf("PARTY_OK id=%d registered=%v hello_sent=%v hello_received=%d expect=%d\n",
					id, registered, helloSent, helloReceived, expectHello)
				return
			}

		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "PARTY_ERROR id=%d err=%v\n", id, err)
			os.Exit(1)

		case <-deadline:
			fmt.Fprintf(os.Stderr, "PARTY_TIMEOUT id=%d registered=%v hello_sent=%v hello_received=%d expect=%d\n",
				id, registered, helloSent, helloReceived, expectHello)
			os.Exit(2)
		}
	}
}

func makeDeterministicPartyIDs(n int) tss.SortedPartyIDs {
	ids := make(tss.UnSortedPartyIDs, 0, n)

	for i := 1; i <= n; i++ {
		ids = append(ids, tss.NewPartyID(
			fmt.Sprintf("%d", i),
			fmt.Sprintf("P[%d]", i),
			big.NewInt(int64(i)),
		))
	}

	return tss.SortPartyIDs(ids)
}

func runKeygen(id, n, threshold int, relayAddr, runID, outDir string, timeout, startDelay time.Duration) {
	if id < 1 || id > n {
		log.Fatalf("party id %d outside range 1..%d", id, n)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir out-dir failed: %v", err)
	}

	pIDs := makeDeterministicPartyIDs(n)
	selfPID := pIDs[id-1]
	p2pCtx := tss.NewPeerContext(pIDs)

	params := tss.NewParameters(tss.S256(), p2pCtx, selfPID, len(pIDs), threshold)

	// Match the fast benchmark harness behavior.
	// This is for benchmarking/prototyping only, not production security.
	params.SetNoProofMod()
	params.SetNoProofFac()

	tssErrCh := make(chan *tss.Error, 8)
	outCh := make(chan tss.Message, n*32)
	endCh := make(chan *kg.LocalPartySaveData, 1)

	party := kg.NewLocalParty(params, outCh, endCh).(*kg.LocalParty)

	conn, enc, dec := connect(relayAddr)
	defer conn.Close()

	register(enc, runID, id)

	msgCh, readErrCh := startReader(dec)
	startCh := make(chan struct{}, 1)
	deadline := time.After(timeout)

	registered := false
	startTimerSet := false
	started := false
	pending := make([]Envelope, 0)

	var sentMessages int
	var sentBytes int
	var receivedMessages int
	var receivedBytes int
	var keygenStart time.Time

	processInbound := func(env Envelope) {
		if env.Type != "tss" {
			return
		}
		if env.From < 1 || env.From > n {
			log.Fatalf("party=%d received invalid from=%d", id, env.From)
		}

		wireBytes, err := base64.StdEncoding.DecodeString(env.Payload)
		if err != nil {
			log.Fatalf("party=%d base64 decode failed from=%d: %v", id, env.From, err)
		}

		fromPID := pIDs[env.From-1]
		isBroadcast := env.To == 0

		ok, tssErr := party.UpdateFromBytes(wireBytes, fromPID, isBroadcast)
		if tssErr != nil {
			log.Fatalf("party=%d UpdateFromBytes failed from=%d broadcast=%v err=%v", id, env.From, isBroadcast, tssErr)
		}

		receivedMessages++
		receivedBytes += len(wireBytes)

		fmt.Printf("RECV_TSS id=%d from=%d to=%d broadcast=%v bytes=%d ok=%v\n",
			id, env.From, env.To, isBroadcast, len(wireBytes), ok)
	}

	for {
		select {
		case env := <-msgCh:
			if env.Type == "registered" {
				registered = true
				fmt.Printf("REGISTERED id=%d run=%s start_delay=%s\n", id, runID, startDelay)

				if !startTimerSet {
					startTimerSet = true
					go func() {
						time.Sleep(startDelay)
						startCh <- struct{}{}
					}()
				}
				continue
			}

			if env.Type == "tss" {
				if !started {
					pending = append(pending, env)
					continue
				}
				processInbound(env)
			}

		case <-startCh:
			if started {
				continue
			}
			started = true

			keygenStart = time.Now()

			fmt.Printf("START_KEYGEN id=%d n=%d t=%d self_index=%d self_moniker=%s\n",
				id, n, threshold, selfPID.Index, selfPID.Moniker)

			go func() {
				if err := party.Start(); err != nil {
					tssErrCh <- err
				}
			}()

			for _, env := range pending {
				processInbound(env)
			}
			pending = nil

		case msg := <-outCh:
			wireBytes, _, err := msg.WireBytes()
			if err != nil {
				log.Fatalf("party=%d WireBytes failed: %v", id, err)
			}

			toID := 0
			if dest := msg.GetTo(); dest != nil {
				toID = partyIDToInt(dest[0])
			}

			env := Envelope{
				Type:            "tss",
				RunID:           runID,
				From:            id,
				To:              toID,
				Payload:         base64.StdEncoding.EncodeToString(wireBytes),
				SentAtUnixNanos: time.Now().UnixNano(),
			}

			if err := enc.Encode(env); err != nil {
				log.Fatalf("party=%d send tss failed: %v", id, err)
			}

			sentMessages++
			sentBytes += len(wireBytes)

			fmt.Printf("SEND_TSS id=%d to=%d broadcast=%v bytes=%d type=%s\n",
				id, toID, msg.IsBroadcast(), len(wireBytes), msg.Type())

		case save := <-endCh:
			keygenElapsed := time.Since(keygenStart)
			writeKeygenOutput(outDir, id, n, threshold, runID, save, keygenElapsed, sentMessages, sentBytes, receivedMessages, receivedBytes)

			fmt.Printf("KEYGEN_OK id=%d n=%d t=%d keygen_ms=%d sent_messages=%d sent_bytes=%d received_messages=%d received_bytes=%d out_dir=%s\n",
				id, n, threshold, keygenElapsed.Milliseconds(), sentMessages, sentBytes, receivedMessages, receivedBytes, outDir)
			return

		case err := <-tssErrCh:
			fmt.Fprintf(os.Stderr, "KEYGEN_TSS_ERROR id=%d err=%v\n", id, err)
			os.Exit(1)

		case err := <-readErrCh:
			fmt.Fprintf(os.Stderr, "KEYGEN_RELAY_READ_ERROR id=%d err=%v\n", id, err)
			os.Exit(1)

		case <-deadline:
			fmt.Fprintf(os.Stderr, "KEYGEN_TIMEOUT id=%d registered=%v started=%v pending=%d sent=%d received=%d\n",
				id, registered, started, len(pending), sentMessages, receivedMessages)
			os.Exit(2)
		}
	}
}

func writeKeygenOutput(outDir string, id, n, threshold int, runID string, save *kg.LocalPartySaveData, keygenElapsed time.Duration, sentMessages, sentBytes, receivedMessages, receivedBytes int) {
	rawPath := filepath.Join(outDir, fmt.Sprintf("keygen_save_party_%02d.json", id))
	summaryPath := filepath.Join(outDir, fmt.Sprintf("keygen_summary_party_%02d.json", id))

	if raw, err := json.MarshalIndent(save, "", "  "); err == nil {
		_ = os.WriteFile(rawPath, raw, 0o644)
	} else {
		_ = os.WriteFile(rawPath+".error.txt", []byte(err.Error()), 0o644)
	}

	originalIndex := -1
	if idx, err := save.OriginalIndex(); err == nil {
		originalIndex = idx
	}

	summary := map[string]interface{}{
		"run_id":            runID,
		"party_id":          id,
		"n":                 n,
		"t":                 threshold,
		"original_index":    originalIndex,
		"keygen_ms":         keygenElapsed.Milliseconds(),
		"sent_messages":     sentMessages,
		"sent_bytes":        sentBytes,
		"received_messages": receivedMessages,
		"received_bytes":    receivedBytes,
		"max_rss_kb":        maxRSSKB(),
		"saved_at":          time.Now().Format(time.RFC3339Nano),
	}

	if b, err := json.MarshalIndent(summary, "", "  "); err == nil {
		_ = os.WriteFile(summaryPath, b, 0o644)
	}
}

func maxRSSKB() int64 {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return -1
	}
	return usage.Maxrss
}

func parseSignerIDs(n, signerCount int, raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		if signerCount <= 0 {
			signerCount = n
		}
		if signerCount > n {
			return nil, fmt.Errorf("signer count %d cannot exceed n=%d", signerCount, n)
		}
		ids := make([]int, signerCount)
		for i := 1; i <= signerCount; i++ {
			ids[i-1] = i
		}
		return ids, nil
	}

	parts := strings.Split(raw, ",")
	seen := make(map[int]bool, len(parts))
	ids := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid signer id %q: %w", part, err)
		}
		if id < 1 || id > n {
			return nil, fmt.Errorf("signer id %d outside range 1..%d", id, n)
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate signer id %d", id)
		}

		seen[id] = true
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("empty signer id set")
	}

	sort.Ints(ids)

	if signerCount > 0 && len(ids) != signerCount {
		return nil, fmt.Errorf("signer id count %d does not match -signers %d", len(ids), signerCount)
	}

	return ids, nil
}

func containsSignerID(ids []int, id int) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func signerIDsCSV(ids []int) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(id))
	}
	return strings.Join(parts, ",")
}

func makeSignerPartyIDs(signerIDs []int) tss.SortedPartyIDs {
	maxID := signerIDs[len(signerIDs)-1]
	allPIDs := makeDeterministicPartyIDs(maxID)

	selected := make(tss.UnSortedPartyIDs, 0, len(signerIDs))
	for _, id := range signerIDs {
		selected = append(selected, allPIDs[id-1])
	}

	return tss.SortPartyIDs(selected)
}

func partyIDToInt(pid *tss.PartyID) int {
	id, err := strconv.Atoi(pid.Id)
	if err != nil {
		log.Fatalf("invalid party id in routing: id=%q err=%v", pid.Id, err)
	}
	return id
}

func findPartyIDByInt(pIDs tss.SortedPartyIDs, id int) *tss.PartyID {
	for _, pid := range pIDs {
		if partyIDToInt(pid) == id {
			return pid
		}
	}
	return nil
}

func loadKeygenSave(path string) kg.LocalPartySaveData {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read keygen save failed path=%s err=%v", path, err)
	}

	var save kg.LocalPartySaveData
	if err := json.Unmarshal(raw, &save); err != nil {
		log.Fatalf("unmarshal keygen save failed path=%s err=%v", path, err)
	}

	return save
}

func runSign(id, n, threshold, signerCount int, signerIDsRaw string, relayAddr, runID, outDir, keygenSavePath, msgValue string, timeout, startDelay time.Duration) {
	signerIDs, err := parseSignerIDs(n, signerCount, signerIDsRaw)
	if err != nil {
		log.Fatalf("invalid signer ids: %v", err)
	}
	signerCount = len(signerIDs)

	if !containsSignerID(signerIDs, id) {
		log.Fatalf("party id %d is not in signer set %s", id, signerIDsCSV(signerIDs))
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir out-dir failed: %v", err)
	}

	save := loadKeygenSave(keygenSavePath)

	if idx, err := save.OriginalIndex(); err == nil && idx != id-1 {
		log.Printf("warning: keygen save original_index=%d does not match party id=%d", idx, id)
	}

	signPIDs := makeSignerPartyIDs(signerIDs)
	selfPID := findPartyIDByInt(signPIDs, id)
	if selfPID == nil {
		log.Fatalf("party id %d not found in signer PartyID set %s", id, signerIDsCSV(signerIDs))
	}
	p2pCtx := tss.NewPeerContext(signPIDs)

	msgInt, ok := new(big.Int).SetString(msgValue, 10)
	if !ok {
		log.Fatalf("invalid -msg integer value: %s", msgValue)
	}

	params := tss.NewParameters(tss.S256(), p2pCtx, selfPID, len(signPIDs), threshold)

	tssErrCh := make(chan *tss.Error, 8)
	outCh := make(chan tss.Message, n*64)
	endCh := make(chan *common.SignatureData, 1)

	party := sg.NewLocalParty(msgInt, params, save, outCh, endCh).(*sg.LocalParty)

	conn, enc, dec := connect(relayAddr)
	defer conn.Close()

	register(enc, runID, id)

	msgCh, readErrCh := startReader(dec)
	startCh := make(chan struct{}, 1)
	deadline := time.After(timeout)

	registered := false
	startTimerSet := false
	started := false
	pending := make([]Envelope, 0)

	var sentMessages int
	var sentBytes int
	var receivedMessages int
	var receivedBytes int
	var signStart time.Time

	processInbound := func(env Envelope) {
		if env.Type != "tss" {
			return
		}
		if !containsSignerID(signerIDs, env.From) {
			log.Fatalf("party=%d received invalid from=%d signer_ids=%s", id, env.From, signerIDsCSV(signerIDs))
		}

		wireBytes, err := base64.StdEncoding.DecodeString(env.Payload)
		if err != nil {
			log.Fatalf("party=%d base64 decode failed from=%d: %v", id, env.From, err)
		}

		fromPID := findPartyIDByInt(signPIDs, env.From)
		if fromPID == nil {
			log.Fatalf("party=%d could not resolve from=%d in signer_ids=%s", id, env.From, signerIDsCSV(signerIDs))
		}
		isBroadcast := env.To == 0

		ok, tssErr := party.UpdateFromBytes(wireBytes, fromPID, isBroadcast)
		if tssErr != nil {
			log.Fatalf("party=%d signing UpdateFromBytes failed from=%d broadcast=%v err=%v", id, env.From, isBroadcast, tssErr)
		}

		receivedMessages++
		receivedBytes += len(wireBytes)

		fmt.Printf("RECV_SIGN_TSS id=%d from=%d to=%d broadcast=%v bytes=%d ok=%v\n",
			id, env.From, env.To, isBroadcast, len(wireBytes), ok)
	}

	for {
		select {
		case env := <-msgCh:
			if env.Type == "registered" {
				registered = true
				fmt.Printf("REGISTERED id=%d run=%s start_delay=%s\n", id, runID, startDelay)

				if !startTimerSet {
					startTimerSet = true
					go func() {
						time.Sleep(startDelay)
						startCh <- struct{}{}
					}()
				}
				continue
			}

			if env.Type == "tss" {
				if !started {
					pending = append(pending, env)
					continue
				}
				processInbound(env)
			}

		case <-startCh:
			if started {
				continue
			}
			started = true
			signStart = time.Now()

			fmt.Printf("START_SIGN id=%d n=%d t=%d signers=%d signer_ids=%s self_index=%d self_moniker=%s msg=%s keygen_save=%s\n",
				id, n, threshold, signerCount, signerIDsCSV(signerIDs), selfPID.Index, selfPID.Moniker, msgValue, keygenSavePath)

			go func() {
				if err := party.Start(); err != nil {
					tssErrCh <- err
				}
			}()

			for _, env := range pending {
				processInbound(env)
			}
			pending = nil

		case msg := <-outCh:
			wireBytes, _, err := msg.WireBytes()
			if err != nil {
				log.Fatalf("party=%d signing WireBytes failed: %v", id, err)
			}

			toID := 0
			if dest := msg.GetTo(); dest != nil {
				toID = partyIDToInt(dest[0])
			}

			env := Envelope{
				Type:            "tss",
				RunID:           runID,
				From:            id,
				To:              toID,
				Payload:         base64.StdEncoding.EncodeToString(wireBytes),
				SentAtUnixNanos: time.Now().UnixNano(),
			}

			if err := enc.Encode(env); err != nil {
				log.Fatalf("party=%d send signing tss failed: %v", id, err)
			}

			sentMessages++
			sentBytes += len(wireBytes)

			fmt.Printf("SEND_SIGN_TSS id=%d to=%d broadcast=%v bytes=%d type=%s\n",
				id, toID, msg.IsBroadcast(), len(wireBytes), msg.Type())

		case sig := <-endCh:
			signElapsed := time.Since(signStart)

			verifyOK := writeSignatureOutput(outDir, id, n, threshold, signerCount, runID, msgValue, save, sig, signElapsed, sentMessages, sentBytes, receivedMessages, receivedBytes)

			fmt.Printf("SIGN_OK id=%d n=%d t=%d signers=%d sign_ms=%d verify_ok=%v sent_messages=%d sent_bytes=%d received_messages=%d received_bytes=%d out_dir=%s\n",
				id, n, threshold, signerCount, signElapsed.Milliseconds(), verifyOK, sentMessages, sentBytes, receivedMessages, receivedBytes, outDir)
			return

		case err := <-tssErrCh:
			fmt.Fprintf(os.Stderr, "SIGN_TSS_ERROR id=%d err=%v\n", id, err)
			os.Exit(1)

		case err := <-readErrCh:
			fmt.Fprintf(os.Stderr, "SIGN_RELAY_READ_ERROR id=%d err=%v\n", id, err)
			os.Exit(1)

		case <-deadline:
			fmt.Fprintf(os.Stderr, "SIGN_TIMEOUT id=%d registered=%v started=%v pending=%d sent=%d received=%d\n",
				id, registered, started, len(pending), sentMessages, receivedMessages)
			os.Exit(2)
		}
	}
}

func writeSignatureOutput(outDir string, id, n, threshold, signerCount int, runID, msgValue string, save kg.LocalPartySaveData, sig *common.SignatureData, signElapsed time.Duration, sentMessages, sentBytes, receivedMessages, receivedBytes int) bool {
	rawPath := filepath.Join(outDir, fmt.Sprintf("signature_party_%02d.json", id))
	summaryPath := filepath.Join(outDir, fmt.Sprintf("signature_summary_party_%02d.json", id))

	if raw, err := json.MarshalIndent(sig, "", "  "); err == nil {
		_ = os.WriteFile(rawPath, raw, 0o644)
	} else {
		_ = os.WriteFile(rawPath+".error.txt", []byte(err.Error()), 0o644)
	}

	r := new(big.Int).SetBytes(sig.R)
	ss := new(big.Int).SetBytes(sig.S)

	verifyOK := false
	if save.ECDSAPub != nil {
		pk := ecdsa.PublicKey{
			Curve: tss.EC(),
			X:     save.ECDSAPub.X(),
			Y:     save.ECDSAPub.Y(),
		}
		verifyOK = ecdsa.Verify(&pk, sig.M, r, ss)
	}

	summary := map[string]interface{}{
		"run_id":            runID,
		"party_id":          id,
		"n":                 n,
		"t":                 threshold,
		"signer_count":      signerCount,
		"msg":               msgValue,
		"sign_ms":           signElapsed.Milliseconds(),
		"verify_ok":         verifyOK,
		"sent_messages":     sentMessages,
		"sent_bytes":        sentBytes,
		"received_messages": receivedMessages,
		"received_bytes":    receivedBytes,
		"max_rss_kb":        maxRSSKB(),
		"r":                 r.String(),
		"s":                 ss.String(),
		"saved_at":          time.Now().Format(time.RFC3339Nano),
	}

	if b, err := json.MarshalIndent(summary, "", "  "); err == nil {
		_ = os.WriteFile(summaryPath, b, 0o644)
	}

	return verifyOK
}
