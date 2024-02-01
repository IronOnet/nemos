package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/ethereum/go-ethereum/common"

	"github.com/irononet/nemos/core"
)

const DefaultBootstrapIp = "node.nemos.chain.root"

const DefaultBootstrapAcc = "0x09ee50f2f37fcba1845de6fe5c762e83e65e755c"
const DefaultMiner = "0x00000000000000000000000000000000000000"
const DefaultIP = "127.0.0.1"
const HttpSSLPort = 443
const endpointStatus = "/node/status"

const endpointSync = "/node/sync"
const endpointSyncQueryKeyFromBlock = "fromblock"

const endpointAddPeer = "/node/peer"
const endpointAddPeerQueryKeyIP = "ip"
const endpointAddPeerQueryKeyPort = "port"
const endpointAddPeerQueryKeyMiner = "miner"
const endpointAddPeerQueryKeyVersion = "version"

const endpointBlockByNumberOrHash = "/block/"
const endpointMempoolViewer = "/mempool"

const miningIntervalSeconds = 10
const DefaultMiningDifficulty = 3

type PeerNode struct {
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	IsBootstrap bool           `json:"is_bootstrap"`
	Account     common.Address `json:"account"`
	NodeVersion string         `json:"node_version"`

	connected bool
}

func (pn PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func (pn PeerNode) ApiProtocol() string {
	if pn.Port == HttpSSLPort {
		return "https"
	}
	return "http"
}

type Node struct {
	dataDir string
	info    PeerNode

	state *core.State

	pendingState    *core.State
	knownPeers      map[string]PeerNode
	pendingTxs      map[string]core.SignedTx
	archivedTx      map[string]core.SignedTx
	newSyncedBlocks chan core.Block
	newPendingTxs   chan core.SignedTx
	nodeVersion     string

	miningDifficulty uint
	isMining         bool
}

func New(dataDir string, ip string, port uint64, acc common.Address, bootstrap PeerNode, version string, miningDifficulty uint) *Node {
	knownPeers := make(map[string]PeerNode)

	n := &Node{
		dataDir:          dataDir,
		info:             NewPeerNode(ip, port, false, acc, true, version),
		knownPeers:       knownPeers,
		pendingTxs:       make(map[string]core.SignedTx),
		archivedTx:       make(map[string]core.SignedTx),
		newSyncedBlocks:  make(chan core.Block),
		newPendingTxs:    make(chan core.SignedTx, 10000),
		nodeVersion:      version,
		isMining:         false,
		miningDifficulty: miningDifficulty,
	}

	n.AddPeer(bootstrap)

	return n
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, acc common.Address, connected bool, version string) PeerNode {
	return PeerNode{ip, port, isBootstrap, acc, version, connected}
}

func (n *Node) Run(ctx context.Context, isSSLDisabled bool, sslEmail string) error {
	fmt.Println(fmt.Sprintf("Listening on: %s:%d", n.info.IP, n.info.Port))

	state, err := core.NewStateFromDisk(n.dataDir, n.miningDifficulty)
	if err != nil {
		return err
	}

	defer state.Close()

	n.state = state

	pendingState := state.Copy()
	n.pendingState = &pendingState

	fmt.Println("Blockchain state:")
	fmt.Printf(" - height: %d\n", n.state.LatestBlock().Header.Number)
	fmt.Printf(" - hash: %s\n", n.state.LatestBlockHash().Hex())

	go n.sync(ctx)
	go n.mine(ctx)

	return n.serveHttp(ctx, isSSLDisabled, sslEmail)
}

func (n *Node) LatestBlockHash() core.Hash {
	return n.state.LatestBlockHash()
}

func (n *Node) serveHttp(ctx context.Context, isSSLDisabled bool, sslEmail string) error {
	handler := http.NewServeMux()

	handler.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalanceHandler(w, r, n.state)
	})

	handler.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, n)
	})

	handler.HandleFunc(endpointStatus, func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, n)
	})

	handler.HandleFunc(endpointSync, func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})

	handler.HandleFunc(endpointAddPeer, func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	handler.HandleFunc(endpointBlockByNumberOrHash, func(w http.ResponseWriter, r *http.Request) {
		blockByNumberOrHash(w, r, n)
	})

	handler.HandleFunc(endpointMempoolViewer, func(w http.ResponseWriter, r *http.Request) {
		mempoolViewer(w, r, n.pendingTxs)
	})

	if isSSLDisabled {
		server := &http.Server{Addr: fmt.Sprintf(":%d", n.info.Port), Handler: handler}

		go func() {
			<-ctx.Done()
			_ = server.Close()
		}()

		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			return err
		}

		return nil
	} else {
		certmagic.DefaultACME.Email = sslEmail

		return certmagic.HTTPS([]string{n.info.IP}, handler)
	}
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTxs) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTxs(miningCtx)
					if err != nil {
						fmt.Errorf("error: %s\n", err)
					}

					n.isMining = false
				}
			}()

		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next block '%s' faster :(\n", blockHash.Hex())

				n.removeMinedPendingTxs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTxs(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Account,
		n.getPendingTXsAsArray(),
	)

	minedBlock, err := Mine(ctx, blockToMine, n.miningDifficulty)
	if err != nil {
		return err
	}

	n.removeMinedPendingTxs(minedBlock)

	err = n.addBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTxs(block core.Block) {
	if len(block.Txs) > 0 && len(n.pendingTxs) > 0 {
		fmt.Println("updating in-memory pending Txs pool:")
	}

	for _, tx := range block.Txs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTxs[txHash.Hex()]; exists {
			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())

			n.archivedTx[txHash.Hex()] = tx
			delete(n.pendingTxs, txHash.Hex())
		}
	}
}

func (n *Node) ChangeMiningDifficulty(newDifficulty uint) {
	n.miningDifficulty = newDifficulty
	n.state.ChangeMiningDifficulty(newDifficulty)
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnwonPeer(peer PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]
	return isKnownPeer
}

func (n *Node) AddPendingTX(tx core.SignedTx, fromPeer PeerNode) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	txJson, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	err = n.validateTxBeforeAddingToMempool(tx)
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTxs[txHash.Hex()]
	_, isArchived := n.archivedTx[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		fmt.Printf("Added peding TX %s from Peer %s\n", txJson, fromPeer.TcpAddress())
		n.pendingTxs[txHash.Hex()] = tx
		n.newPendingTxs <- tx
	}

	return nil
}

func (n *Node) addBlock(block core.Block) error {
	_, err := n.state.AddBlock(block)
	if err != nil {
		return err
	}

	pendingState := n.state.Copy()
	n.pendingState = &pendingState

	return nil
}

func (n *Node) validateTxBeforeAddingToMempool(tx core.SignedTx) error {
	return core.ApplyTx(tx, n.pendingState)
}

func (n *Node) getPendingTXsAsArray() []core.SignedTx {
	txs := make([]core.SignedTx, len(n.pendingTxs))

	i := 0
	for _, tx := range n.pendingTxs {
		txs[i] = tx
		i++
	}
	return txs
}
