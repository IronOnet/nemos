package main 

import (
	"fmt" 
	"github.com/spf13/cobra"
)

const MAJOR = "1" 
const MINOR = "0" 
const FIX = "0" 
const VERBAL = "TX Gas" 

// Confirgured via -ldflags during build 
var GitCommit string 

var versionCmd = &cobra.Command{
	Use: "version", 
	Short: "Describes version.", 
	Run: func(cmd *cobra.Command, args []string){
		fmt.Println(fmt.Sprintf("version: %s.%s.%s-alpha %s %s", MAJOR, MINOR, FIX, shortGitCommit(GitCommit), VERBAL))
	},
}

func shortGitCommit(fullGitCommit string) string{
	shortCommit := "" 
	if len(fullGitCommit) >= 6{
		shortCommit = fullGitCommit[0:6] 
	}
	return shortCommit
}