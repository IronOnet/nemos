package node

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/irononet/nemos/core"
)

func (n *Node) sync(ctx context.Context) error {
	n.doSync()

	ticker := time.NewTicker(45 * time.Second)

	for {
		select {
		case <-ticker.C:
			n.doSync()
		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doSync() {
	for _, peer := range n.knownPeers {
		if n.info.IP == peer.IP && n.info.Port == peer.Port {
			continue
		}

		if peer.IP == "" {
			continue
		}

		fmt.Printf("searching for new peers and their blocks and peers: '%s'\n", peer.TcpAddress())

		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("error: %s\n", err)
			fmt.Printf("peer '%s' was removed from knownpeers\n", peer.TcpAddress())

			n.RemovePeer(peer)
			continue
		}

		err = n.joinKnownPeers(peer)
		if err != nil {
			fmt.Errorf("error: %s\n", err)
			continue
		}

		err = n.syncBlocks(peer, status)
		if err != nil {
			fmt.Printf("error: %s\n", err)
			continue
		}

		err = n.syncKnownPeers(status)
		if err != nil {
			fmt.Printf("error: %s\n", err)
			continue
		}

		err = n.syncPendingTXs(peer, status.PendingTxs)
		if err != nil {
			fmt.Printf("error: %s\n", err)
			continue
		}
	}
}

func (n *Node) syncBlocks(peer PeerNode, status StatusRes) error {
	localBlockNumber := n.state.LatestBlock().Header.Number

	if status.Hash.IsEmpty() {
		return nil
	}

	// If the peer has less blocks than us, ignore it
	if status.Number < localBlockNumber {
		return nil
	}

	// If it's the genesis block and we already synced it, ignore it
	if status.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	// display found 1 new block if we sync the genesis block 0
	newBlocksCount := status.Number - localBlockNumber
	if localBlockNumber == 0 && status.Number == 0 {
		newBlocksCount = 1
	}
	fmt.Printf("found %d new blocks from peer %s\n", newBlocksCount, peer.TcpAddress())
	blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash())
	if err != nil {
		return err
	}

	for _, block := range blocks {
		err = n.addBlock(block)
		if err != nil {
			return err
		}

		n.newSyncedBlocks <- block
	}

	return nil
}

func (n *Node) syncKnownPeers(status StatusRes) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnwonPeer(statusPeer) {
			fmt.Printf("found new peer %s\n", statusPeer.TcpAddress())
			n.AddPeer(statusPeer)
		}
	}
	return nil
}

func (n *Node) syncPendingTXs(peer PeerNode, txs []core.SignedTx) error {
	for _, tx := range txs {
		err := n.AddPendingTX(tx, peer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) joinKnownPeers(peer PeerNode) error {
	if peer.connected {
		return nil
	}

	p_url := fmt.Sprintf(
		"%s://%s%s?%s=%s&%s=%d&%s=%s&%s=%s",
		peer.ApiProtocol(),
		peer.TcpAddress(),
		endpointAddPeer,
		endpointAddPeerQueryKeyIP,
		n.info.IP,
		endpointAddPeerQueryKeyPort,
		n.info.Port,
		endpointAddPeerQueryKeyMiner,
		n.info.Account.String(),
		endpointAddPeerQueryKeyVersion,
		url.QueryEscape(n.info.NodeVersion),
	)

	res, err := http.Get(p_url)
	if err != nil {
		return err
	}

	addPeersRes := AddPeerRes{}
	err = readRes(res, &addPeersRes)
	if err != nil {
		return err
	}
	if addPeersRes.Error != "" {
		return fmt.Errorf(addPeersRes.Error)
	}

	knownPeer := n.knownPeers[peer.TcpAddress()]
	knownPeer.connected = addPeersRes.Success

	n.AddPeer(knownPeer)
	if !addPeersRes.Success {
		return fmt.Errorf("unable to join known peers of '%s'", peer.TcpAddress())
	}
	return nil
}

func queryPeerStatus(peer PeerNode) (StatusRes, error) {
	url := fmt.Sprintf("%s://%s%s", peer.ApiProtocol(), peer.TcpAddress(), endpointStatus)
	res, err := http.Get(url)
	if err != nil {
		return StatusRes{}, err
	}

	statusRes := StatusRes{}
	err = readRes(res, &statusRes)
	if err != nil {
		return StatusRes{}, err
	}

	return statusRes, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock core.Hash) ([]core.Block, error) {
	fmt.Printf("importing blocks from peer %s...\n", peer.TcpAddress())

	url := fmt.Sprintf(
		"%s://%s%s?%s=%s",
		peer.ApiProtocol(),
		peer.TcpAddress(),
		endpointSync,
		endpointSyncQueryKeyFromBlock,
		fromBlock.Hex(),
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	syncRes := SyncRes{}
	err = readRes(res, &syncRes)
	if err != nil {
		return nil, err
	}

	return syncRes.Blocks, nil
}
