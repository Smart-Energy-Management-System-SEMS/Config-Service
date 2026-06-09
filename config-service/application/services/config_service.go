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

var officialTopics = []string{
	"alert.created",
	"analytics.anomaly.detected",
	"analytics.bill_prediction.generated",
	"analytics.consumption_ranking.generated",
	"analytics.device_identified",
	"analytics.recommendation.generated",
	"device.configuration.updated",
	"device.event.recorded",
	"device.linked",
	"device.registered",
	"device.status.updated",
	"device.unlinked",
	"energy.consumption.recorded",
	"energy.reading.created",
	"iam.role-assignment.requested",
	"iam.role.assigned",
	"iam.user.logged-in",
	"iam.user.registered",
	"invoice.generated",
	"monitoring.alert.created",
	"monitoring.reading.ingest",
	"monitoring.reading.processed",
	"payment.failed",
	"payment.method.added",
	"payment.processed",
	"subscription.cancelled",
	"subscription.created",
	"subscription.expired",
	"subscription.plan.changed",
	"subscription.renewal.requested",
	"subscription.updated",
}

type ConfigService struct {
	services        []model.ServiceConfig
	inconsistencies []model.TopicInconsistency
}

func NewConfigService(loader *configuration.Loader) (*ConfigService, error) {
	services, err := loader.LoadServices()
	if err != nil {
		return nil, err
	}

	filtered, inconsistencies := normalizeServices(services)
	return &ConfigService{
		services:        filtered,
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
	key := normalizeLookup(name)
	for _, svc := range s.services {
		if normalizeLookup(svc.Name) == key {
			return svc, nil
		}
	}
	return model.ServiceConfig{}, errors.New("service not found")
}

func (s *ConfigService) GetInconsistencies() []model.TopicInconsistency {
	out := make([]model.TopicInconsistency, len(s.inconsistencies))
	copy(out, s.inconsistencies)
	return out
}

func (s *ConfigService) GatewayConfig() model.GatewayConfig {
	return model.GatewayConfig{
		Port:               osOrEmpty("API_GATEWAY_PORT"),
		BaseURLLocal:       getEnv("API_GATEWAY_URL", "http://localhost:8081"),
		BaseURLDeploy:      osOrEmpty("API_GATEWAY_BASE_URL_DEPLOY"),
		CORSAllowedOrigins: osOrEmpty("API_GATEWAY_CORS_ALLOWED_ORIGINS"),
		AuthRequired:       osOrEmpty("API_GATEWAY_AUTH_REQUIRED"),
		TimeoutSeconds:     osOrEmpty("API_GATEWAY_TIMEOUT_SECONDS"),
		Services:           s.GetAllServices(),
	}
}

func (s *ConfigService) KafkaConfig(profile string) model.KafkaConfig {
	produced := map[string][]string{}
	consumed := map[string][]string{}
	for _, svc := range s.services {
		produced[svc.Name] = append([]string{}, svc.TopicsPublished...)
		consumed[svc.Name] = append([]string{}, svc.TopicsConsumed...)
	}

	return model.KafkaConfig{
		BootstrapServers: resolveKafkaBootstrap(profile),
		Brokers:          splitCSV(resolveKafkaBootstrap(profile)),
		SecurityProtocol: getEnv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
		SASLMechanism:    getEnv("KAFKA_SASL_MECHANISM", ""),
		Username:         getEnv("KAFKA_USERNAME", ""),
		Password:         getEnv("KAFKA_PASSWORD", ""),
		ClientID:         getEnv("KAFKA_CLIENT_ID", ""),
		ConsumerGroup:    getEnv("KAFKA_CONSUMER_GROUP", ""),
		OfficialTopics:   append([]string{}, officialTopics...),
		ProducedTopics:   produced,
		ConsumedTopics:   consumed,
		Inconsistencies:  s.GetInconsistencies(),
	}
}

func (s *ConfigService) ServiceEndpoints(profile string) map[string]string {
	profile = normalizeProfile(profile)
	azure := profile == "azure"

	return map[string]string{
		"apiGatewayUrl":              resolveServiceURL(azure, "http://localhost:8081", "API_GATEWAY_URL_AZURE", "API_GATEWAY_URL"),
		"iamServiceUrl":              resolveServiceURL(azure, serviceURL(s.services, "iam-service", "http://localhost:8082"), "IAM_SERVICE_URL_AZURE", "IAM_SERVICE_URL"),
		"deviceManagementServiceUrl": resolveServiceURL(azure, serviceURL(s.services, "device-management-service", "http://localhost:8083"), "DEVICE_SERVICE_URL_AZURE", "DEVICE_SERVICE_URL", "DEVICE_MANAGEMENT_SERVICE_URL_AZURE", "DEVICE_MANAGEMENT_SERVICE_URL"),
		"subscriptionsServiceUrl":    resolveServiceURL(azure, serviceURL(s.services, "subscriptions-service", "http://localhost:18083"), "SUBSCRIPTIONS_SERVICE_URL_AZURE", "SUBSCRIPTIONS_SERVICE_URL"),
		"paymentsServiceUrl":         resolveServiceURL(azure, serviceURL(s.services, "payments-service", "http://localhost:8086"), "PAYMENTS_SERVICE_URL_AZURE", "PAYMENTS_SERVICE_URL"),
		"alertServiceUrl":            resolveServiceURL(azure, serviceURL(s.services, "alert-service", "http://localhost:8085"), "ALERTS_SERVICE_URL_AZURE", "ALERTS_SERVICE_URL", "ALERT_SERVICE_URL_AZURE", "ALERT_SERVICE_URL"),
		"analyticsServiceUrl":        resolveServiceURL(azure, serviceURL(s.services, "analytics-service", "http://localhost:8004"), "ANALYTICS_SERVICE_URL_AZURE", "ANALYTICS_SERVICE_URL"),
		"energyMonitoringServiceUrl": resolveServiceURL(azure, serviceURL(s.services, "energy-monitoring-service", "http://localhost:8001"), "ENERGY_SERVICE_URL_AZURE", "ENERGY_SERVICE_URL", "ENERGY_MONITORING_SERVICE_URL_AZURE", "ENERGY_MONITORING_SERVICE_URL"),
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

	common := map[string]string{
		"ENVIRONMENT":             getEnv("ENVIRONMENT", "local"),
		"KAFKA_BOOTSTRAP_SERVERS": resolveKafkaBootstrap(profile),
		"KAFKA_BROKERS":           resolveKafkaBootstrap(profile),
		"KAFKA_SECURITY_PROTOCOL": getEnv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
		"KAFKA_SASL_MECHANISM":    getEnv("KAFKA_SASL_MECHANISM", "NONE"),
		"KAFKA_USERNAME":          getEnv("KAFKA_USERNAME", ""),
		"KAFKA_PASSWORD":          getEnv("KAFKA_PASSWORD", ""),
		"KAFKA_CLIENT_ID":         getEnv("KAFKA_CLIENT_ID", ""),
		"KAFKA_CONSUMER_GROUP":    getEnv("KAFKA_CONSUMER_GROUP", ""),
		"CONFIG_SOURCE_PATH":      getEnv("CONFIG_SOURCE_PATH", "Rutas"),
	}

	return model.RuntimeConfigResponse{
		Service:       svc.Name,
		Profile:       profile,
		ConfigVersion: "v2",
		Common:        common,
		ServiceConfig: svc,
		Kafka:         s.KafkaConfig(profile),
		Gateway:       s.GatewayConfig(),
		Metadata: map[string]interface{}{
			"served_at_utc":    time.Now().UTC().Format(time.RFC3339),
			"inconsistencies":  s.GetInconsistencies(),
			"official_topics":  append([]string{}, officialTopics...),
			"source_directory": shared.EnvOrDefault("CONFIG_SOURCE_PATH", "Rutas"),
		},
	}, nil
}

func normalizeServices(services []model.ServiceConfig) ([]model.ServiceConfig, []model.TopicInconsistency) {
	allowed := officialTopicSet()
	inconsistencies := []model.TopicInconsistency{}
	out := make([]model.ServiceConfig, 0, len(services))

	for _, svc := range services {
		svc.TopicsPublished, inconsistencies = filterTopics(svc.Name, "published", svc.TopicsPublished, allowed, inconsistencies)
		svc.TopicsConsumed, inconsistencies = filterTopics(svc.Name, "consumed", svc.TopicsConsumed, allowed, inconsistencies)
		svc.Routes = uniqueStrings(svc.Routes)
		svc.TopicsPublished = uniqueStrings(svc.TopicsPublished)
		svc.TopicsConsumed = uniqueStrings(svc.TopicsConsumed)
		svc.HealthAliases = uniqueStrings(svc.HealthAliases)
		out = append(out, svc)
	}

	sort.Slice(inconsistencies, func(i, j int) bool {
		if inconsistencies[i].Service == inconsistencies[j].Service {
			if inconsistencies[i].Type == inconsistencies[j].Type {
				return inconsistencies[i].Topic < inconsistencies[j].Topic
			}
			return inconsistencies[i].Type < inconsistencies[j].Type
		}
		return inconsistencies[i].Service < inconsistencies[j].Service
	})

	return out, inconsistencies
}

func filterTopics(serviceName, topicType string, topics []string, allowed map[string]struct{}, acc []model.TopicInconsistency) ([]string, []model.TopicInconsistency) {
	valid := make([]string, 0, len(topics))
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		if _, ok := allowed[topic]; !ok {
			acc = append(acc, model.TopicInconsistency{
				Service: serviceName,
				Topic:   topic,
				Type:    topicType,
				Reason:  "topic is not in the official allowed list",
			})
			continue
		}
		valid = append(valid, topic)
	}
	return valid, acc
}

func officialTopicSet() map[string]struct{} {
	out := make(map[string]struct{}, len(officialTopics))
	for _, topic := range officialTopics {
		out[topic] = struct{}{}
	}
	return out
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

func osOrEmpty(k string) string { return shared.EnvOrDefault(k, "") }

func getEnv(key, fallback string) string { return shared.EnvOrDefault(key, fallback) }

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
	v = strings.ReplaceAll(v, "microservice-", "")
	v = strings.ReplaceAll(v, "_", "-")
	return v
}

func resolveKafkaBootstrap(profile string) string {
	switch normalizeProfile(profile) {
	case "docker", "container", "compose":
		return getEnv("KAFKA_BOOTSTRAP_SERVERS_DOCKER", getEnv("KAFKA_BROKERS", "localhost:9092"))
	case "azure", "prod", "production":
		return getEnv("KAFKA_BROKERS_AZURE", getEnv("KAFKA_BROKERS", "localhost:9092"))
	default:
		return getEnv("KAFKA_BOOTSTRAP_SERVERS_LOCAL", getEnv("KAFKA_BOOTSTRAP_SERVERS", getEnv("KAFKA_BROKERS", "localhost:9092")))
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

func resolveServiceURL(azure bool, fallback string, keys ...string) string {
	value := fallback
	for i := len(keys) - 1; i >= 0; i-- {
		key := strings.TrimSpace(keys[i])
		if key == "" {
			continue
		}
		if strings.HasSuffix(key, "_AZURE") && !azure {
			continue
		}
		value = getEnv(key, value)
	}
	return value
}

func serviceURL(services []model.ServiceConfig, name, fallback string) string {
	for _, svc := range services {
		if normalizeLookup(svc.Name) == normalizeLookup(name) && strings.TrimSpace(svc.URL) != "" {
			return svc.URL
		}
	}
	return fallback
}
