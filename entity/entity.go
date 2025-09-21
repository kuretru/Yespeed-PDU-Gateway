package entity

type MQTTConfig struct {
	URL       string `yaml:"url"`
	Keepalive uint16 `yaml:"keepalive"`
	Topic     string `yaml:"topic"`
	ClientID  string `yaml:"client_id"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

type CollectorConfig struct {
	Type string      `yaml:"type"`
	MQTT *MQTTConfig `yaml:"mqtt"`
}

type PublisherConfig struct {
	Type string      `yaml:"type"`
	MQTT *MQTTConfig `yaml:"mqtt"`
}

type DeviceType string

var (
	DeviceTypePDU DeviceType = "pdu"
)

// PDUDevice PDU设备
type PDUDevice struct {
	NodeID    string  `json:"node_id"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Voltage   float32 `json:"voltage"`   // 电压
	Current   float32 `json:"current"`   // 电流
	Power     float32 `json:"power"`     // 有功功率
	Energy    float32 `json:"energy"`    // 总视在功率
	Factor    float32 `json:"factor"`    // 功率因数
	Frequency float32 `json:"frequency"` // 电网频率
}
