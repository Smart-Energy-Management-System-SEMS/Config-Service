package services

import (
	"errors"
	"sort"
	"strings"
	"time"

	"config-service/config-service/domain/model"
	"config-service/config-service/infrastructure/configuration"
	"config-service/shared"
)

var groupedTopics = []string{
	"iam.events",
	"device.events",
	"energy.events",
	"analytics.events",
	"alerts.events",
	"payments.events",
	"subscriptions.events",
	"billing.events",
}

type serviceDefinition struct {
	Key          string
	Name         string
	DisplayName  string
	LocalURL     string
	LocalPort    string
	Prefix       string
	Health       string
	LocalEnvKeys []string
	AzureEnvKeys []string
	Aliases      []string
	Publish      []string
	Consume      []string
}

var serviceCatalog = []serviceDefinition{
	{
		Key:          "iam-service",
		Name:         "iam-service",
		DisplayName:  "IAM Service",
		LocalURL:     "http://localhost:8082",
		LocalPort:    "8082",
		Prefix:       "/api/v1",
		Health:       "/health",
		LocalEnvKeys: []string{"IAM_SERVICE_URL"},
		AzureEnvKeys: []string{"IAM_SERVICE_URL_AZURE", "IAM_SERVICE_URL"},
		Aliases:      []string{"iam", "iam-service"},
		Publish:      []string{"iam.events"},
	},
	{
		Key:          "device-management-service",
		Name:         "device-management-service",
		DisplayName:  "Device Management Service",
		LocalURL:     "http://localhost:8083",
		LocalPort:    "8083",
		Prefix:       "/api/v1/device-management",
		Health:       "/api/v1/device-management/health",
		LocalEnvKeys: []string{"DEVICE_SERVICE_URL", "DEVICE_MANAGEMENT_SERVICE_URL"},
		AzureEnvKeys: []string{"DEVICE_SERVICE_URL_AZURE", "DEVICE_MANAGEMENT_SERVICE_URL_AZURE", "DEVICE_SERVICE_URL", "DEVICE_MANAGEMENT_SERVICE_URL"},
		Aliases:      []string{"device", "device-service", "device-management", "device-management-service"},
		Publish:      []string{"device.events"},
	},
	{
		Key:          "energy-monitoring-service",
		Name:         "energy-monitoring-service",
		DisplayName:  "Energy Monitoring Service",
		LocalURL:     "http://localhost:8001",
		LocalPort:    "8001",
		Prefix:       "/api/v1",
		Health:       "/api/v1/health",
		LocalEnvKeys: []string{"ENERGY_SERVICE_URL", "ENERGY_MONITORING_SERVICE_URL"},
		AzureEnvKeys: []string{"ENERGY_SERVICE_URL_AZURE", "ENERGY_MONITORING_SERVICE_URL_AZURE", "ENERGY_SERVICE_URL", "ENERGY_MONITORING_SERVICE_URL"},
		Aliases:      []string{"energy", "energy-service", "energy-monitoring", "energy-monitoring-service"},
		Publish:      []string{"energy.events"},
		Consume:      []string{"device.events"},
	},
	{
		Key:          "analytics-service",
		Name:         "analytics-service",
		DisplayName:  "Analytics Service",
		LocalURL:     "http://localhost:8004",
		LocalPort:    "8004",
		Prefix:       "/api/v1/analytics",
		Health:       "/api/v1/analytics/health",
		LocalEnvKeys: []string{"ANALYTICS_SERVICE_URL"},
		AzureEnvKeys: []string{"ANALYTICS_SERVICE_URL_AZURE", "ANALYTICS_SERVICE_URL"},
		Aliases:      []string{"analytics", "analytics-service"},
		Publish:      []string{"analytics.events", "billing.events"},
		Consume:      []string{"energy.events"},
	},
	{
		Key:          "alerts-service",
		Name:         "alerts-service",
		DisplayName:  "Alerts Service",
		LocalURL:     "http://localhost:8085",
		LocalPort:    "8085",
		Prefix:       "/api/v1",
		Health:       "/api/v1/health",
		LocalEnvKeys: []string{"ALERTS_SERVICE_URL", "ALERT_SERVICE_URL"},
		AzureEnvKeys: []string{"ALERTS_SERVICE_URL_AZURE", "ALERT_SERVICE_URL_AZURE", "ALERTS_SERVICE_URL", "ALERT_SERVICE_URL"},
		Aliases:      []string{"alerts", "alerts-service", "alert", "alert-service"},
		Publish:      []string{"alerts.events"},
		Consume:      []string{"energy.events", "analytics.events"},
	},
	{
		Key:          "payments-service",
		Name:         "payments-service",
		DisplayName:  "Payments Service",
		LocalURL:     "http://localhost:8086",
		LocalPort:    "8086",
		Prefix:       "/api/v1",
		Health:       "/api/v1/health",
		LocalEnvKeys: []string{"PAYMENTS_SERVICE_URL"},
		AzureEnvKeys: []string{"PAYMENTS_SERVICE_URL_AZURE", "PAYMENTS_SERVICE_URL"},
		Aliases:      []string{"payments", "payments-service"},
		Publish:      []string{"payments.events"},
		Consume:      []string{"billing.events", "subscriptions.events"},
	},
	{
		Key:          "subscriptions-service",
		Name:         "subscriptions-service",
		DisplayName:  "Subscriptions Service",
		LocalURL:     "http://localhost:18083",
		LocalPort:    "18083",
		Prefix:       "/api/v1",
		Health:       "/health",
		LocalEnvKeys: []string{"SUBSCRIPTIONS_SERVICE_URL"},
		AzureEnvKeys: []string{"SUBSCRIPTIONS_SERVICE_URL_AZURE", "SUBSCRIPTIONS_SERVICE_URL"},
		Aliases:      []string{"subscriptions", "subscription", "subscriptions-service"},
		Publish:      []string{"subscriptions.events"},
	},
}

