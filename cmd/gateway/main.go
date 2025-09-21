package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/goccy/go-yaml"
	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/collector"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/database"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/publisher"
)

type Config struct {
	Collectors []*entity.CollectorConfig `yaml:"collectors"`
	Publishers []*entity.PublisherConfig `yaml:"publishers"`
}

func main() {
	config := loadConfig()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database.Init(ctx)
	if err := collector.Init(ctx, config.Collectors); err != nil {
	}
	if err := publisher.Init(ctx, config.Publishers); err != nil {
		log.Fatal(err.Error())
	}

	<-ctx.Done()
	log.Printf("Received shutdown signal, exiting gracefully...")

	stopCtx := context.Background()
	collector.Stop(stopCtx)
	publisher.Stop(stopCtx)
}

func loadConfig() *Config {
	configFilePath := flag.String("config", "./configs/gateway.yaml", "Config file path")
	flag.Parse()
	if configFilePath == nil || *configFilePath == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Config file not provide")
		os.Exit(2)
	}
	if _, err := os.Stat(*configFilePath); err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintf(os.Stderr, "Config file not exist")
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "Stat config file failed, %v", err)
		}
		os.Exit(3)
	}

	configBytes, err := os.ReadFile(*configFilePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Read config file failed, %v", err)
		os.Exit(3)
	}
	var config Config
	if err = yaml.Unmarshal(configBytes, &config); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unmarshal config file failed, %v", err)
		os.Exit(3)
	}
	return &config
}
