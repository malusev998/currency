package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/spf13/viper"

	"github.com/malusev998/currency/cli/cmd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	configPath, err := cmd.Init("./config.yml")

	if err != nil {
		log.Fatalf("Error while initializing ")
	}

	viper.SetConfigFile(configPath)
	viper.SetEnvPrefix("CURRENCY_FETCHER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error while reading in the config file: %v\n", err)
	}

	defer cancel()

	config, err := getConfig(ctx)

	if err != nil {
		log.Fatal(err)
	}

	storages, err := createStorages(config)

	if err != nil {
		log.Fatalf("Error while creating storages: %v\n", err)
	}

	fetchServices, err := createCurrencyService(config, storages)

	if err != nil {
		log.Fatalf("Error while creating fetchers services: %v\n", err)
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	go func(signalChannel <-chan os.Signal, cancel context.CancelFunc) {
		<-signalChannel
		cancel()
	}(signalChannel, cancel)

	err = cmd.Execute(&cmd.Config{
		Ctx:               ctx,
		CurrenciesToFetch: config.CurrenciesToFetch,
		CurrencyService:   fetchServices,
	})

	if err != nil {
		log.Fatalf("Error while executing command: %v", err)
	}

	for _, st := range storages {
		if err := st.Close(); err != nil {
			log.Fatalf("Error while closing the storage %s: %v\n", st.GetStorageProviderName(), err)
		}
	}
}