type ConfigService struct {
	services        []model.ServiceConfig
	servicesByKey   map[string]model.ServiceConfig
	inconsistencies []model.TopicInconsistency
}

func NewConfigService(loader *configuration.Loader) (*ConfigService, error) {
	loadedServices, err := loader.LoadServices()
	if err != nil {
		return nil, err
	}

	normalized, inconsistencies := normalizeServices(loadedServices)
	bySource := map[string]model.ServiceConfig{}
	for _, svc := range normalized {
		key := catalogKeyForInput(svc.Name)
		if key == "" {
			key = normalizeLookup(svc.Name)
		}
		bySource[key] = svc
	}

	services := make([]model.ServiceConfig, 0, len(serviceCatalog))
	servicesByKey := make(map[string]model.ServiceConfig, len(serviceCatalog))
	for _, def := range serviceCatalog {
		svc := buildServiceConfig(def, bySource[def.Key])
		services = append(services, svc)
		servicesByKey[def.Key] = svc
	}

	return &ConfigService{
		services:        services,
		servicesByKey:   servicesByKey,
		inconsistencies: inconsistencies,
	}, nil
}

func (s *ConfigService) Health() model.HealthResponse {
	return model.HealthResponse{
		Status:      "ok",
		Service:     "config-service",
		Environment: shared.EnvOrDefault("ENVIRONMENT", "local"),
	}
}

func (s *ConfigService) GetAllServices() []model.ServiceConfig {
	out := make([]model.ServiceConfig, len(s.services))
	copy(out, s.services)
	return out
}

func (s *ConfigService) GetServiceByName(name string) (model.ServiceConfig, error) {
	key := catalogKeyForInput(name)
	if key == "" {
		key = normalizeLookup(name)
	}

	if svc, ok := s.servicesByKey[key]; ok {
		return svc, nil
	}
	return model.ServiceConfig{}, errors.New("service not found")
}

func (s *ConfigService) GetInconsistencies() []model.TopicInconsistency {
	out := make([]model.TopicInconsistency, len(s.inconsistencies))
	copy(out, s.inconsistencies)
	return out
}

