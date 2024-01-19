package core 

import (
	"crypto/elliptic" 
	"crypto/sha256" 
	"encoding/json" 
	"time" 

	"github.com/ethereum/go-ethereum/common" 
	"github.com/ethereum/go-ethereum/crypto"
)

func NewAccount(value string) common.Address{
	return common.HexToAddress(value)
}


type Tx struct{
	From common.Address `json:"from"`
	To common.Address `json:"to"`
	Gas uint `json:"gas"`
	GasPrice uint `json:"gas_price"`
	Value uint `json:"value"`
	Nonce uint `json:"nonce"`
	Data string `json:"data"`
	Time uint64 `json:"time"`
}

type SignedTx struct{
	Tx 
	Sig []byte `json:"signature"`
}

func NewTx(from, to common.Address, gas uint, gasPrice uint, value, nonce uint, data string)Tx{
	return Tx{from, to, gas, gasPrice, value, nonce, data, uint64(time.Now().Unix())}
}

func NewBaseTx(from, to common.Address, value, nonce uint, data string) Tx{
	return NewTx(from, to, TxGas, TxGasPriceDefault, value, nonce, data)
}

func NewSignedTx(tx Tx, sig []byte) SignedTx{
	return SignedTx{tx, sig} 
}

func (tx Tx) IsReward() bool{
	return tx.Data == "reward"
}

func (tx Tx) Cost() uint{
	return tx.Value + tx.GasCost()
}

func (tx Tx) GasCost() uint{
	return tx.Gas + tx.GasPrice
}

func (tx Tx) Hash() (Hash, error){
	txJson, err := tx.Encode() 
	if err != nil{
		return Hash{}, err
	}
	return sha256.Sum256(txJson), nil 
}

func (tx Tx) Encode() ([]byte, error){
	return json.Marshal(tx)
}

// MarshalJson is the source of truth when it comes to 
// encoding a transactionfor hash calculations
func (t Tx) MarshalJSON() ([]byte, error){
	if t.Gas == 0{
		type LegacyTx struct{
			From common.Address `json:"from"`
			To common.Address `json:"to"`
			Value uint `json:"value"`
			Nonce uint `json:"nonce"`
			Data string `json:"data"`
			Time uint64 `json:"time"`
		}
		return json.Marshal(LegacyTx{
			From: t.From, 
			To: t.To, 
			Value: t.Value, 
			Nonce: t.Nonce, 
			Data: t.Data, 
			Time: t.Time,
		})
	}
	type NemosTx struct{
		From common.Address `json:"address"`
		To common.Address `json:"to"`
		Gas uint `json:"gas"`
		GasPrice uint `json:"gas_price"`
		Value uint `json:"value"`
		Nonce uint `json:"nonce"`
		Data string `json:"data"`
		Time uint64 `json:"time"` 
		Sig []byte `json:"signature"`
	}

	return json.Marshal(NemosTx{
		From: t.From, 
		To: t.To, 
		Gas: t.Gas, 
		GasPrice: t.GasPrice, 
		Value: t.Value, 
		Nonce: t.Nonce, 
		Data: t.Data, 
		Time: t.Time,

	})
}

func (t SignedTx) MarshalJSON() ([]byte, error){
	if t.Gas == 0{
		type LegacyTx struct{
			From common.Address `json:"from"`
			To common.Address `json:"to"`
			Value uint `json:"value"`
			Nonce uint `json:"nonce"`
			Data string `json:"data"`
			Time uint64 `json:"time"`
			Sig []byte `json:"signature"`
		}

		return json.Marshal(LegacyTx{
			From: t.From, 
			To: t.To, 
			Value: t.Value, 
			Nonce: t.Nonce, 
			Data: t.Data, 
			Time: t.Time, 
			Sig: t.Sig, 
		})
	}

	type NemosTx struct{
		From common.Address `json:"from"`
		To common.Address `json:"to"`
		Gas uint `json:"gas"`
		GasPrice uint `json:"gas_price"`
		Value uint `json:"value"`
		Nonce uint `json:"nonce"`
		Data string `json:"data"`
		Time uint64 `json:"time"`
		Sig []byte `json:"signature"`
	}

	return json.Marshal(NemosTx{
		From: t.From, 
		To: t.To, 
		Gas: t.Gas, 
		GasPrice: t.GasPrice, 
		Value: t.Value, 
		Nonce: t.Nonce, 
		Data: t.Data, 
		Time: t.Time, 
		Sig: t.Sig, 
	})
}

func (t SignedTx) Hash() (Hash, error){
	txJson, err := t.Encode() 
	if err != nil{
		return Hash{}, err
	}

	return sha256.Sum256(txJson), nil 
}

func (t SignedTx) IsAuthentic() (bool, error){
	txHash, err := t.Tx.Hash() 
	if err != nil{
		return false, err
	}

	recoveredPubKey, err := crypto.SigToPub(txHash[:], t.Sig) 
	if err != nil{
		return false, err 
	}

	recoveredPubKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPubKey.X, recoveredPubKey.Y)
	recoveredPubKeyHash := crypto.Keccak256(recoveredPubKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPubKeyHash[12:])

	return recoveredAccount.Hex() == t.From.Hex(), nil 
}