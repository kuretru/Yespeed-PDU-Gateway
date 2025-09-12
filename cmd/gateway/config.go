package main

type Config struct {
	Collector struct {
		Type string `yaml:"type"`
		MQTT struct {
			URL string `yaml:"url"`
		} `yaml:"mqtt"`
	} `yaml:"collector"`
}