func (s *ConfigService) GetTopics() map[string]any {
	return map[string]any{
		"topics":        append([]string{}, groupedTopics...),
		"publishTopics": s.publishTopicsByService(),
		"consumeTopics": s.consumeTopicsByService(),
		"eventEnvelope": map[string]any{
			"eventType":  "energy.consumption.recorded",
			"eventId":    "uuid",
			"occurredAt": "ISO_DATE",
			"data":       map[string]any{},
		},
		"notes": []string{
			"Los topics agrupados son los topics principales.",
			"Los eventos especificos deben viajar dentro del mensaje usando eventType.",
		},
	}
}

func (s *ConfigService) GatewayConfig() model.GatewayConfig {
	return model.GatewayConfig{
		Port:               osOrEmpty("API_GATEWAY_PORT"),
		BaseURLLocal:       getEnv("API_GATEWAY_URL", "http://localhost:8081"),
		BaseURLDeploy:      firstEnv("API_GATEWAY_URL_AZURE", "API_GATEWAY_BASE_URL_DEPLOY"),
		CORSAllowedOrigins: osOrEmpty("API_GATEWAY_CORS_ALLOWED_ORIGINS"),
		AuthRequired:       osOrEmpty("API_GATEWAY_AUTH_REQUIRED"),
		TimeoutSeconds:     osOrEmpty("API_GATEWAY_TIMEOUT_SECONDS"),
		Services:           s.GetAllServices(),
	}
}

func (s *ConfigService) KafkaConfig(profile string) model.KafkaConfig {
	bootstrapServers := resolveKafkaBootstrap(profile)
	protocol := getEnv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT")
	mechanism := getEnv("KAFKA_SASL_MECHANISM", "")
	username := firstEnv("KAFKA_SASL_USERNAME", "KAFKA_USERNAME")
	password := firstEnv("KAFKA_SASL_PASSWORD", "KAFKA_PASSWORD")
	publish := s.publishTopicsByService()
	consume := s.consumeTopicsByService()

	return model.KafkaConfig{
		Enabled:          bootstrapServers != "",
		BootstrapServers: bootstrapServers,
		Brokers:          firstNonEmpty(firstEnv("KAFKA_BROKERS", "KAFKA_BOOTSTRAP_SERVERS"), bootstrapServers),
		BrokerList:       splitCSV(bootstrapServers),
		SecurityProtocol: protocol,
		SASLMechanism:    mechanism,
		Username:         username,
		Password:         maskSecret(password),
		ClientID:         getEnv("KAFKA_CLIENT_ID", "config-service"),
		ConsumerGroup:    firstEnv("KAFKA_CONSUMER_GROUP", "KAFKA_GROUP_ID"),
		Topics:           append([]string{}, groupedTopics...),
		PublishTopics:    publish,
		ConsumeTopics:    consume,
		OfficialTopics:   append([]string{}, groupedTopics...),
		ProducedTopics:   publish,
		ConsumedTopics:   consume,
		Inconsistencies:  s.GetInconsistencies(),
	}
}

func (s *ConfigService) ServiceEndpoints(profile string) map[string]string {
	profile = normalizeProfile(profile)
	azure := isAzureProfile(profile)

	return map[string]string{
		"apiGatewayUrl":              resolveByProfile(azure, "http://localhost:8081", []string{"API_GATEWAY_URL"}, []string{"API_GATEWAY_URL_AZURE", "API_GATEWAY_URL"}),
		"iamServiceUrl":              resolveServiceCatalogURL("iam-service", azure),
		"deviceManagementServiceUrl": resolveServiceCatalogURL("device-management-service", azure),
		"subscriptionsServiceUrl":    resolveServiceCatalogURL("subscriptions-service", azure),
		"paymentsServiceUrl":         resolveServiceCatalogURL("payments-service", azure),
		"alertServiceUrl":            resolveServiceCatalogURL("alerts-service", azure),
		"alertsServiceUrl":           resolveServiceCatalogURL("alerts-service", azure),
		"analyticsServiceUrl":        resolveServiceCatalogURL("analytics-service", azure),
		"energyMonitoringServiceUrl": resolveServiceCatalogURL("energy-monitoring-service", azure),
		"kafkaBrokers":               resolveKafkaBootstrap(profile),
	}
}

