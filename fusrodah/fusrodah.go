package fusrodah

/*
	This program use modified go-ethereum library (https://github.com/sonm-io/go-ethereum)
	Author Sonm.io team (@sonm-io on GitHub)
	Copyright 2017
*/

import (
	"crypto/ecdsa"
	"fmt"
	"os"

	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/whisper/whisperv2"
	"github.com/sonm-io/fusrodah/util"
)

type serverState int

const (
	serverStateStopped = 0
	serverStateRunning = 1
	maxPeers           = 80
)

var (
	errServerNotRunning = errors.New("Server is not running")
)

type Fusrodah struct {
	Prv     *ecdsa.PrivateKey
	asymKey *ecdsa.PrivateKey

	cfg           p2p.Config
	p2pServer     p2p.Server
	whisperServer *whisperv2.Whisper

	p2pServerStatus     string
	whisperServerStatus serverState

	Enode string
	Port  string
}

func NewServer(prv *ecdsa.PrivateKey, port string, enode string) *Fusrodah {
	if prv == nil {
		prv, _ = crypto.GenerateKey()
	}

	shh := whisperv2.New()

	return &Fusrodah{
		Prv:                 prv,
		Port:                port,
		Enode:               enode,
		whisperServer:       shh,
		asymKey:             shh.NewIdentity(),
		whisperServerStatus: serverStateStopped,
	}
}

func (fusrodah *Fusrodah) GetMsgPrivateKey() *ecdsa.PrivateKey {
	return fusrodah.asymKey
}

func (fusrodah *Fusrodah) GetMsgPublicKey() *ecdsa.PublicKey {
	return &fusrodah.asymKey.PublicKey
}

// Start start whisper server
// private key is needed
func (fusrodah *Fusrodah) Start() {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(5), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
	// Creates new instance of whisper protocol entity. NOTE - using whisper v.2 (not v5)

	var peers []*discover.Node
	peer := discover.MustParseNode(fusrodah.Enode)
	peers = append(peers, peer)

	// Configuration to running p2p server. Configuration values can't be modified after launch.
	fusrodah.p2pServer = p2p.Server{
		Config: p2p.Config{
			PrivateKey: fusrodah.Prv,
			MaxPeers:   maxPeers,
			Name:       common.MakeName("wnode", "2.0"),
			// here we can define what additional protocols will be used *above* p2p server.
			Protocols:      fusrodah.whisperServer.Protocols(),
			ListenAddr:     util.GetLocalIP() + fusrodah.Port,
			NAT:            nat.Any(),
			BootstrapNodes: peers,
			StaticNodes:    peers,
			TrustedNodes:   peers,
		},
	}

	// Starting p2p server
	if err := fusrodah.p2pServer.Start(); err != nil {
		fmt.Println("could not start server:", err)
		os.Exit(1)
	}

	// Starting whisper protocol on running p2p server.
	// NOTE whisper *should* be started automatically but it is not happening... possible BUG in go-ethereum.
	if err := fusrodah.whisperServer.Start(&fusrodah.p2pServer); err != nil {
		fmt.Println("could not start server:", err)
		os.Exit(1)
	}

	log.Info("my public key", "key", common.ToHex(crypto.FromECDSAPub(&fusrodah.asymKey.PublicKey)))
	fusrodah.whisperServerStatus = serverStateRunning
}

// Stop stops whisper and p2p servers
func (fusrodah *Fusrodah) Stop() {
	fusrodah.whisperServer.Stop()
	fusrodah.p2pServer.Stop()
}

// getFilterTopics Creating new filters for a few topics.
// NOTE more info about filters in /whisperv2/filters.go
func (fusrodah *Fusrodah) getFilterTopics(data ...string) [][]whisperv2.Topic {
	topics := whisperv2.NewFilterTopicsFromStringsFlat(data...)
	return topics
}

