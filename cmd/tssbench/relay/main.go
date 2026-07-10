package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Envelope struct {
	Type          string `json:"type"`
	RunID         string `json:"run_id"`
	From          int    `json:"from"`
	To            int    `json:"to,omitempty"` // 0 means broadcast
	Payload       string `json:"payload,omitempty"`
	SentAtUnixNanos int64 `json:"sent_at_unix_nanos,omitempty"`
}

type Client struct {
	id   int
	conn net.Conn
	enc  *json.Encoder
	mu   sync.Mutex
}

func (c *Client) send(msg Envelope) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enc.Encode(msg)
}

type Relay struct {
	mu      sync.RWMutex
	clients map[int]*Client
}

func NewRelay() *Relay {
	return &Relay{
		clients: make(map[int]*Client),
	}
}

func (r *Relay) register(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if old, ok := r.clients[c.id]; ok {
		_ = old.conn.Close()
	}

	r.clients[c.id] = c
	log.Printf("registered party=%d remote=%s", c.id, c.conn.RemoteAddr())
}

func (r *Relay) unregister(id int, conn net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.clients[id]
	if ok && current.conn == conn {
		delete(r.clients, id)
		log.Printf("unregistered party=%d", id)
	}
}

func (r *Relay) route(msg Envelope) {
	if msg.To == 0 {
		r.broadcast(msg)
		return
	}

	r.mu.RLock()
	target := r.clients[msg.To]
	r.mu.RUnlock()

	if target == nil {
		log.Printf("drop message: target party=%d not registered; from=%d type=%s", msg.To, msg.From, msg.Type)
		return
	}

	if err := target.send(msg); err != nil {
		log.Printf("send failed: to=%d from=%d type=%s err=%v", msg.To, msg.From, msg.Type, err)
	}
}

func (r *Relay) broadcast(msg Envelope) {
	r.mu.RLock()
	targets := make([]*Client, 0, len(r.clients))
	for id, c := range r.clients {
		if id == msg.From {
			continue
		}
		targets = append(targets, c)
	}
	r.mu.RUnlock()

	for _, target := range targets {
		if err := target.send(msg); err != nil {
			log.Printf("broadcast send failed: to=%d from=%d type=%s err=%v", target.id, msg.From, msg.Type, err)
		}
	}
}

func handleConn(conn net.Conn, relay *Relay, runID string) {
	defer conn.Close()

	dec := json.NewDecoder(conn)

	var registeredID int

	for {
		var msg Envelope
		if err := dec.Decode(&msg); err != nil {
			if err != io.EOF {
				log.Printf("decode error remote=%s err=%v", conn.RemoteAddr(), err)
			}
			if registeredID != 0 {
				relay.unregister(registeredID, conn)
			}
			return
		}

		if msg.RunID == "" {
			msg.RunID = runID
		}
		if msg.SentAtUnixNanos == 0 {
			msg.SentAtUnixNanos = time.Now().UnixNano()
		}

		if msg.Type == "register" {
			if msg.From <= 0 {
				log.Printf("invalid register message from=%d remote=%s", msg.From, conn.RemoteAddr())
				continue
			}

			client := &Client{
				id:   msg.From,
				conn: conn,
				enc:  json.NewEncoder(conn),
			}
			registeredID = msg.From
			relay.register(client)

			_ = client.send(Envelope{
				Type:    "registered",
				RunID:   msg.RunID,
				From:    0,
				To:      msg.From,
				Payload: "ok",
				SentAtUnixNanos: time.Now().UnixNano(),
			})
			continue
		}

		if registeredID == 0 {
			log.Printf("drop unregistered message remote=%s type=%s", conn.RemoteAddr(), msg.Type)
			continue
		}

		log.Printf("route run=%s type=%s from=%d to=%d payload_bytes=%d", msg.RunID, msg.Type, msg.From, msg.To, len(msg.Payload))
		relay.route(msg)
	}
}

func main() {
	listenAddr := flag.String("listen", "0.0.0.0:9100", "TCP listen address")
	runID := flag.String("run", "manual", "run identifier for logs/messages")
	flag.Parse()

	log.Printf("starting tssbench relay listen=%s run=%s", *listenAddr, *runID)

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	relay := NewRelay()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept failed: %v", err)
			continue
		}

		go handleConn(conn, relay, *runID)
	}
}
