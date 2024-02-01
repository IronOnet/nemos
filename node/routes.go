package node

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/irononet/nemos/core"
	"github.com/irononet/nemos/wallet"
)

type ErrRes struct {
	Error string `json:"error"`
}

type BalanceRes struct {
	Hash     core.Hash               `json:"block_hash"`
	Balances map[common.Address]uint `json:"balances"`
}

type TxAddReq struct {
	From     string `json:"from"`
	FromPwd  string `json:"from_pwd"`
	To       string `json:"to"`
	Gas      uint   `json:"gas"`
	GasPrice uint   `json:"gas_price"`
	Value    uint   `json:"value"`
	Data     string `json:"data"`
}

type TxAddress struct {
	Success bool `json:"success"`
}

type StatusRes struct {
	Hash        core.Hash           `json:"block_hash"`
	Number      uint64              `json:"block_number"`
	KnownPeers  map[string]PeerNode `json:"peers_known"`
	PendingTxs  []core.SignedTx     `json:"pending_txs"`
	NodeVersion string              `json:"node_version"`
	Account     common.Address      `json:"account"`
}

type SyncRes struct {
	Blocks []core.Block `json:"blocks"`
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func listBalanceHandler(w http.ResponseWriter, r *http.Request, state *core.State) {
	enableCors(&w)

	writeRes(w, BalanceRes{state.LatestBlockHash(), state.Balances})
}

func txAddHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := TxAddReq{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	from := core.NewAccount(req.From)

	if from.String() == common.HexToAddress("").String() {
		writeErrRes(w, fmt.Errorf("%s is an invalid 'from' sender", from.String()))
		return
	}

	if req.FromPwd == "" {
		writeErrRes(w, fmt.Errorf("password to decrypt the %s account is required. 'from_pwd' is empty", from.String()))
		return
	}

	nonce := node.state.GetNextAccountNonce(from)
	tx := core.NewTx(from, core.NewAccount(req.To), req.Gas, req.GasPrice, req.Value, nonce, req.Data)

	signedTx, err := wallet.SignWithKeystoreAccount(tx, from, req.FromPwd, wallet.GetKeystoreDirPath(node.dataDir))

	if err != nil {
		writeErrRes(w, err)
		return
	}

	err = node.AddPendingTX(signedTx, node.info)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, TxAddress{Success: true})
}

func statusHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	enableCors(&w)

	res := StatusRes{
		Hash:        node.state.LatestBlockHash(),
		Number:      node.state.LatestBlock().Header.Number,
		KnownPeers:  node.knownPeers,
		PendingTxs:  node.getPendingTXsAsArray(),
		NodeVersion: node.nodeVersion,
		Account:     core.NewAccount(node.info.Account.String()),
	}

	writeRes(w, res)
}

func syncHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	reqHash := r.URL.Query().Get(endpointSyncQueryKeyFromBlock)

	hash := core.Hash{}
	err := hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrRes(w, err)
		return
	}

	blocks, err := core.GetBlockAfter(hash, node.dataDir)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, SyncRes{Blocks: blocks})
}

func addPeerHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	peerIP := r.URL.Query().Get(endpointAddPeerQueryKeyIP)
	peerPortRaw := r.URL.Query().Get(endpointAddPeerQueryKeyPort)
	minerRaw := r.URL.Query().Get(endpointAddPeerQueryKeyMiner)
	versionRaw := r.URL.Query().Get(endpointAddPeerQueryKeyVersion)

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, core.NewAccount(minerRaw), true, versionRaw)
	node.AddPeer(peer)

	fmt.Println("peer '%s' was added into knownpeers\n", peer.TcpAddress())

	writeRes(w, AddPeerRes{true, ""})
}

func blockByNumberOrHash(w http.ResponseWriter, r *http.Request, node *Node) {
	enableCors(&w)

	errorParamsRequired := errors.New("height or hash param is required")
	params := strings.Split(r.URL.Path, "/")[1:]
	if len(params) < 2 {
		writeErrRes(w, errorParamsRequired)
		return
	}

	p := strings.TrimSpace(params[1])
	if len(p) == 0 {
		writeErrRes(w, errorParamsRequired)
		return
	}

	hsh := ""
	height, err := strconv.ParseUint(p, 10, 64)
	if err != nil {
		hsh = p
	}

	block, err := core.GetBlockByHeightOrHash(node.state, height, hsh, node.dataDir)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, block)
}

func mempoolViewer(w http.ResponseWriter, r *http.Request, txs map[string]core.SignedTx) {
	enableCors(&w)
	writeRes(w, txs)
}
