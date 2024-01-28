package core

import (
	"encoding/hex"
	"testing"

	//"github.com/ethereum/go-ethereum/common"
)

func TestHash(t *testing.T){
	hash := Hash{} 

	// Test MarshalText and UnmarshalText 
	text, err := hash.MarshalText()
	if err != nil{
		t.Errorf("Error in MarshalText: %v", err) 
	}

	err = hash.UnmarshalText(text) 
	if err != nil{
		t.Errorf("Error in UnmarshalText: %v", err) 
	}

	// Test Hex 
	hexString := hex.EncodeToString(hash[:]) 
	if hash.Hex() != hexString{
		t.Errorf("incorrect hex value, expecting %s but got %s", hexString, hash.Hex())
	}

	// Test Empty 
	if !hash.IsEmpty(){
		t.Errorf("Expected is empty to be true but got false")
	}

	// Test is empty with non empty hash 
	hash = Hash{1} 
	if hash.IsEmpty(){
		t.Errorf("Expected is empty to be false but got true")
	}
}

func TestBlock(t *testing.T){
	
}