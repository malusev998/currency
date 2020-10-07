package cmd

import (
	"context"
	"github.com/BrosSquad/currency-fetcher"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "currency-fetcher",
		Short:   "ISO Currency rate fetcher",
		Version: "v1.0.0",
	}
	debug bool
)

type (
	Config struct {
		Ctx               context.Context
		CurrenciesToFetch []string
		CurrencyService   currency_fetcher.Service
		debug             *bool
	}
)

func Execute(config *Config) error {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Debug flag")
	config.debug = &debug
	rootCmd.AddCommand(fetch(config))
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()
}
