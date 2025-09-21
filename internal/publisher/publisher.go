package publisher

import (
	"context"
	"fmt"

	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
)

var (
	publishers []YespeedPDUPublisher
)

type YespeedPDUPublisher interface {
	Run(ctx context.Context, config *entity.PublisherConfig) error
	Stop(ctx context.Context)
}

func Init(ctx context.Context, configs []*entity.PublisherConfig) error {
	if len(configs) == 0 {
		return fmt.Errorf("publisher config is empty")
	}

	publishers = make([]YespeedPDUPublisher, 0, len(configs))
	for _, config := range configs {
		var publisher YespeedPDUPublisher
		switch config.Type {
		case "hass_mqtt":
			publisher = &HomeAssistantMQTTPublisher{}
		default:
			return fmt.Errorf("unknown publisher type %v", config.Type)
		}

		if err := publisher.Run(ctx, config); err != nil {
			return fmt.Errorf("publisher: run %v publisher failed, %v", config.Type, err)
		}
		publishers = append(publishers, publisher)
	}
	return nil
}

func Stop(ctx context.Context) {
	for _, publisher := range publishers {
		publisher.Stop(ctx)
	}
}
