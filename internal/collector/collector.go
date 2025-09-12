package collector

import (
	"context"
	"fmt"
	"log"

	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
)

type YespeedPDUCollector interface {
	Run(ctx context.Context, config *entity.CollectorConfig) error
}

func Init(ctx context.Context, config *entity.CollectorConfig) error {
	if config == nil {
		return fmt.Errorf("collector config is nil")
	}

	var collector YespeedPDUCollector
	switch config.Type {
	case "mqtt":
		collector = &MQTTCollector{}
	default:
		return fmt.Errorf("unknown collector type %v", config.Type)
	}

	if err := collector.Run(ctx, config); err != nil {
		log.Fatalf("Collector: run %v collector failed, %v", config.Type, err)
		return err
	}
	log.Printf("Collector: %v collector initialized", config.Type)
	return nil
}
