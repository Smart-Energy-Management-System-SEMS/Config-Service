package services

import (
	"errors"
	"sort"
	"strings"

	"config-service/domain/model"
	"config-service/infrastructure/configuration"
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

func (s *ConfigService) GetServiceByName(name string) (model.ServiceConfig, error) {
	for _, svc := range s.services {
		if strings.EqualFold(svc.Name, name) {
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

func (s *ConfigService) KafkaConfig() model.KafkaConfig {
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
		BootstrapServers: shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS", "PENDING_CONFIGURATION"),
		SecurityProtocol: shared.EnvOrDefault("KAFKA_SECURITY_PROTOCOL", "PENDING_CONFIGURATION"),
		SASLMechanism:    shared.EnvOrDefault("KAFKA_SASL_MECHANISM", "PENDING_CONFIGURATION"),
		ProducedTopics:   produced,
		ConsumedTopics:   consumed,
		ConsumerGroups:   groups,
	}
}

func osOrEmpty(k string) string { return shared.EnvOrDefault(k, "") }

func sortTopics(m map[string][]string) {
	for _, topics := range m {
		sort.Strings(topics)
	}
}

func producedTopicsFor(service string) []string {
	switch strings.ToLower(service) {
	case "microservice-analytics-service", "analytics-service":
		return []string{"analytics.anomaly.detected", "analytics.bill_prediction.generated", "analytics.consumption_ranking.generated", "analytics.device_identified", "analytics.recommendation.generated"}
	case "device-management-service":
		return []string{"device.configuration.updated", "device.event.recorded", "device.linked", "device.registered", "device.status.updated", "device.unlinked"}
	case "microservice-energy-monitoring-service", "energy-monitoring-service":
		return []string{"monitoring.alert.created", "monitoring.reading.processed"}
	case "iam-service":
		return []string{"iam.role.assigned", "iam.user.logged-in", "iam.user.registered"}
	case "payments-service":
		return []string{"invoice.generated", "payment.failed", "payment.method.added", "payment.processed"}
	case "microservice-subscriptions-service", "subscriptions-service":
		return []string{"SubscriptionCancelled", "SubscriptionCreated", "SubscriptionExpired", "SubscriptionPlanChanged", "SubscriptionUpdated"}
	default:
		return []string{}
	}
}

func consumedTopicsFor(service string) []string {
	switch strings.ToLower(service) {
	case "alert-service":
		return []string{"energy.consumption.recorded"}
	case "microservice-analytics-service", "analytics-service":
		return []string{"device.registered", "device.updated", "energy.consumption.recorded"}
	case "microservice-energy-monitoring-service", "energy-monitoring-service":
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
	switch strings.ToLower(service) {
	case "alert-service":
		return "alert-service-group"
	case "microservice-analytics-service", "analytics-service":
		return "analytics-service-group"
	case "device-management-service":
		return "device-management-group"
	case "microservice-energy-monitoring-service", "energy-monitoring-service":
		return "energy-monitoring-group"
	case "iam-service":
		return "iam-service"
	case "payments-service":
		return "payments-service-group"
	case "microservice-subscriptions-service", "subscriptions-service":
		return "PENDING_CONFIGURATION"
	default:
		return "PENDING_CONFIGURATION"
	}
}
