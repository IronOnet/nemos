package main

import (
	"fmt"
	"os"

	"github.com/irononet/nemos/core"
	"github.com/irononet/nemos/node"
	"github.com/spf13/cobra"
)

func balancesCmd() *cobra.Command{
	var balancesCmd = &cobra.Command{
		Use: "balances", 
		Short: "Interacts with balances (list...).", 
		PreRunE: func(cmd *cobra.Command, args []string) error{
			return incorrectUsageErr() 
		}, 
		Run: func(cmd *cobra.Command, args []string){

		},
	}

	balancesCmd.AddCommand(balancesListCmd())

	return balancesCmd
}

func balancesListCmd() *cobra.Command{
	var balancesListCmd = &cobra.Command{
		Use: "list", 
		Short: "List all balances.", 
		Run: func(cmd *cobra.Command, args []string){
			state, err := core.NewStateFromDisk(getDataDirFromCmd(cmd), node.DefaultMiningDifficulty)
			if err != nil{
				fmt.Fprintln(os.Stderr, err) 
				os.Exit(1) 
			}
			defer state.Close()

			fmt.Println("Accounts balances at '%x:\n", state.LatestBlockHash()) 
			fmt.Println("________________________") 
			fmt.Println("") 
			for account, balance := range state.Balances{
				fmt.Println(fmt.Sprintf("%s:%d", account.String(), balance)) 
			}

			fmt.Println("") 
			fmt.Println("Acount nonces:") 
			fmt.Println("") 
			fmt.Println("________________________") 
			fmt.Println("") 
			for account, nonce := range state.AccountToNonce{
				fmt.Println(fmt.Sprintf("%s:%d", account.String(), nonce)) 
			}
		},
	}

	addDefaultRequiredFlags(balancesListCmd) 

	return balancesListCmd
}