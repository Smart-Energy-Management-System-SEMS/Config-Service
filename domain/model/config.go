package model

type ServiceConfig struct {
	Name             string   `json:"name"`
	BoundedContext   string   `json:"bounded_context"`
	LocalPort        string   `json:"local_port"`
	BaseURLLocal     string   `json:"base_url_local"`
	BaseURLDeploy    string   `json:"base_url_deploy"`
	RoutePrefix      string   `json:"route_prefix"`
	MainEndpoints    []string `json:"main_endpoints"`
	Dependencies     []string `json:"dependencies"`
	ExternalServices []string `json:"external_services"`
	SourceFile       string   `json:"source_file"`
}

type GatewayConfig struct {
	Port               string `json:"api_gateway_port"`
	BaseURLLocal       string `json:"api_gateway_base_url_local"`
	BaseURLDeploy      string `json:"api_gateway_base_url_deploy"`
	CORSAllowedOrigins string `json:"api_gateway_cors_allowed_origins"`
	AuthRequired       string `json:"api_gateway_auth_required"`
	TimeoutSeconds     string `json:"api_gateway_timeout_seconds"`
}

type KafkaConfig struct {
	BootstrapServers string            `json:"bootstrap_servers"`
	SecurityProtocol string            `json:"security_protocol"`
	SASLMechanism    string            `json:"sasl_mechanism"`
	ProducedTopics   map[string][]string `json:"produced_topics"`
	ConsumedTopics   map[string][]string `json:"consumed_topics"`
	ConsumerGroups   map[string]string `json:"consumer_groups"`
}

type HealthResponse struct {
	Status      string `json:"status"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
}
