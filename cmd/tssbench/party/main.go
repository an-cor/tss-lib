package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type Envelope struct {
	Type             string `json:"type"`
	RunID            string `json:"run_id"`
	From             int    `json:"from"`
	To               int    `json:"to,omitempty"` // 0 means broadcast
	Payload          string `json:"payload,omitempty"`
	SentAtUnixNanos  int64  `json:"sent_at_unix_nanos,omitempty"`
}

func main() {
	id := flag.Int("id", 0, "party id")
	n := flag.Int("n", 0, "number of parties")
	relayAddr := flag.String("relay", "127.0.0.1:9100", "relay host:port")
	runID := flag.String("run", "manual", "run identifier")
	sendHello := flag.Bool("send", false, "send a hello broadcast after registration")
	to := flag.Int("to", 0, "destination party id; 0 means broadcast")
	expectHello := flag.Int("expect", 0, "number of hello messages expected before success")
	timeout := flag.Duration("timeout", 10*time.Second, "max time to wait")
	flag.Parse()

	if *id <= 0 {
		log.Fatalf("missing or invalid -id")
	}
	if *n <= 0 {
		log.Fatalf("missing or invalid -n")
	}

	conn, err := net.DialTimeout("tcp", *relayAddr, 5*time.Second)
	if err != nil {
		log.Fatalf("party=%d connect relay=%s failed: %v", *id, *relayAddr, err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	if err := enc.Encode(Envelope{
		Type:            "register",
		RunID:           *runID,
		From:            *id,
		Payload:         fmt.Sprintf("party-%d-register", *id),
		SentAtUnixNanos: time.Now().UnixNano(),
	}); err != nil {
		log.Fatalf("party=%d register failed: %v", *id, err)
	}

	msgCh := make(chan Envelope, 16)
	errCh := make(chan error, 1)

	go func() {
		for {
			var msg Envelope
			if err := dec.Decode(&msg); err != nil {
				errCh <- err
				return
			}
			msgCh <- msg
		}
	}()

	deadline := time.After(*timeout)

	registered := false
	helloSent := false
	helloReceived := 0

	for {
		select {
		case msg := <-msgCh:
			b, _ := json.Marshal(msg)
			fmt.Printf("RECV party=%d msg=%s\n", *id, string(b))

			if msg.Type == "registered" {
				registered = true

				if *sendHello && !helloSent {
					out := Envelope{
						Type:            "hello",
						RunID:           *runID,
						From:            *id,
						To:              *to,
						Payload:         fmt.Sprintf("hello from party %d of %d", *id, *n),
						SentAtUnixNanos: time.Now().UnixNano(),
					}

					if err := enc.Encode(out); err != nil {
						log.Fatalf("party=%d send hello failed: %v", *id, err)
					}

					fmt.Printf("SENT party=%d type=hello to=%d\n", *id, *to)
					helloSent = true
				}
			}

			if msg.Type == "hello" {
				helloReceived++
			}

			if registered && (!*sendHello || helloSent) && helloReceived >= *expectHello {
				fmt.Printf("PARTY_OK id=%d registered=%v hello_sent=%v hello_received=%d expect=%d\n",
					*id, registered, helloSent, helloReceived, *expectHello)
				return
			}

		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "PARTY_ERROR id=%d err=%v\n", *id, err)
			os.Exit(1)

		case <-deadline:
			fmt.Fprintf(os.Stderr, "PARTY_TIMEOUT id=%d registered=%v hello_sent=%v hello_received=%d expect=%d\n",
				*id, registered, helloSent, helloReceived, *expectHello)
			os.Exit(2)
		}
	}
}
