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

type ConfigService struct {
	services []model.ServiceConfig
}

func NewConfigService(loader *configuration.Loader) (*ConfigService, error) {
	services, err := loader.LoadServices()
	if err != nil {
		return nil, err
	}
	return &ConfigService{services: services}, nil
}

func (s *ConfigService) Health() model.HealthResponse {
	return model.HealthResponse{
		Status:      "ok",
		Service:     "config-service",
		Environment: shared.EnvOrDefault("ENVIRONMENT", "local"),
	}
}

func (s *ConfigService) GetAllServices() []model.ServiceConfig {
	return s.services
}

func (s *ConfigService) GetAllServicesMap(profile string) map[string]map[string]string {
	if strings.TrimSpace(profile) == "" {
		profile = "local"
	}
	out := map[string]map[string]string{}
	for _, svc := range s.services {
		out[svc.Name] = s.compatibilityConfigFor(svc.Name, svc, profile)
	}
	return out
}

func (s *ConfigService) GetServiceByName(name string) (model.ServiceConfig, error) {
	key := normalizeLookup(name)
	for _, svc := range s.services {
		if normalizeLookup(svc.Name) == key {
			return svc, nil
		}
	}
	return model.ServiceConfig{}, errors.New("service not found")
}

func (s *ConfigService) GatewayConfig() model.GatewayConfig {
	return model.GatewayConfig{
		Port:               osOrEmpty("API_GATEWAY_PORT"),
		BaseURLLocal:       osOrEmpty("API_GATEWAY_BASE_URL_LOCAL"),
		BaseURLDeploy:      osOrEmpty("API_GATEWAY_BASE_URL_DEPLOY"),
		CORSAllowedOrigins: osOrEmpty("API_GATEWAY_CORS_ALLOWED_ORIGINS"),
		AuthRequired:       osOrEmpty("API_GATEWAY_AUTH_REQUIRED"),
		TimeoutSeconds:     osOrEmpty("API_GATEWAY_TIMEOUT_SECONDS"),
	}
}

