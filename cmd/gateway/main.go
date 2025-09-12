package main

import (
	"flag"
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
)

func main() {
	config := loadConfig()
	fmt.Printf("%+v", config)
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