func (s *ConfigService) GetRuntimeConfig(serviceName, profile string) (model.RuntimeConfigResponse, error) {
	svc, err := s.GetServiceByName(serviceName)
	if err != nil {
		return model.RuntimeConfigResponse{}, err
	}
	if strings.TrimSpace(profile) == "" {
		profile = "local"
	}

	bootstrap := resolveKafkaBootstrap(profile)
	common := map[string]string{
		"ENVIRONMENT":             getEnv("ENVIRONMENT", "local"),
		"KAFKA_BOOTSTRAP_SERVERS": bootstrap,
		"KAFKA_BROKERS":           firstNonEmpty(firstEnv("KAFKA_BROKERS", "KAFKA_BOOTSTRAP_SERVERS"), bootstrap),
		"KAFKA_SECURITY_PROTOCOL": getEnv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
		"KAFKA_SASL_MECHANISM":    getEnv("KAFKA_SASL_MECHANISM", ""),
		"KAFKA_SASL_USERNAME":     firstEnv("KAFKA_SASL_USERNAME", "KAFKA_USERNAME"),
		"KAFKA_SASL_PASSWORD":     maskSecret(firstEnv("KAFKA_SASL_PASSWORD", "KAFKA_PASSWORD")),
		"KAFKA_CLIENT_ID":         getEnv("KAFKA_CLIENT_ID", "config-service"),
		"KAFKA_CONSUMER_GROUP":    firstEnv("KAFKA_CONSUMER_GROUP", "KAFKA_GROUP_ID"),
		"CONFIG_SOURCE_PATH":      getEnv("CONFIG_SOURCE_PATH", "Rutas"),
	}

	return model.RuntimeConfigResponse{
		Service:       svc.Name,
		Profile:       profile,
		ConfigVersion: "v3",
		Common:        common,
		ServiceConfig: svc,
		Kafka:         s.KafkaConfig(profile),
		Gateway:       s.GatewayConfig(),
		Metadata: map[string]interface{}{
			"served_at_utc":    time.Now().UTC().Format(time.RFC3339),
			"inconsistencies":  s.GetInconsistencies(),
			"grouped_topics":   append([]string{}, groupedTopics...),
			"source_directory": shared.EnvOrDefault("CONFIG_SOURCE_PATH", "Rutas"),
			"event_envelope": map[string]string{
				"eventType":  "energy.consumption.recorded",
				"eventId":    "uuid",
				"occurredAt": "ISO_DATE",
			},
		},
	}, nil
}

