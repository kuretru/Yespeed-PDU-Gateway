package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/database"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/utils"
)

type MQTTCollector struct {
	connectionManager *autopaho.ConnectionManager
}

func (collector *MQTTCollector) Run(ctx context.Context, config *entity.CollectorConfig) error {
	u, err := url.Parse(config.MQTT.URL)
	if err != nil {
		return fmt.Errorf("Collector.MQTT: parse mqtt url failed: %v, %v", config.MQTT.URL, err)
	}

	router := paho.NewStandardRouter()
	router.DefaultHandler(func(publish *paho.Publish) {
		slog.Warn("Collector.MQTT: message received without hit any route", "topic", publish.Topic)
	})
	router.RegisterHandler("/yespeed/pdu/yespeed/#/out/1000000", queryDeviceGroupHandler)

	clientConfig := autopaho.ClientConfig{
		ServerUrls:      []*url.URL{u},
		KeepAlive:       config.MQTT.Keepalive,
		ConnectUsername: config.MQTT.Username,
		ConnectPassword: []byte(config.MQTT.Password),
		// CleanStartOnInitialConnection defaults to false. Setting this to true will clear the session on the first connection.
		CleanStartOnInitialConnection: false,
		// SessionExpiryInterval - Seconds that a session will survive after disconnection.
		// It is important to set this because otherwise, any queued messages will be lost if the connection drops and
		// the server will not queue messages while it is down. The specific setting will depend upon your needs
		// (60 = 1 minute, 3600 = 1 hour, 86400 = one day, 0xFFFFFFFE = 136 years, 0xFFFFFFFF = don't expire)
		SessionExpiryInterval: 60,
		OnConnectionUp: func(connectionManager *autopaho.ConnectionManager, connAck *paho.Connack) {
			slog.Info("Collector.MQTT: connected to server")
			if _, err = connectionManager.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: config.MQTT.Topic, QoS: 1},
				},
			}); err != nil {
				slog.Error("Collector.MQTT: subscribe failed", "err", err)
				return
			}
			slog.Info("Collector.MQTT: subscribed to", "topic", config.MQTT.Topic)
		},
		OnConnectError: func(err error) {
			slog.Error("Collector.MQTT: connect failed", "err", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: config.MQTT.ClientID,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(publishReceived paho.PublishReceived) (bool, error) {
					router.Route(publishReceived.Packet.Packet())
					return true, nil
				}},
			OnClientError: func(err error) {
				slog.Info("Collector.MQTT: client error", "err", err)
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil && d.Properties.ReasonString != "" {
					slog.Error("Collector.MQTT: server requested disconnect", "reason", d.Properties.ReasonString)
				} else {
					slog.Error("Collector.MQTT: server requested disconnect", "reasonCode", d.ReasonCode)
				}
			},
		},
	}

	collector.connectionManager, err = autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		return fmt.Errorf("Collector.MQTT: NewConnection failed, %v", err)
	}
	if err = collector.connectionManager.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("Collector.MQTT: AwaitConnection failed, %v", err)
	}
	slog.Info("Collector.MQTT: initialized", "server", config.MQTT.URL)

	return nil
}

func (collector *MQTTCollector) Stop(ctx context.Context) {
	if collector.connectionManager != nil {
		collector.connectionManager.Done()
	}
	slog.Info("Collector.MQTT: stopped")
}

func (collector *MQTTCollector) SendCommand(ctx context.Context, command *entity.Command) {

}

type DeviceGroupMessage struct {
	Devices []DeviceGroup `json:"devices"`
}

type DeviceGroup struct {
	ID           int         `json:"id"` // 设备组标识号
	VID          int         `json:"vid"`
	Type         any         `json:"type"` // 设备组类型，1->插座类设备组，2->空调类设备组
	Slave        int         `json:"slave"`
	Name         string      `json:"name"`     // 设备组名称
	Voltage      string      `json:"voltage"`  // 设备组的当前电压
	TotalCurrent string      `json:"tcurrent"` // 设备组的当前总电流
	Power        int         `json:"power"`    // 设备组当前功率
	Freq         string      `json:"freq"`     // 设备组当前频率
	Factor       string      `json:"factor"`
	Energy       string      `json:"energy"` // 设备组当前电量
	Thresmask    int         `json:"thresmask"`
	HW           int         `json:"hw"`
	SubDevices   []SubDevice `json:"subdevs"` // 设备组下的子设备
	DeviceName   string      `json:"deviceName"`
}

type SubDevice struct {
	ID          int    `json:"id"`   // 设备标识
	Type        int    `json:"type"` // 设备类型，1->插座设备
	On          int    `json:"on"`   // 设备的开关状态，1->打开，0->关闭
	Name        string `json:"name"` // 设备名称
	Icon        string `json:"icon"` // 设备图标
	VID         int    `json:"vid"`
	Rintv       int    `json:"rintv"`   // 重启间隔，从关到开的延迟时间
	Dintv       int    `json:"dintv"`   // 延时动作时间
	Who         string `json:"who"`     // 最后一次操作，来源于哪个接口
	Action      string `json:"act"`     // 最后一次操作的动作
	Time        string `json:"tim"`     // 最后一次操作的时间
	Description string `json:"det"`     // 最后一次操作的描述
	Current     string `json:"current"` // 设备当前电流
	Power       string `json:"power"`   // 设备当前功率
	Energy      string `json:"energy"`  // 设备当前电量
}

func queryDeviceGroupHandler(publish *paho.Publish) {
	ctx := context.Background()

	nodeID := "unknown"
	topicSeg := strings.Split(publish.Topic, "/")
	if len(topicSeg) == 7 && topicSeg[4] != "" {
		nodeID = topicSeg[4]
	}

	messageBytes := append([]byte{'{'}, publish.Payload...)
	messageBytes = append(messageBytes, '}')
	var message DeviceGroupMessage
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		slog.Error("Collector.MQTT: DeviceGroupMessage unmarshal failed", "err", err)
	}
	for _, switchGroup := range message.Devices {
		for _, _switch := range switchGroup.SubDevices {
			switchGlobalID := (switchGroup.ID-1)*len(switchGroup.SubDevices) + _switch.ID
			pduDevice := entity.PDUDevice{
				NodeID:    nodeID,
				ID:        fmt.Sprintf("%v", switchGlobalID),
				Name:      _switch.Name,
				Voltage:   utils.ParseFloat32OrZero(switchGroup.Voltage),
				Current:   utils.ParseFloat32OrZero(_switch.Current),
				Power:     utils.ParseFloat32OrZero(_switch.Power),
				Energy:    utils.ParseFloat32OrZero(_switch.Energy),
				Factor:    utils.ParseFloat32OrZero(switchGroup.Factor), // 读上来都是0
				Frequency: utils.ParseFloat32OrZero(switchGroup.Freq),
			}
			if _switch.On == 1 {
				pduDevice.On = true
			}
			database.SetPUDDevice(ctx, pduDevice.NodeID, pduDevice.ID, &pduDevice)
		}
	}
}
