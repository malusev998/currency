package cmd

import (
	"log"
	"time"

	"github.com/spf13/cobra"
)

func handleCurrencySave(config *Config, logger *log.Logger) error {
	for _, service := range config.CurrencyService {
		currenciesMap, err := service.Save(config.CurrenciesToFetch)

		if err != nil {
			return err
		}

		if !*config.debug {
			return nil
		}

		for storage, currencies := range currenciesMap {
			for i, currency := range currencies {
				logger.Printf("%d\tCurrency %s_%s saved to %s: Rate: %f\n", i, currency.From, currency.To, storage, currency.Rate)
			}
		}
	}

	return nil
}

func fetchCobraCommand(
	standalone bool,
	after time.Duration,
	config *Config, logger,
	errLogger *log.Logger,
) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := handleCurrencySave(config, logger); err != nil {
			errLogger.Printf("ERROR: %v", err)
		}

		if !standalone {
			return
		}

		for {
			select {
			case <-time.After(after):
				if err := handleCurrencySave(config, logger); err != nil {
					errLogger.Printf("ERROR: %v", err)
				}
			case <-config.Ctx.Done():
				return
			}
		}
	}
}

func fetch(config *Config) *cobra.Command {
	var standalone bool
	var after time.Duration

	fetchCmd := &cobra.Command{
		Use: "fetch",
	}

	logger := log.New(fetchCmd.OutOrStdout(), "fetch ", 0)
	errLogger := log.New(fetchCmd.OutOrStderr(), "fetch-error ", 0)

	fetchCmd.Run = fetchCobraCommand(standalone, after, config, logger, errLogger)
	fetchCmd.Flags().BoolVar(&standalone, "standalone", false, "Start up a long running fetching service")
	fetchCmd.Flags().DurationVar(&after, "after", time.Duration(1)*time.Hour, "Fetching for standalone process")

	return fetchCmd
}
