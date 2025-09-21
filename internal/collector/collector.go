package collector

import (
	"context"
	"fmt"

	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
)

var (
	collectors []YespeedPDUCollector
)

type YespeedPDUCollector interface {
	Run(ctx context.Context, config *entity.CollectorConfig) error
	Stop(ctx context.Context)
	SendCommand(ctx context.Context, command *entity.Command)
}

func Init(ctx context.Context, configs []*entity.CollectorConfig) error {
	if len(configs) == 0 {
		return fmt.Errorf("collector config is empty")
	}

	collectors = make([]YespeedPDUCollector, 0, len(configs))
	for _, config := range configs {
		var collector YespeedPDUCollector
		switch config.Type {
		case "mqtt":
			collector = &MQTTCollector{}
		default:
			return fmt.Errorf("unknown collector type %v", config.Type)
		}

		if err := collector.Run(ctx, config); err != nil {
			return fmt.Errorf("collector: run %v collector failed, %v", config.Type, err)
		}
		collectors = append(collectors, collector)
	}

	return nil
}

func Stop(ctx context.Context) {
	for _, collector := range collectors {
		collector.Stop(ctx)
	}
}

func SendCommand(ctx context.Context, command *entity.Command) {
	for _, collector := range collectors {
		collector.SendCommand(ctx, command)
	}
}