// createEnvelop wraps message into envelope to transmit over the network.
//
// pow (Proof Of Work) controls how much time to spend on hashing the message,
// inherently controlling its priority through the network (smaller hash, bigger
// priority).
//
// The user can control the amount of identity, privacy and encryption through
// the options parameter as follows:
//   - options.From == nil && options.To == nil: anonymous broadcast
//   - options.From != nil && options.To == nil: signed broadcast (known sender)
//   - options.From == nil && options.To != nil: encrypted anonymous message
//   - options.From != nil && options.To != nil: encrypted signed message
func (fusrodah *Fusrodah) createEnvelop(message *whisperv2.Message, to *ecdsa.PublicKey, from *ecdsa.PrivateKey, topics []whisperv2.Topic) (*whisperv2.Envelope, error) {
	envelope, err := message.Wrap(whisperv2.DefaultPoW, whisperv2.Options{
		To:     to,
		From:   from, // Sign it
		Topics: topics,
		TTL:    whisperv2.DefaultTTL,
	})
	if err != nil {
		return nil, err
	}

	return envelope, nil
}

// isRunning check if Fusrodah server is running
func (fusrodah *Fusrodah) isRunning() bool {
	return fusrodah.whisperServerStatus == serverStateRunning
}

func (fusrodah *Fusrodah) Send(payload string, anonymous bool, topics ...string) error {
	if !fusrodah.isRunning() {
		return errServerNotRunning
	}

	var from *ecdsa.PrivateKey
	if anonymous {
		from = nil
	} else {
		from = fusrodah.asymKey
	}

	opts := whisperv2.Options{
		From:   from,
		To:     nil,
		Topics: whisperv2.NewTopicsFromStrings(topics...),
		TTL:    whisperv2.DefaultTTL,
	}

	msg := whisperv2.NewMessage([]byte(payload))
	env, err := msg.Wrap(whisperv2.DefaultPoW, opts)
	if err != nil {
		log.Error("failed to wrap new message", "err", err)
		return err

	}

	err = fusrodah.whisperServer.Send(env)
	if err != nil {
		fmt.Printf("failed to send message: %v \n", err)
		return err
	}

	fmt.Println("message sent")
	return nil
}

func (fusrodah *Fusrodah) SendPrivateMsg(payload string, to *ecdsa.PublicKey, topics ...string) error {
	if !fusrodah.isRunning() {
		return errServerNotRunning
	}

	opts := whisperv2.Options{
		From:   fusrodah.asymKey,
		To:     to,
		Topics: whisperv2.NewTopicsFromStrings(topics...),
		TTL:    whisperv2.DefaultTTL,
	}

	msg := whisperv2.NewMessage([]byte(payload))

	env, err := msg.Wrap(0, opts)
	if err != nil {
		log.Error("failed to wrap new message", "err", err)
		return err

	}

	err = fusrodah.whisperServer.Send(env)
	if err != nil {
		fmt.Printf("failed to send message: %v \n", err)
		return err
	}

	return nil
}

// AddHandling adds register handler for messages with given keys and on given topics
func (fusrodah *Fusrodah) AddHandling(to *ecdsa.PublicKey, from *ecdsa.PublicKey, cb func(msg *whisperv2.Message), topics ...string) int {
	if !fusrodah.isRunning() {
		fusrodah.Start()
	}

	// add watcher with any topics
	id := fusrodah.whisperServer.Watch(whisperv2.Filter{
		// setting up filter by topic
		Topics: fusrodah.getFilterTopics(topics...),
		// setting up message handler
		Fn: cb,
		// settings up sender and recipient
		From: from,
		To:   to,
	})

	fmt.Printf("Filter installed: %d \r\n", id)
	return id
}

// RemoveHandling removes message handler by their id
func (fusrodah *Fusrodah) RemoveHandling(id int) {
	fusrodah.whisperServer.Unwatch(id)
	fmt.Printf("Filter uninstalled: %d \r\n", id)
}
