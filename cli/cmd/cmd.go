package cmd

import (
	"context"
	currency_fetcher "github.com/malusev998/currency-fetcher"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:     "fetchers-fetcher",
		Short:   "ISO Currency rate fetcher",
		Version: "v1.1.0",
	}
	debug      bool
	configFile string
)

type (
	Config struct {
		Ctx               context.Context
		CurrenciesToFetch []string
		CurrencyService   []currency_fetcher.Service
		debug             *bool
	}
)

func Execute(config *Config) error {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Debug flag")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "./config.yml", "Path to config file")
	cobra.OnInitialize()

	absolutePath, _ := filepath.Abs(configFile)

	viper.SetConfigFile(absolutePath)
	viper.SetEnvPrefix("CURRENCY_FETCHER")
	viper.AutomaticEnv()

	config.debug = &debug

	rootCmd.AddCommand(fetch(config))

	return rootCmd.Execute()
}
