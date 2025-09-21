package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
	"github.com/kuretru/Yespeed-PDU-Gateway/entity/hass"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/collector"
	"github.com/kuretru/Yespeed-PDU-Gateway/internal/database"
)

const (
	devicePrefix = "yespeed_pdu_"
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

	router := paho.NewStandardRouter()
	router.DefaultHandler(func(publish *paho.Publish) {
		slog.Info("Publisher.HASS_MQTT: message received without hit any route", "topic", publish.Topic)
	})
	router.RegisterHandler("homeassistant/device/+/set", setDeviceStateHandler)

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
			slog.Info("Publisher.HASS_MQTT: connected to server")
			if _, err = connectionManager.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: config.MQTT.Topic, QoS: 1},
				},
			}); err != nil {
				slog.Error("Publisher.HASS_MQTT: subscribe failed", "err", err)
				return
			}
			slog.Info("Publisher.HASS_MQTT: subscribed to", "topic", config.MQTT.Topic)
		},
		OnConnectError: func(err error) {
			slog.Error("Publisher.HASS_MQTT: connect failed", "err", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: config.MQTT.ClientID,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(publishReceived paho.PublishReceived) (bool, error) {
					router.Route(publishReceived.Packet.Packet())
					return true, nil
				}},
			OnClientError: func(err error) {
				slog.Info("Publisher.HASS_MQTT: client error", "err", err)
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil && d.Properties.ReasonString != "" {
					slog.Error("Publisher.HASS_MQTT: server requested disconnect", "reason", d.Properties.ReasonString)
				} else {
					slog.Error("Publisher.HASS_MQTT: server requested disconnect", "reasonCode", d.ReasonCode)
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
	slog.Info("Publisher.HASS_MQTT: initialized", "server", config.MQTT.URL)

	// Slow start, waiting point value is ready
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(20 * time.Second):
		go publisher.runConfigTopic(ctx)
		go publisher.runStateTopic(ctx)
	}

	return nil
}

func (publisher *HomeAssistantMQTTPublisher) Stop(ctx context.Context) {
	if publisher.connectionManager != nil {
		publisher.connectionManager.Done()
	}
	slog.Info("Publisher.HASS_MQTT: stopped")
}

func (publisher *HomeAssistantMQTTPublisher) runConfigTopic(ctx context.Context) {
	publisher.publishConfigTopic(context.Background())

	configTopicTicker := time.NewTicker(5 * time.Minute)
	defer configTopicTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-configTopicTicker.C:
			publisher.publishConfigTopic(context.Background())
		}
	}
}

func (publisher *HomeAssistantMQTTPublisher) runStateTopic(ctx context.Context) {
	stateTopicTicker := time.NewTicker(15 * time.Second)
	defer stateTopicTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stateTopicTicker.C:
			publisher.publishStateTopic(context.Background())
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
			Components:   make(map[string]hass.Component),
			CommandTopic: fmt.Sprintf("homeassistant/device/%v%v/set", devicePrefix, nodeId),
			StateTopic:   fmt.Sprintf("homeassistant/device/%v%v/state", devicePrefix, nodeId),
			QOS:          0,
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
			Topic:   fmt.Sprintf("homeassistant/device/%v%v/config", devicePrefix, nodeId),
			Payload: payloadBytes,
		})
		slog.Info("Publisher.HASS_MQTT: published config topic")
	}
}