func normalizeServices(services []model.ServiceConfig) ([]model.ServiceConfig, []model.TopicInconsistency) {
	inconsistencies := []model.TopicInconsistency{}
	out := make([]model.ServiceConfig, 0, len(services))

	for _, svc := range services {
		svc.Routes = uniqueStrings(svc.Routes)
		svc.TopicsPublished = uniqueStrings(svc.TopicsPublished)
		svc.TopicsConsumed = uniqueStrings(svc.TopicsConsumed)
		svc.HealthAliases = uniqueStrings(svc.HealthAliases)
		out = append(out, svc)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out, inconsistencies
}

func buildServiceConfig(def serviceDefinition, loaded model.ServiceConfig) model.ServiceConfig {
	localURL := resolveByProfile(false, def.LocalURL, def.LocalEnvKeys, def.AzureEnvKeys)
	deployURL := resolveByProfile(true, def.LocalURL, def.LocalEnvKeys, def.AzureEnvKeys)
	prefix := firstNonEmpty(loaded.Prefix, def.Prefix)
	health := firstNonEmpty(loaded.Health, def.Health)
	routes := loaded.Routes
	if len(routes) == 0 {
		routes = loaded.MainEndpoints
	}
	routes = uniqueStrings(routes)

	svc := loaded
	svc.Name = firstNonEmpty(loaded.Name, def.Name)
	svc.DisplayName = def.DisplayName
	svc.ServiceKey = def.Key
	svc.URL = localURL
	svc.DeployURL = deployURL
	svc.Port = firstNonEmpty(explicitPortFromURL(localURL), loaded.Port, def.LocalPort)
	svc.Prefix = prefix
	svc.Health = health
	svc.LocalPort = svc.Port
	svc.BaseURLLocal = localURL
	svc.BaseURLDeploy = deployURL
	svc.RoutePrefix = prefix
	svc.Routes = routes
	svc.MainEndpoints = append([]string{}, routes...)
	svc.PublishTopics = append([]string{}, def.Publish...)
	svc.ConsumeTopics = append([]string{}, def.Consume...)
	svc.TopicsPublished = append([]string{}, def.Publish...)
	svc.TopicsConsumed = append([]string{}, def.Consume...)
	svc.URLByProfile = map[string]string{
		"local": localURL,
		"azure": deployURL,
	}

	if svc.SourceFile == "" {
		svc.SourceFile = "catalog"
	}

	return svc
}

func resolveServiceCatalogURL(key string, azure bool) string {
	for _, def := range serviceCatalog {
		if def.Key == key {
			return resolveByProfile(azure, def.LocalURL, def.LocalEnvKeys, def.AzureEnvKeys)
		}
	}
	return ""
}

func catalogKeyForInput(value string) string {
	normalized := normalizeLookup(value)
	for _, def := range serviceCatalog {
		if normalized == normalizeLookup(def.Key) || normalized == normalizeLookup(def.Name) {
			return def.Key
		}
		for _, alias := range def.Aliases {
			if normalized == normalizeLookup(alias) {
				return def.Key
			}
		}
	}
	return ""
}

func (s *ConfigService) publishTopicsByService() map[string][]string {
	out := make(map[string][]string, len(s.services))
	for _, svc := range s.services {
		out[svc.ServiceKey] = append([]string{}, svc.PublishTopics...)
	}
	return out
}

func (s *ConfigService) consumeTopicsByService() map[string][]string {
	out := make(map[string][]string, len(s.services))
	for _, svc := range s.services {
		out[svc.ServiceKey] = append([]string{}, svc.ConsumeTopics...)
	}
	return out
}

func osOrEmpty(k string) string { return shared.EnvOrDefault(k, "") }

func getEnv(key, fallback string) string { return shared.EnvOrDefault(key, fallback) }

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(shared.EnvOrDefault(key, "")); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeLookup(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "`", "")
	v = strings.ReplaceAll(v, "_", "-")
	return v
}

func resolveKafkaBootstrap(profile string) string {
	switch normalizeProfile(profile) {
	case "docker", "container", "compose":
		return firstNonEmpty(
			firstEnv("KAFKA_BOOTSTRAP_SERVERS_DOCKER"),
			firstEnv("KAFKA_BOOTSTRAP_SERVERS", "KAFKA_BROKERS"),
			"localhost:9092",
		)
	case "azure", "prod", "production":
		return firstNonEmpty(
			firstEnv("KAFKA_BOOTSTRAP_SERVERS_AZURE", "KAFKA_BROKERS_AZURE"),
			firstEnv("KAFKA_BOOTSTRAP_SERVERS", "KAFKA_BROKERS"),
			"localhost:9092",
		)
	default:
		return firstNonEmpty(
			firstEnv("KAFKA_BOOTSTRAP_SERVERS_LOCAL"),
			firstEnv("KAFKA_BOOTSTRAP_SERVERS", "KAFKA_BROKERS"),
			"localhost:9092",
		)
	}
}

func normalizeProfile(profile string) string {
	p := normalizeLookup(profile)
	if p == "" {
		p = normalizeLookup(shared.EnvOrDefault("ENVIRONMENT", "local"))
	}
	if p == "" {
		p = "local"
	}
	return p
}

func isAzureProfile(profile string) bool {
	switch normalizeProfile(profile) {
	case "azure", "prod", "production":
		return true
	default:
		return false
	}
}

func resolveByProfile(azure bool, fallback string, localKeys, azureKeys []string) string {
	if azure {
		return firstNonEmpty(firstEnv(azureKeys...), firstEnv(localKeys...), fallback)
	}
	return firstNonEmpty(firstEnv(localKeys...), fallback)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func explicitPortFromURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ":")
	if len(parts) < 3 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(parts[len(parts)-1]), "/")
}

func maskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "***"
}
