package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

const TxGas = 1
const TxGasPriceDefault = 1
const TxFee = uint(50)

type State struct {
	Balances map[common.Address]uint 
	AccountToNonce map[common.Address]uint 

	dbFile *os.File 

	latestBlock Block 
	latestBlockHash Hash 
	hasGenesisBlock bool 

	miningDifficulty uint 

	HashCache map[string]int64 
	HeightCache map[uint64]int64
}

func NewStateFromDisk(dataDir string, miningDifficulty uint) (*State, error){
	err := InitDataDirIfNotExists(dataDir, []byte(genesisJson))
	if err != nil{
		return nil, err 
	}

	gen, err := loadGenesis(getGenesisJsonFilePath(dataDir)) 
	if err != nil {
		return nil, err 
	}

	balances := make(map[common.Address]uint)
	for account, balance := range gen.Balances{
		balances[account] = balance
	}
	
	accountToNonce := make(map[common.Address]uint) 

	dbFilepath := getBlocksDbFilePath(dataDir)
	f, err := os.OpenFile(dbFilepath, os.O_APPEND|os.O_RDWR, 0600) 
	if err != nil{
		return nil, err 
	}

	scanner := bufio.NewScanner(f)

	state := &State{
		balances, 
		accountToNonce, 
		f, 
		Block{}, 
		Hash{}, 
		false, 
		miningDifficulty, 
		map[string]int64{}, 
		map[uint64]int64{},
	}

	// File position 
	filePos := int64(0) 

	for scanner.Scan(){
		if err := scanner.Err(); err != nil{
			return nil, err 
		}

		blockFsJson := scanner.Bytes() 

		if len(blockFsJson) == 0{
			break
		}

		var blockFs BlockFS 
		err = json.Unmarshal(blockFsJson, &blockFs)
		if err != nil{
			return nil, err 
		}

		err = applyBlock(blockFs.Value, state)
		if err != nil{
			return nil, err 
		}

		// Set search caches 
		state.HashCache[blockFs.Key.Hex()] = filePos 
		state.HeightCache[blockFs.Value.Header.Number] = filePos 
		filePos += int64(len(blockFsJson)) + 1 

		state.latestBlock = blockFs.Value 
		state.latestBlockHash = blockFs.Key 
		state.hasGenesisBlock = true 
	}
	return state, nil 
}

func (s *State) AddBlocks(blocks []Block) error{
	for _, b := range blocks{
		_, err := s.AddBlock(b)
		if err != nil{
			return err
		}
	}
	return nil 
}

func (s *State) AddBlock(b Block) (Hash, error){
	pendingState := s.Copy() 

	err := applyBlock(b, &pendingState) 
	if err != nil{
		return Hash{}, err 
	}
	blockHash, err := b.Hash() 
	if err != nil{
		return Hash{}, err 
	}

	blockFs := BlockFS{blockHash, b} 

	blockFsJson, err := json.Marshal(blockFs) 
	if err != nil{
		return Hash{}, err 
	}

	fmt.Printf("\nPerssisting new block to disk:\n") 
	fmt.Printf("\t%s\n", blockFsJson) 

	fs, _ := s.dbFile.Stat() 
	filePos := fs.Size() + 1 

	_, err = s.dbFile.Write(append(blockFsJson, '\n')) 
	if err != nil{
		return Hash{}, err 
	}

	// Set search caches
	s.HashCache[blockFs.Key.Hex()] = filePos 
	s.HeightCache[blockFs.Value.Header.Number] = filePos 

	s.Balances = pendingState.Balances 
	s.AccountToNonce = pendingState.AccountToNonce
	s.latestBlockHash = blockHash 
	s.latestBlock = b 
	s.hasGenesisBlock = true 
	s.miningDifficulty = pendingState.miningDifficulty

	return blockHash, nil 
}

func (s *State) NextBlockNumber() uint64{
	if !s.hasGenesisBlock{
		return uint64(0)
	}

	return s.LatestBlock().Header.Number + 1
}

func (s *State) LatestBlock() Block{
	return s.latestBlock
}

func (s *State) LatestBlockHash() Hash{
	return s.latestBlockHash
}

func (s *State) GetNextAccountNonce(account common.Address) uint{
	return s.AccountToNonce[account] + 1 
}

func (s *State) ChangeMiningDifficulty(newDifficulty uint){
	s.miningDifficulty = newDifficulty
}

func (s *State) Copy() State{
	c := State{} 
	c.hasGenesisBlock = s.hasGenesisBlock 
	c.latestBlock = s.latestBlock 
	c.latestBlockHash = s.latestBlockHash 
	c.Balances = make(map[common.Address]uint) 
	c.AccountToNonce = make(map[common.Address]uint) 
	c.miningDifficulty = s.miningDifficulty 
	
	
	for acc, balance := range s.Balances{
		c.Balances[acc] = balance
	}

	for acc, nonce := range s.AccountToNonce{
		c.AccountToNonce[acc] = nonce 
	}

	return c 
}

func (s *State) Close() error{
	return s.dbFile.Close() 
}

// applyBlock verifies whether this block can be added to the blockchain 
// block meta data are verified as well as transactions within (are blanaces sufficient, etc) 
func applyBlock(b Block, s *State) error{
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1 

	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber{
		return fmt.Errorf("next expected block number must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number) 


	}

	if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash){
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	hash, err := b.Hash()
	if err != nil{
		return err
	}

	if !IsBlockHashValid(hash, s.miningDifficulty){
		return fmt.Errorf("invalid block hash %x", hash) 
	}

	err = applyTxs(b.Txs, s)  
	if err != nil{
		return err
	}

	s.Balances[b.Header.Miner] += BlockReward 
	s.Balances[b.Header.Miner] += b.GasReward()

	return nil 
}

func applyTxs(txs []SignedTx, s *State) error{
	sort.Slice(txs, func(i, j int) bool{
		return txs[i].Time < txs[j].Time
	})

	for _, tx := range txs{
		err := ApplyTx(tx, s)  
		if err != nil{
			return err 
		}
	}

	return nil 
}

func ApplyTx(tx SignedTx, s *State) error{
	err := ValidateTx(tx, s) 
	if err != nil{
		return err 
	}

	s.Balances[tx.From] -= tx.Cost() 
	s.Balances[tx.To] += tx.Value 

	s.AccountToNonce[tx.From] = tx.Nonce 

	return nil 
}

func ValidateTx(tx SignedTx, s *State) error{
	ok, err := tx.IsAuthentic() 
	if err != nil{
		return err 
	}

	if !ok{
		return fmt.Errorf("wrong TX. Sender is '%s' is forged", tx.From.String())
	}

	expectedNonce := s.GetNextAccountNonce(tx.From) 
	if tx.Nonce != expectedNonce{
		return fmt.Errorf("wrong Tx. Sender '%s' next nonce must be '%d', not '%d'", tx.From.String(), expectedNonce, tx.Nonce)
	}

	if tx.Gas != TxGas{
		return fmt.Errorf("insufficient Tx Gas %v. required: %v", tx.Gas, TxGas) 
	}
	if tx.GasPrice < TxGasPriceDefault{
		return fmt.Errorf("insufficient Tx gasPrice %v. required at least: %v", tx.GasPrice, TxGasPriceDefault)
	}

	if tx.Cost() > s.Balances[tx.From]{
		return fmt.Errorf("wrong TX. Sender '%s' balance is %d NEM. Tx cost is %d NEM", tx.From.String(), s.Balances[tx.From], tx.Cost)
	}
	return nil 
}