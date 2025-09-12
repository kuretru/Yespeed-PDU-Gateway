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
