package cmd

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	currency_fetcher "github.com/malusev998/currency-fetcher"
)

var (
	rootCmd = &cobra.Command{
		Use:     "currency-fetcher",
		Short:   "ISO Currency rate fetcher",
		Version: "v1.0.0",
	}
	debug      bool
	configFile string
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
	config.debug = &debug
	rootCmd.AddCommand(fetch(config))
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Debug flag")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "./config.yml", "Path to config file")
	cobra.OnInitialize()
	absolutePath, _ := filepath.Abs(configFile)
	viper.SetConfigFile(absolutePath)
}