func (s *ConfigService) KafkaConfig(profile string) model.KafkaConfig {
	produced := map[string][]string{}
	consumed := map[string][]string{}
	groups := map[string]string{}
	for _, svc := range s.services {
		produced[svc.Name] = producedTopicsFor(svc.Name)
		consumed[svc.Name] = consumedTopicsFor(svc.Name)
		groups[svc.Name] = consumerGroupFor(svc.Name)
	}

	sortTopics(produced)
	sortTopics(consumed)

	return model.KafkaConfig{
		BootstrapServers: resolveKafkaBootstrap(profile),
		SecurityProtocol: shared.EnvOrDefault("KAFKA_SECURITY_PROTOCOL", "PENDING_CONFIGURATION"),
		SASLMechanism:    shared.EnvOrDefault("KAFKA_SASL_MECHANISM", "NONE"),
		ProducedTopics:   produced,
		ConsumedTopics:   consumed,
		ConsumerGroups:   groups,
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

	common := map[string]string{
		"ENVIRONMENT":             shared.EnvOrDefault("ENVIRONMENT", "local"),
		"KAFKA_BOOTSTRAP_SERVERS": resolveKafkaBootstrap(profile),
		"KAFKA_SECURITY_PROTOCOL": shared.EnvOrDefault("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
		"KAFKA_SASL_MECHANISM":    shared.EnvOrDefault("KAFKA_SASL_MECHANISM", "NONE"),
		"CONFIG_SOURCE_PATH":      shared.EnvOrDefault("CONFIG_SOURCE_PATH", "Config"),
	}

	return model.RuntimeConfigResponse{
		Service:       svc.Name,
		Profile:       profile,
		ConfigVersion: "v1",
		Common:        common,
		ServiceConfig: svc,
		Kafka:         s.KafkaConfig(profile),
		Gateway:       s.GatewayConfig(),
		Metadata: map[string]interface{}{
			"served_at_utc": time.Now().UTC().Format(time.RFC3339),
			"language_hint": languageHint(svc.Name),
		},
	}, nil
}

func osOrEmpty(k string) string { return shared.EnvOrDefault(k, "") }

func sortTopics(m map[string][]string) {
	for _, topics := range m {
		sort.Strings(topics)
	}
}

func producedTopicsFor(service string) []string {
	switch normalizeLookup(service) {
	case "analytics-service":
		return []string{"analytics.anomaly.detected", "analytics.bill_prediction.generated", "analytics.consumption_ranking.generated", "analytics.device_identified", "analytics.recommendation.generated"}
	case "device-management-service":
		return []string{"device.configuration.updated", "device.event.recorded", "device.linked", "device.registered", "device.status.updated", "device.unlinked"}
	case "energy-monitoring-service":
		return []string{"monitoring.alert.created", "monitoring.reading.processed"}
	case "iam-service":
		return []string{"iam.role.assigned", "iam.user.logged-in", "iam.user.registered"}
	case "payments-service":
		return []string{"invoice.generated", "payment.failed", "payment.method.added", "payment.processed"}
	case "subscriptions-service":
		return []string{"SubscriptionCancelled", "SubscriptionCreated", "SubscriptionExpired", "SubscriptionPlanChanged", "SubscriptionUpdated"}
	default:
		return []string{}
	}
}

func consumedTopicsFor(service string) []string {
	switch normalizeLookup(service) {
	case "alert-service":
		return []string{"energy.consumption.recorded"}
	case "analytics-service":
		return []string{"device.registered", "device.updated", "energy.consumption.recorded"}
	case "energy-monitoring-service":
		return []string{"analytics.anomaly.detected", "monitoring.reading.ingest"}
	case "iam-service":
		return []string{"iam.role-assignment.requested"}
	case "payments-service":
		return []string{"subscription.cancelled", "subscription.created", "subscription.renewal.requested"}
	default:
		return []string{}
	}
}

func consumerGroupFor(service string) string {
	switch normalizeLookup(service) {
	case "alert-service":
		return "alert-service-group"
	case "analytics-service":
		return "analytics-service-group"
	case "device-management-service":
		return "device-management-group"
	case "energy-monitoring-service":
		return "energy-monitoring-group"
	case "iam-service":
		return "iam-service"
	case "payments-service":
		return "payments-service-group"
	case "subscriptions-service":
		return "subscriptions-service-group"
	default:
		return "PENDING_CONFIGURATION"
	}
}

func languageHint(service string) string {
	switch normalizeLookup(service) {
	case "iam-service":
		return "java"
	case "analytics-service", "energy-monitoring-service":
		return "python"
	default:
		return "go"
	}
}

func (s *ConfigService) GetServiceCompatibilityConfig(name, profile string) (map[string]string, error) {
	svc, err := s.GetServiceByName(name)
	if err != nil {
		return nil, err
	}
	return s.compatibilityConfigFor(name, svc, profile), nil
}

func (s *ConfigService) compatibilityConfigFor(name string, svc model.ServiceConfig, profile string) map[string]string {
	key := normalizeLookup(name)
	brokers := resolveKafkaBootstrap(profile)
	group := consumerGroupFor(key)
	if group == "PENDING_CONFIGURATION" {
		group = key + "-group"
	}
	consumed := consumedTopicsFor(key)
	consumptionTopic := firstOrDefault(consumed, "PENDING_CONFIGURATION")

	cfg := map[string]string{
		"serviceName":             svc.Name,
		"service_name":            svc.Name,
		"serverPort":              svc.LocalPort,
		"server_port":             svc.LocalPort,
		"base_url_local":          svc.BaseURLLocal,
		"route_prefix":            svc.RoutePrefix,
		"gateway_health_path":     gatewayHealthPathFor(key),
		"kafkaConsumerGroup":      group,
		"kafka_consumer_group":    group,
		"kafkaConsumptionTopic":   consumptionTopic,
		"kafka_consumption_topic": consumptionTopic,
		"kafkaBrokers":            brokers,
		"kafka_brokers":           brokers,
	}

	if key == "payments-service" {
		const paymentsPort = "8086"
		cfg["name"] = svc.Name
		cfg["port"] = paymentsPort
		cfg["serverPort"] = paymentsPort
		cfg["server_port"] = paymentsPort
		cfg["api_base_path"] = "/api/v1"
		cfg["stripe_currency"] = shared.EnvOrDefault("STRIPE_CURRENCY", "pen")
	}

	if key == "alert-service" {
		alertTopic := "alert.created"
		mailHost := shared.EnvOrDefault("MAIL_HOST", "smtp.gmail.com")
		mailFrom := shared.EnvOrDefault("MAIL_FROM", "PENDING_CONFIGURATION")
		cfg["kafkaAlertCreatedTopic"] = alertTopic
		cfg["kafka_alert_created_topic"] = alertTopic
		cfg["mailHost"] = mailHost
		cfg["mail_host"] = mailHost
		cfg["mailFrom"] = mailFrom
		cfg["mail_from"] = mailFrom
	}

	if key == "energy-monitoring-service" {
		energyBrokers := resolveEnergyKafkaBootstrap(profile)
		kafkaSecurityProtocol := shared.EnvOrDefault("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT")
		kafkaSASLMechanism := shared.EnvOrDefault("KAFKA_SASL_MECHANISM", "NONE")
		kafkaGroupID := consumerGroupFor(key)
		if kafkaGroupID == "PENDING_CONFIGURATION" {
			kafkaGroupID = "energy-monitoring-group"
		}

		cfg["app.host"] = shared.EnvOrDefault("ENERGY_MONITORING_APP_HOST", "0.0.0.0")
		cfg["app.port"] = svc.LocalPort
		cfg["app.env"] = normalizeLookup(profile)
		cfg["api.base_path"] = svc.RoutePrefix
		cfg["mongodb.database"] = shared.EnvOrDefault("MONGODB_DATABASE", "PENDING_CONFIGURATION")
		cfg["kafka.bootstrap_servers"] = energyBrokers
		cfg["kafka.group_id"] = kafkaGroupID
		cfg["kafka.security_protocol"] = kafkaSecurityProtocol
		cfg["kafka.sasl_mechanism"] = kafkaSASLMechanism
		cfg["kafka.topics.reading_ingest"] = "monitoring.reading.ingest"
		cfg["kafka.topics.anomaly_detected"] = "analytics.anomaly.detected"
		cfg["kafka.topics.alert_created"] = "monitoring.alert.created"
		cfg["kafka.topics.reading_processed"] = "monitoring.reading.processed"
	}

	return cfg
}

func firstOrDefault(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func normalizeLookup(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "`", "")
	v = strings.ReplaceAll(v, "microservice-", "")
	v = strings.ReplaceAll(v, "_", "-")
	return v
}

func gatewayHealthPathFor(service string) string {
	switch normalizeLookup(service) {
	case "iam-service":
		return "/api/v1/iam/health"
	case "analytics-service":
		return "/api/v1/analytics/health"
	case "device-management-service":
		return "/api/v1/device-management/health"
	case "alert-service":
		return "/api/v1/alerts-service/health"
	case "subscriptions-service":
		return "/api/v1/subscriptions/health"
	case "payments-service":
		return "/api/v1/payments/health"
	case "energy-monitoring-service":
		return "/api/v1/energy/health"
	default:
		return "/api/v1/health"
	}
}

func resolveKafkaBootstrap(profile string) string {
	switch normalizeLookup(profile) {
	case "docker", "container", "compose":
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_DOCKER", shared.EnvOrDefault("KAFKA_BROKERS", "kafka:9092"))
	default:
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_LOCAL", shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS", shared.EnvOrDefault("KAFKA_BROKERS", "localhost:9092")))
	}
}

func resolveEnergyKafkaBootstrap(profile string) string {
	switch normalizeLookup(profile) {
	case "docker", "container", "compose":
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_DOCKER", shared.EnvOrDefault("KAFKA_BROKERS", "kafka:9092"))
	default:
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_LOCAL", shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS", shared.EnvOrDefault("KAFKA_BROKERS", "localhost:9092")))
	}
}
