package node  

import (
	"context" 
	"fmt" 
	"math/rand" 
	"time" 

	"github.com/ethereum/go-ethereum/common" 
	"github.com/irononet/nemos/core"
)

type PendingBlock struct{
	parent core.Hash 
	number uint64 
	time uint64 
	miner common.Address 
	txs []core.SignedTx
}

func NewPendingBlock(parent core.Hash, number uint64, miner common.Address, txs []core.SignedTx) PendingBlock{
	return PendingBlock{parent, number, uint64(time.Now().Unix()), miner, txs}
}

func Mine(ctx context.Context, pb PendingBlock, miningDifficulty uint) (core.Block, error){
	if len(pb.txs) == 0{
		return core.Block{}, fmt.Errorf("mining empty blocks is not allowed") 
	}

	start := time.Now() 
	attempt := 0 
	var block core.Block 
	var hash core.Hash 
	var nonce uint32 

	for !core.IsBlockHashValid(hash, miningDifficulty){
		select{
		case <-ctx.Done(): 
			fmt.Println("mining cancelled!") 
			return core.Block{}, fmt.Errorf("mining cancelled. %s", ctx.Err()) 
		default: 
		}

		attempt++ 
		nonce = generateNonce() 

		if attempt%1000000 == 0 || attempt ==  1{
			fmt.Printf("mining %d pending Txs. Attempt: %d\n", len(pb.txs), attempt) 
		}
		block = core.NewBlock(pb.parent, pb.number, nonce, pb.time, pb.miner, pb.txs) 
		blockHash, err := block.Hash() 
		if err != nil{
			return core.Block{}, fmt.Errorf("couldn't mine block. %s", err.Error()) 
		}

		hash = blockHash
	}

	fmt.Printf("\nMined new block '%x' using Pow \n", hash) 
	fmt.Printf("\tHeight: '%v'\n", block.Header.Number) 
	fmt.Printf("\tNonce: '%v'\n", block.Header.Nonce) 
	fmt.Printf("\tCreated: '%v'\n", block.Header.Time) 
	fmt.Printf("\tMiner: '%v'\n", block.Header.Miner.String()) 
	fmt.Printf("\tParent: '%v'\n\n", block.Header.Parent.Hex()) 

	fmt.Printf("\tAttempt: '%v'\n", attempt) 
	fmt.Printf("\tTime: %s\n\n", time.Since(start)) 

	return block, nil 
}

func generateNonce() uint32{
	rand.Seed(time.Now().UTC().UnixNano()) 

	return rand.Uint32()
}