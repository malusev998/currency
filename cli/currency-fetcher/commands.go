package main

import (
	"log"
	"time"

	currency_fetcher "github.com/BrosSquad/currency-fetcher"
	"github.com/spf13/cobra"
)

var (
	rootCmd *cobra.Command
)

type commandConfig struct {
	CurrenciesToFetch []string
	CurrencyService   currency_fetcher.Service
	fetchCommand      struct {
		Standalone bool
		After      time.Duration
	}
	debug bool
}

func fetchCommand(config *commandConfig) *cobra.Command {

	handle := func() error {
		currenciesMap, err := config.CurrencyService.Save(config.CurrenciesToFetch)

		if err != nil {
			return err
		}

		if !config.debug {
			return nil
		}

		for storage, currencies := range currenciesMap {
			for i, currency := range currencies {
				log.Printf("%d\tCurrency %s_%s saved to %s: Rate: %f\n", i, currency.From, currency.To, storage, currency.Rate)
			}
		}

		return nil
	}

	fetchCmd := &cobra.Command{
		Use: "fetch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.fetchCommand.Standalone {
				for {
					select {
					case <-time.After(config.fetchCommand.After):
						if err := handle(); err != nil {
							return err
						}
					}
				}
			}

			return handle()
		},
	}

	config.fetchCommand.Standalone = *fetchCmd.Flags().BoolP("standalone", "s", false, "Start up a long running fetching service")
	config.fetchCommand.After = *fetchCmd.Flags().DurationP("after", "a", time.Duration(1)*time.Hour, "Fetching for standalone process")

	return fetchCmd
}

func execute(config *commandConfig) error {
	rootCmd = &cobra.Command{
		Use:     "currency-fetcher",
		Short:   "ISO Currency rate fetcher",
		Version: "v1.0.0",
	}

	config.debug = *rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug flag")
	rootCmd.AddCommand(fetchCommand(config))

	return rootCmd.Execute()
}
