package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
	"github.com/kuretru/Yespeed-PDU-Gateway/entity/hass"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/database"
)

type HomeAssistantMQTTPublisher struct {
	config            *entity.PublisherConfig
	connectionManager *autopaho.ConnectionManager
}

func (publisher *HomeAssistantMQTTPublisher) Run(ctx context.Context, config *entity.PublisherConfig) error {
	publisher.config = config
	u, err := url.Parse(config.MQTT.URL)
	if err != nil {
		return fmt.Errorf("Publisher.HASS_MQTT: parse mqtt url failed: %v, %v", config.MQTT.URL, err)
	}

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
			log.Printf("Publisher.HASS_MQTT: connected to server")
		},
		OnConnectError: func(err error) {
			log.Fatalf("Publisher.HASS_MQTT: connect failed, %v", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: config.MQTT.ClientID,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(publishReceived paho.PublishReceived) (bool, error) {
					return true, nil
				}},
			OnClientError: func(err error) {
				log.Printf("Publisher.HASS_MQTT: client error, %v", err)
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					log.Fatalf("Publisher.HASS_MQTT: server requested disconnect, %v", d.Properties.ReasonString)
				} else {
					log.Fatalf("Publisher.HASS_MQTT: server requested disconnect, reason code: %v", d.ReasonCode)
				}
			},
		},
	}

	publisher.connectionManager, err = autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		return fmt.Errorf("Publisher.HASS_MQTT: NewConnection failed, %v", err)
	}
	if err = publisher.connectionManager.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("Publisher.HASS_MQTT: AwaitConnection failed, %v", err)
	}
	log.Printf("Publisher.HASS_MQTT: initialized, server=%v", config.MQTT.URL)

	go publisher.runConfigTopic(ctx)

	return nil
}

func (publisher *HomeAssistantMQTTPublisher) Stop(ctx context.Context) {
	if publisher.connectionManager != nil {
		publisher.connectionManager.Done()
	}
	log.Printf("Publisher.HASS_MQTT: stopped")
}

func (publisher *HomeAssistantMQTTPublisher) runConfigTopic(ctx context.Context) {
	// Slow start, waiting point value is ready
	time.Sleep(20 * time.Second)
	publisher.publishConfigTopic(context.Background())

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publisher.publishConfigTopic(context.Background())
		}
	}
}

func (publisher *HomeAssistantMQTTPublisher) publishConfigTopic(ctx context.Context) {
	for _, nodeId := range database.GetAllPDUNodes(ctx) {
		payload := hass.MQTTDiscoveryMessage{
			Device: hass.DeviceInfo{
				ConfigurationUrl: "http://192.168.91.126/pc.html",
				Connections:      nil,
				Identifiers:      nodeId,
				Name:             "PDU",
				Manufacturer:     "Yespeed",
				Model:            "YS-NT6835",
				ModelID:          "",
				HardwareVersion:  "",
				SoftwareVersion:  "OCF 3.0 r29.66",
				SuggestedArea:    "",
				SerialNumber:     "71636183218528358101118199602193",
			},
			Origin: hass.OriginInfo{
				Name:            "mqtt",
				SoftwareVersion: "OCF 3.0 r29.66",
				SupportUrl:      "http://192.168.91.126/pc.html",
			},
			Components: make(map[string]hass.Component),
			StateTopic: fmt.Sprintf("homeassistant/device/yespeed_pdu_%v/state", nodeId),
			QOS:        0,
		}
		for _, device := range database.GetPDUNodeDevices(ctx, nodeId) {
			for _, component := range buildConfigPayload(device.PduDevice, "normal") {
				payload.Components[component.Key] = component
			}
		}

		payloadBytes, _ := json.Marshal(payload)
		_, _ = publisher.connectionManager.Publish(ctx, &paho.Publish{
			QoS:     0,
			Retain:  true,
			Topic:   fmt.Sprintf("homeassistant/device/yespeed_pdu_%v/config", nodeId),
			Payload: payloadBytes,
		})
		log.Printf("Publisher.HASS_MQTT: published config topic")
	}
}

// buildConfigPayload mode=normal->正常情况 mode=delete->删除
func buildConfigPayload(device *entity.PDUDevice, mode string) []hass.Component {
	result := make([]hass.Component, 0)

	voltage := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_voltage", device.ID),
	}
	if mode != "delete" {
		voltage.DeviceClass = "voltage"
		voltage.Name = fmt.Sprintf("%v 电压", device.Name)
		voltage.ObjectID = fmt.Sprintf("yespeed_pdu_%v_%v", device.NodeID, voltage.Key)
		voltage.UniqueID = voltage.Key
		voltage.UnitOfMeasurement = "V"
		voltage.ValueTemplate = "{{ value_json.voltage }}"
	}
	result = append(result, voltage)

	current := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_current", device.ID),
	}
	if mode != "delete" {
		current.DeviceClass = "current"
		current.Name = fmt.Sprintf("%v 电流", device.Name)
		current.ObjectID = fmt.Sprintf("yespeed_pdu_%v_%v", device.NodeID, current.Key)
		current.UniqueID = current.Key
		current.UnitOfMeasurement = "A"
		current.ValueTemplate = "{{ value_json.current }}"
	}
	result = append(result, current)

	power := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_power", device.ID),
	}
	if mode != "delete" {
		power.DeviceClass = "power"
		power.Name = fmt.Sprintf("%v 有功功率", device.Name)
		power.ObjectID = fmt.Sprintf("yespeed_pdu_%v_%v", device.NodeID, power.Key)
		power.UniqueID = power.Key
		power.UnitOfMeasurement = "W"
		power.ValueTemplate = "{{ value_json.power }}"
	}
	result = append(result, power)

	energy := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_energy", device.ID),
	}
	if mode != "delete" {
		energy.DeviceClass = "energy"
		energy.Name = fmt.Sprintf("%v 有功总电能", device.Name)
		energy.ObjectID = fmt.Sprintf("yespeed_pdu_%v_%v", device.NodeID, energy.Key)
		energy.StateClass = "total_increasing"
		energy.UniqueID = energy.Key
		energy.UnitOfMeasurement = "kWh"
		energy.ValueTemplate = "{{ value_json.energy }}"
	}
	result = append(result, energy)

	return result
}
