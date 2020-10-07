package main

import (
	"time"

	"github.com/spf13/cobra"
)

var (
	rootCmd *cobra.Command
)

func fetchCommand() {
	
}


func Execute() error {
	rootCmd = &cobra.Command{
		Use:   "currency-fetcher",
		Short: "ISO Currency rate fetcher",
	}

	fetchCmd := &cobra.Command{
		Use: "fetch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	fetchCmd.Flags().Bool("standalone", false, "Start up a long running fetching service")
	fetchCmd.Flags().Duration("after", time.Duration(1)*time.Hour, "Fetching for standalone process")

	rootCmd.AddCommand(fetchCmd)

	return rootCmd.Execute()
}
