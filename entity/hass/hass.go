package hass

type MQTTDiscoveryMessage struct {
	Device       DeviceInfo           `json:"device"`
	Origin       OriginInfo           `json:"origin"`
	Components   map[string]Component `json:"components"`
	CommandTopic string               `json:"command_topic"`
	StateTopic   string               `json:"state_topic"`
	QOS          int                  `json:"qos"`
}

type DeviceInfo struct {
	ConfigurationUrl string   `json:"configuration_url"`
	Connections      []string `json:"connections"`
	Identifiers      string   `json:"identifiers"`
	Name             string   `json:"name"`
	Manufacturer     string   `json:"manufacturer"`
	Model            string   `json:"model"`
	ModelID          string   `json:"model_id"`
	HardwareVersion  string   `json:"hw_version"`
	SoftwareVersion  string   `json:"sw_version"`
	SuggestedArea    string   `json:"suggested_area"`
	SerialNumber     string   `json:"serial_number"`
}

type OriginInfo struct {
	Name            string `json:"name"`
	SoftwareVersion string `json:"sw_version"`
	SupportUrl      string `json:"support_url"`
}

type Component struct {
	Key               string `json:"-"`
	Platform          string `json:"platform"`
	DeviceClass       string `json:"device_class,omitempty"`
	Name              string `json:"name,omitempty"`
	ObjectID          string `json:"object_id,omitempty"`
	StateClass        string `json:"state_class,omitempty"`
	UniqueID          string `json:"unique_id,omitempty"`
	UnitOfMeasurement string `json:"unit_of_measurement,omitempty"`
	ValueTemplate     string `json:"value_template,omitempty"`

	// switch
	Optimistic *bool  `json:"optimistic,omitempty"`
	PayloadOn  string `json:"payload_on,omitempty"`
	PayloadOff string `json:"payload_off,omitempty"`
	StateOn    string `json:"state_on,omitempty"`
	StateOff   string `json:"state_off,omitempty"`
}
