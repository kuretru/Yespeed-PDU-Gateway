package entity

type MQTTCollectorConfig struct {
	URL       string `yaml:"url"`
	Keepalive uint16 `yaml:"keepalive"`
	Topic     string `yaml:"topic"`
	ClientID  string `yaml:"client_id"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

type CollectorConfig struct {
	Type string               `yaml:"type"`
	MQTT *MQTTCollectorConfig `yaml:"mqtt"`
}

type PublisherConfig struct {
	Type string `yaml:"type"`
}

type DeviceType string

var (
	DeviceTypePDU DeviceType = "pdu"
)

// PDUDevice PDU设备
type PDUDevice struct {
	NodeID        string  `json:"node_id"`
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Voltage       float32 `json:"voltage"`        // 电压
	Current       float32 `json:"current"`        // 电流
	Power         float32 `json:"power"`          // 有功功率
	ApparentPower float32 `json:"apparent_power"` // 视在功率
	Factor        float32 `json:"factor"`         // 功率因数
	Frequency     float32 `json:"frequency"`      // 电网频率
}
