package cmd

import (
	"github.com/spf13/cobra"
	"log"
	"time"
)

func fetch(config *Config) *cobra.Command {
	var standalone bool
	var after time.Duration

	handle := func() error {
		currenciesMap, err := config.CurrencyService.Save(config.CurrenciesToFetch)

		if err != nil {
			return err
		}

		if !*config.debug {
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
		Run: func(cmd *cobra.Command, args []string) {
			if standalone {
				for {
					select {
					case <-time.After(after):
						if err := handle(); err != nil {
							log.Printf("ERROR: %v", err)
						}
						case <-config.Ctx.Done():
							return
					}
				}
			}

			if err := handle(); err != nil {
				log.Printf("ERROR: %v", err)
			}
		},
	}


	fetchCmd.Flags().BoolVar(&standalone, "standalone", false, "Start up a long running fetching service")
	fetchCmd.Flags().DurationVar(&after, "after", time.Duration(1)*time.Hour, "Fetching for standalone process")

	return fetchCmd
}
