package cmd

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/malusev998/currency"
)

var (
	rootCmd = &cobra.Command{
		Use:     "currency-fetcher",
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
		CurrencyService   []currency.Service
		debug             *bool
	}
)

func Init(configPath string) (string, error) {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Debug flag")
	rootCmd.PersistentFlags().
		StringVar(&configFile, "config", configPath, "Path to config file")
	cobra.OnInitialize()

	absolutePath, err := filepath.Abs(configFile)

	if err != nil {
		return "", err
	}

	return absolutePath, nil
}

func Execute(config *Config) error {
	config.debug = &debug

	rootCmd.AddCommand(fetch(config))

	return rootCmd.Execute()
}
