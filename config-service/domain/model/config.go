package model

type ServiceConfig struct {
	Name             string   `json:"name"`
	URL              string   `json:"url"`
	Port             string   `json:"port"`
	Prefix           string   `json:"prefix"`
	Health           string   `json:"health"`
	HealthAliases    []string `json:"healthAliases,omitempty"`
	Routes           []string `json:"routes"`
	TopicsPublished  []string `json:"topicsPublished"`
	TopicsConsumed   []string `json:"topicsConsumed"`
	SourceFile       string   `json:"sourceFile"`
	BoundedContext   string   `json:"boundedContext,omitempty"`
	LocalPort        string   `json:"local_port,omitempty"`
	BaseURLLocal     string   `json:"base_url_local,omitempty"`
	BaseURLDeploy    string   `json:"base_url_deploy,omitempty"`
	RoutePrefix      string   `json:"route_prefix,omitempty"`
	MainEndpoints    []string `json:"main_endpoints,omitempty"`
	Dependencies     []string `json:"dependencies,omitempty"`
	ExternalServices []string `json:"external_services,omitempty"`
}

type GatewayConfig struct {
	Port               string          `json:"port"`
	BaseURLLocal       string          `json:"baseUrlLocal"`
	BaseURLDeploy      string          `json:"baseUrlDeploy"`
	CORSAllowedOrigins string          `json:"corsAllowedOrigins"`
	AuthRequired       string          `json:"authRequired"`
	TimeoutSeconds     string          `json:"timeoutSeconds"`
	Services           []ServiceConfig `json:"services"`
}

type KafkaConfig struct {
	BootstrapServers string               `json:"bootstrapServers"`
	Brokers          []string             `json:"brokers"`
	SecurityProtocol string               `json:"securityProtocol"`
	SASLMechanism    string               `json:"saslMechanism"`
	Username         string               `json:"username"`
	Password         string               `json:"password"`
	ClientID         string               `json:"clientId"`
	ConsumerGroup    string               `json:"consumerGroup"`
	OfficialTopics   []string             `json:"officialTopics"`
	ProducedTopics   map[string][]string  `json:"producedTopics"`
	ConsumedTopics   map[string][]string  `json:"consumedTopics"`
	Inconsistencies  []TopicInconsistency `json:"inconsistencies,omitempty"`
}

type TopicInconsistency struct {
	Service string `json:"service"`
	Topic   string `json:"topic"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
}

type HealthResponse struct {
	Status      string `json:"status"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
}

type RuntimeConfigResponse struct {
	Service       string                 `json:"service"`
	Profile       string                 `json:"profile"`
	ConfigVersion string                 `json:"config_version"`
	Common        map[string]string      `json:"common"`
	ServiceConfig ServiceConfig          `json:"service_config"`
	Kafka         KafkaConfig            `json:"kafka"`
	Gateway       GatewayConfig          `json:"gateway"`
	Metadata      map[string]interface{} `json:"metadata"`
}
