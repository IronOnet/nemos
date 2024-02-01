package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/irononet/nemos/core"
	"github.com/irononet/nemos/fs"
)

const testKeystoreAccountPwd = "security123" 

func TestSignCryptoParams(t *testing.T){
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader) 
	if err != nil{
		t.Fatal(err) 
	}
	spew.Dump(privKey) 

	msg := []byte("nemos is a revolutionary blockchain") 

	sig, err := Sign(msg, privKey) 
	if err != nil{
		t.Fatal(err) 
	}

	if len(sig) != crypto.SignatureLength{
		t.Errorf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength)
	}

	r := new(big.Int).SetBytes(sig[:32]) 
	s := new(big.Int).SetBytes(sig[32:64]) 
	v := new(big.Int).SetBytes([]byte{sig[64]}) 

	spew.Dump(r, s, v) 
}

func TestSign(t *testing.T){
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader) 
	if err != nil{
		t.Fatal(err) 
	}

	pubKey := privKey.PublicKey 
	pubKeyBytes := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y) 
	pubKeyBytesHash := crypto.Keccak256(pubKeyBytes[1:]) 

	account := common.BytesToAddress(pubKeyBytesHash[12:]) 
	msg := []byte("nemos is a revolutionary blockchain")

	sig, err := Sign(msg, privKey) 
	if err != nil{
		t.Fatal(err) 
	}

	recoveredPubKey, err := Verify(msg, sig) 
	if err != nil{
		t.Fatal(err) 
	}

	recoveredPubKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPubKey.X, recoveredPubKey.Y) 
	recoveredPubKeyBytesHash := crypto.Keccak256(recoveredPubKeyBytes[1:]) 
	recoveredAccount := common.BytesToAddress(recoveredPubKeyBytesHash[12:]) 

	if account.Hex() != recoveredAccount.Hex(){
		t.Errorf("msg was signed by account %s but signature recovery produced an account %s", account.Hex(), recoveredAccount.Hex()) 
	}
}


func TestSignTxWithKeyStoreAccount(t *testing.T){
	tempDir, err := ioutil.TempDir("", "wallet_test") 
	if err != nil{
		t.Fatal(err) 
	}

	defer fs.RemoveDir(tempDir) 

	nemosRoot, err := NewKeystoreAccount(tempDir, testKeystoreAccountPwd) 
	if err != nil{
		t.Error(err) 
		return 
	}

	optimus, err := NewKeystoreAccount(tempDir, testKeystoreAccountPwd) 
	if err != nil{
		t.Error(err) 
		return 
	}

	tx := core.NewBaseTx(nemosRoot, optimus, 100, 1, "")  

	signedTx, err := SignWithKeystoreAccount(tx, nemosRoot, testKeystoreAccountPwd, GetKeystoreDirPath(tempDir))
	if err != nil{
		t.Error(err) 
		return 
	}

	ok, err := signedTx.IsAuthentic() 
	if err != nil{
		t.Error(err) 
		return 
	}
	
	if !ok{
		t.Error("the tx was signed by from account and should have been authentic") 
		return 
	}

	signedTxJson, err := json.Marshal(signedTx) 
	if err != nil{
		t.Error(err) 
		return 
	}

	var signedTxUnmarshalled core.SignedTx 
	err = json.Unmarshal(signedTxJson, &signedTxUnmarshalled) 
	if err != nil{
		t.Error(err) 
		return 
	}

	assert.Equal(t, signedTx, signedTxUnmarshalled) 
}

func TestSignForgedTxWithKeystoreAccount(t *testing.T){
	tempDir, err := ioutil.TempDir("", "wallet_test") 
	if err != nil{
		t.Fatal(err) 
	}
	defer fs.RemoveDir(tempDir) 

	attacker, err := NewKeystoreAccount(tempDir, testKeystoreAccountPwd) 
	if err != nil{
		t.Error(err) 
		return 
	}

	optimusAccount, err := NewKeystoreAccount(tempDir, testKeystoreAccountPwd) 
	if err != nil{
		t.Error(err) 
		return 
	}

	forgedTx := core.NewBaseTx(optimusAccount, attacker, 100, 1, "") 

	signedTx, err := SignWithKeystoreAccount(forgedTx, attacker, testKeystoreAccountPwd, GetKeystoreDirPath(tempDir))
	if err != nil{
		t.Error(err) 
		return 
	}

	ok, err := signedTx.IsAuthentic() 
	if err != nil{
		t.Error(err) 
		return 
	}

	if ok{
		t.Fatal("the tx from attribute was forged and should have not been authentic") 
	}
}

