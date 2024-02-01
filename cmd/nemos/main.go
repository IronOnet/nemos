package main

import (
	"fmt"
	"os"

	"github.com/irononet/nemos/fs"
	"github.com/spf13/cobra"
)

const flagKeystoreFile = "keystore"
const flagDataDir = "dataDir"
const flagMiner = "miner" 
const flagSSLEmail = "ssl-email" 
const flagDisableSSL = "disable-ssl" 
const flagIP = "ip" 
const flagPort = "port" 
const flagBootstrapAcc = "bootstrap-account" 
const flagBootstrapIp = "bootstrap-ip" 
const flagBootstrapPort = "bootstrap-port" 

func main(){
	var nemosCmd = &cobra.Command{
		Use: "nemos", 
		Short: "nemos cli tool", 
		Run: func(cmd *cobra.Command, args []string){

		},
	}

	nemosCmd.AddCommand(versionCmd) 
	nemosCmd.AddCommand(balancesCmd()) 
	nemosCmd.AddCommand(walletCmd()) 
	nemosCmd.AddCommand(runCmd()) 

	err := nemosCmd.Execute() 
	if err != nil{
		fmt.Println(os.Stderr, err) 
		os.Exit(1)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command){
	cmd.Flags().String(flagDataDir, "", "absolute path to your node's data dir where the DB will be/is stored") 
	cmd.MarkFlagRequired(flagDataDir) 
}

func addKeystoreFlag(cmd *cobra.Command){
	cmd.Flags().String(flagKeystoreFile, "", "absolute path to the encrypted keystore file") 
	cmd.MarkFlagRequired(flagKeystoreFile) 
}

func getDataDirFromCmd(cmd *cobra.Command) string{
	dataDir, _ := cmd.Flags().GetString(flagDataDir) 
	return fs.ExpandPath(dataDir) 
}

func incorrectUsageErr() error{
	return fmt.Errorf("incorrect usage") 
}