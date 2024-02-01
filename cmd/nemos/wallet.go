package main 

import (
	"fmt" 
	"io/ioutil" 
	"os" 

	"github.com/davecgh/go-spew/spew" 
	"github.com/ethereum/go-ethereum/accounts/keystore" 
	"github.com/ethereum/go-ethereum/cmd/utils" 
	"github.com/spf13/cobra" 
	"github.com/irononet/nemos/wallet"
)

func walletCmd() *cobra.Command{
	var walletCmd = &cobra.Command{
		Use : "wallet", 
		Short: "Managees blockchain accounts and keys.", 
		PreRunE: func(cmd *cobra.Command, args []string) error{
			return incorrectUsageErr() 
		},
		Run: func(cmd *cobra.Command, args []string){

		},
	}

	walletCmd.AddCommand(walletNewAccountCmd()) 
	walletCmd.AddCommand(walletPrintPrivKeyCmd()) 

	return walletCmd
}

func walletNewAccountCmd() *cobra.Command{
	var cmd = &cobra.Command{
		Use: "new-account", 
		Short: "creates a new account with a new set of elliptic-curve private + public keys.", 
		Run: func(cmd *cobra.Command, args []string){
			password := getPassPhrase("please enter a password to encrypto the new wallet:", true) 
			dataDir := getDataDirFromCmd(cmd) 

			acc, err := wallet.NewKeystoreAccount(dataDir, password) 
			if err != nil{
				fmt.Println(err) 
				os.Exit(1) 
			}

			fmt.Printf("new account created: %s\n", acc.Hex()) 
			fmt.Printf("saved in: %s\n", wallet.GetKeystoreDirPath(dataDir)) 
		}, 
	}

	addDefaultRequiredFlags(cmd) 

	return cmd 
}

func walletPrintPrivKeyCmd() *cobra.Command{
	var cmd = &cobra.Command{
		Use: "pk-print", 
		Short: "unlocks keystore file and prints the private + public keys.", 
		Run: func(cmd *cobra.Command, args []string){
			ksFile, _ := cmd.Flags().GetString(flagKeystoreFile) 
			password := getPassPhrase("please enter a password to decrypt the wallet:", false) 

			keyJson, err := ioutil.ReadFile(ksFile) 
			if err != nil{
				fmt.Println(err.Error()) 
				os.Exit(1) 
			}

			key, err := keystore.DecryptKey(keyJson, password) 
			if err != nil{
				fmt.Println(err.Error()) 
				os.Exit(1) 
			}

			spew.Dump(key)
		},
	}

	addKeystoreFlag(cmd) 

	return cmd 
}

func getPassPhrase(prompt string, confirmation bool) string{
	return utils.GetPassPhrase(prompt, confirmation)
}