// buildConfigPayload mode=normal->正常情况 mode=delete->删除
func buildConfigPayload(device *entity.PDUDevice, mode string) []hass.Component {
	result := make([]hass.Component, 0)

	_switch := hass.Component{
		Platform: "switch",
		Key:      fmt.Sprintf("switch_%v_switch", device.ID),
	}
	if mode != "delete" {
		_switch.DeviceClass = "outlet"
		_switch.Name = fmt.Sprintf("%v 开关", device.Name)
		_switch.ObjectID = fmt.Sprintf("%v%v_%v", devicePrefix, device.NodeID, _switch.Key)
		_switch.UniqueID = _switch.Key
		_switch.ValueTemplate = fmt.Sprintf("{{ value_json.switch_%v.switch }}", device.ID)
		_switch.PayloadOn = fmt.Sprintf(`{"switch_%v_switch":"ON"}`, device.ID)
		_switch.PayloadOff = fmt.Sprintf(`{"switch_%v_switch":"OFF"}`, device.ID)
		_switch.StateOn = "ON"
		_switch.StateOff = "OFF"
	}
	result = append(result, _switch)

	voltage := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_voltage", device.ID),
	}
	if mode != "delete" {
		voltage.DeviceClass = "voltage"
		voltage.Name = fmt.Sprintf("%v 电压", device.Name)
		voltage.ObjectID = fmt.Sprintf("%v%v_%v", devicePrefix, device.NodeID, voltage.Key)
		voltage.UniqueID = voltage.Key
		voltage.UnitOfMeasurement = "V"
		voltage.ValueTemplate = fmt.Sprintf("{{ value_json.switch_%v.voltage }}", device.ID)
	}
	result = append(result, voltage)

	current := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_current", device.ID),
	}
	if mode != "delete" {
		current.DeviceClass = "current"
		current.Name = fmt.Sprintf("%v 电流", device.Name)
		current.ObjectID = fmt.Sprintf("%v%v_%v", devicePrefix, device.NodeID, current.Key)
		current.UniqueID = current.Key
		current.UnitOfMeasurement = "A"
		current.ValueTemplate = fmt.Sprintf("{{ value_json.switch_%v.current }}", device.ID)
	}
	result = append(result, current)

	power := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_power", device.ID),
	}
	if mode != "delete" {
		power.DeviceClass = "power"
		power.Name = fmt.Sprintf("%v 有功功率", device.Name)
		power.ObjectID = fmt.Sprintf("%v%v_%v", devicePrefix, device.NodeID, power.Key)
		power.UniqueID = power.Key
		power.UnitOfMeasurement = "W"
		power.ValueTemplate = fmt.Sprintf("{{ value_json.switch_%v.power }}", device.ID)
	}
	result = append(result, power)

	energy := hass.Component{
		Platform: "sensor",
		Key:      fmt.Sprintf("switch_%v_energy", device.ID),
	}
	if mode != "delete" {
		energy.DeviceClass = "energy"
		energy.Name = fmt.Sprintf("%v 有功总电能", device.Name)
		energy.ObjectID = fmt.Sprintf("%v%v_%v", devicePrefix, device.NodeID, energy.Key)
		energy.StateClass = "total_increasing"
		energy.UniqueID = energy.Key
		energy.UnitOfMeasurement = "kWh"
		energy.ValueTemplate = fmt.Sprintf("{{ value_json.switch_%v.energy }}", device.ID)
	}
	result = append(result, energy)

	return result
}

func (publisher *HomeAssistantMQTTPublisher) publishStateTopic(ctx context.Context) {
	for _, nodeId := range database.GetAllPDUNodes(ctx) {
		payload := make(map[string]any)
		for _, device := range database.GetPDUNodeDevices(ctx, nodeId) {
			switchState := "OFF"
			if device.PduDevice.On {
				switchState = "ON"
			}
			payload[fmt.Sprintf("switch_%v", device.PduDevice.ID)] = map[string]any{
				"switch":  switchState,
				"voltage": device.PduDevice.Voltage,
				"current": device.PduDevice.Current,
				"power":   device.PduDevice.Power,
				"energy":  device.PduDevice.Energy,
			}
		}

		payloadBytes, _ := json.Marshal(payload)
		_, _ = publisher.connectionManager.Publish(ctx, &paho.Publish{
			QoS:     0,
			Retain:  true,
			Topic:   fmt.Sprintf("homeassistant/device/%v%v/state", devicePrefix, nodeId),
			Payload: payloadBytes,
		})
	}
	slog.Info("Publisher.HASS_MQTT: published state topic")
}

func setDeviceStateHandler(publish *paho.Publish) {
	ctx := context.Background()

	command := entity.Command{
		NodeID:   "unknown",
		DeviceID: "",
		Type:     "",
		Command:  "",
	}
	topicSeg := strings.Split(publish.Topic, "/")
	if !strings.HasPrefix(topicSeg[2], devicePrefix) {
		slog.Info("Publisher.HASS_MQTT: received not my topic", "topic", publish.Topic)
		return
	}

	command.NodeID = strings.ReplaceAll(topicSeg[2], devicePrefix, "")
	var payload map[string]string
	if err := json.Unmarshal(publish.Payload, &payload); err != nil {
		slog.Info("Publisher.HASS_MQTT: unmarshal payload failed", "err", err)
		return
	}

	for key, state := range payload {
		keySeg := strings.Split(key, "_")
		if len(keySeg) != 3 {
			continue
		}
		command.DeviceID = keySeg[1]
		command.Type = keySeg[2]
		command.Command = state
		collector.SendCommand(ctx, &command)
	}
}
