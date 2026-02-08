package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "leetcli",
	Short: "Terminal-first LeetCode workflow",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(solveCmd)
	rootCmd.AddCommand(browseCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(timerCmd)
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(statsCmd)
}
