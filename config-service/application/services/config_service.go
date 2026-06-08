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

func (s *ConfigService) GetAllServicesMap(profile string) map[string]map[string]any {
	if strings.TrimSpace(profile) == "" {
		profile = "local"
	}
	out := map[string]map[string]any{}
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
		SecurityProtocol: shared.EnvOrDefault("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
		SASLMechanism:    shared.EnvOrDefault("KAFKA_SASL_MECHANISM", ""),
		ProducedTopics:   produced,
		ConsumedTopics:   consumed,
		ConsumerGroups:   groups,
	}
}

func (s *ConfigService) ServiceEndpoints(profile string) map[string]string {
	profile = normalizeProfile(profile)
	azure := profile == "azure"

	deviceLocal := serviceLocalURL(s.services, "device-management-service", "http://localhost:8083")
	alertLocal := serviceLocalURL(s.services, "alert-service", "http://localhost:8085")
	analyticsLocal := serviceLocalURL(s.services, "analytics-service", "http://localhost:8004")
	energyLocal := serviceLocalURL(s.services, "energy-monitoring-service", "http://localhost:8001")
	iamLocal := serviceLocalURL(s.services, "iam-service", "http://localhost:8082")
	subscriptionsLocal := serviceLocalURL(s.services, "subscriptions-service", "http://localhost:18083")
	paymentsLocal := serviceLocalURL(s.services, "payments-service", "http://localhost:8086")

	return map[string]string{
		"apiGatewayUrl":              resolveServiceURL("API_GATEWAY_URL", "API_GATEWAY_URL_AZURE", "http://localhost:8081", azure),
		"iamServiceUrl":              resolveServiceURL("IAM_SERVICE_URL", "IAM_SERVICE_URL_AZURE", iamLocal, azure),
		"deviceManagementServiceUrl": resolveServiceURL("DEVICE_MANAGEMENT_SERVICE_URL", "DEVICE_MANAGEMENT_SERVICE_URL_AZURE", deviceLocal, azure),
		"subscriptionsServiceUrl":    resolveServiceURL("SUBSCRIPTIONS_SERVICE_URL", "SUBSCRIPTIONS_SERVICE_URL_AZURE", subscriptionsLocal, azure),
		"paymentsServiceUrl":         resolveServiceURL("PAYMENTS_SERVICE_URL", "PAYMENTS_SERVICE_URL_AZURE", paymentsLocal, azure),
		"alertServiceUrl":            resolveServiceURL("ALERT_SERVICE_URL", "ALERT_SERVICE_URL_AZURE", alertLocal, azure),
		"analyticsServiceUrl":        resolveServiceURL("ANALYTICS_SERVICE_URL", "ANALYTICS_SERVICE_URL_AZURE", analyticsLocal, azure),
		"energyMonitoringServiceUrl": resolveServiceURL("ENERGY_MONITORING_SERVICE_URL", "ENERGY_MONITORING_SERVICE_URL_AZURE", energyLocal, azure),
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
		return []string{"energy.reading.created", "monitoring.alert.created", "monitoring.reading.processed"}
	case "iam-service":
		return []string{"iam.role.assigned", "iam.user.logged-in", "iam.user.registered"}
	case "payments-service":
		return []string{"invoice.generated", "payment.failed", "payment.method.added", "payment.processed"}
	case "subscriptions-service":
		return []string{"subscription.cancelled", "subscription.created", "subscription.expired", "subscription.plan.changed", "subscription.updated"}
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

func (s *ConfigService) GetServiceCompatibilityConfig(name, profile string) (map[string]any, error) {
	svc, err := s.GetServiceByName(name)
	if err != nil {
		return nil, err
	}
	return s.compatibilityConfigFor(name, svc, profile), nil
}

func (s *ConfigService) compatibilityConfigFor(name string, svc model.ServiceConfig, profile string) map[string]any {
	key := normalizeLookup(name)
	brokers := resolveKafkaBootstrap(profile)
	group := consumerGroupFor(key)
	if group == "PENDING_CONFIGURATION" {
		group = key + "-group"
	}
	produced := producedTopicsFor(key)
	consumed := consumedTopicsFor(key)
	consumptionTopic := firstOrDefault(consumed, "PENDING_CONFIGURATION")

	cfg := map[string]any{
		"serviceName":             svc.Name,
		"service_name":            svc.Name,
		"serverPort":              svc.LocalPort,
		"server_port":             svc.LocalPort,
		"base_url_local":          svc.BaseURLLocal,
		"route_prefix":            svc.RoutePrefix,
		"routePrefixes":           strings.Join(routePrefixesFor(key), ","),
		"route_prefixes":          strings.Join(routePrefixesFor(key), ","),
		"gatewayRouteBlocks":      strings.Join(gatewayRouteBlocksFor(key), ","),
		"gateway_route_blocks":    strings.Join(gatewayRouteBlocksFor(key), ","),
		"gateway_health_path":     gatewayHealthPathFor(key),
		"kafkaConsumerGroup":      group,
		"kafka_consumer_group":    group,
		"kafkaConsumptionTopic":   consumptionTopic,
		"kafka_consumption_topic": consumptionTopic,
		"kafkaBrokers":            brokers,
		"kafka_brokers":           brokers,
	}

	cfg["kafka"] = serviceKafkaBlock(key, brokers, group, produced, consumed)

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
		cfg["kafka.topics.energy_reading_created"] = "energy.reading.created"
	}

	if key == "device-management-service" {
		cfg["kafkaTopics"] = map[string]string{
			"deviceRegistered":          "device.registered",
			"deviceStatusUpdated":       "device.status.updated",
			"deviceLinked":              "device.linked",
			"deviceUnlinked":            "device.unlinked",
			"deviceConfigurationUpdated": "device.configuration.updated",
			"deviceEventRecorded":       "device.event.recorded",
		}
	}

	if key == "subscriptions-service" {
		cfg["topicSubscriptionCreated"] = "subscription.created"
		cfg["topicSubscriptionCancelled"] = "subscription.cancelled"
		cfg["topicSubscriptionPlanChanged"] = "subscription.plan.changed"
		cfg["topicSubscriptionExpired"] = "subscription.expired"
		cfg["topicSubscriptionUpdated"] = "subscription.updated"
		cfg["kafkaTopicSubscriptionCreated"] = "subscription.created"
		cfg["kafkaTopicSubscriptionCancelled"] = "subscription.cancelled"
		cfg["kafkaTopicSubscriptionPlanChanged"] = "subscription.plan.changed"
		cfg["kafkaTopicSubscriptionExpired"] = "subscription.expired"
		cfg["kafkaTopicSubscriptionUpdated"] = "subscription.updated"
	}

	return cfg
}

func routePrefixesFor(service string) []string {
	switch normalizeLookup(service) {
	case "device-management-service":
		return []string{"/api/v1/device-management"}
	case "subscriptions-service":
		return []string{"/api/v1/subscription-plans", "/api/v1/subscriptions", "/api/v1/webhooks"}
	case "alert-service":
		return []string{"/api/v1/alerts", "/api/v1/thresholds", "/api/v1/inactivity-rules", "/api/v1/notification-preferences"}
	case "payments-service":
		return []string{"/api/v1/payment-methods", "/api/v1/payments", "/api/v1/invoices", "/api/v1/webhooks"}
	case "energy-monitoring-service":
		return []string{"/api/v1/energy", "/api/v1/energy-readings", "/api/v1/energy-meters", "/api/v1/device-consumptions", "/api/v1/consumption-alerts"}
	case "analytics-service":
		return []string{"/api/v1/analytics"}
	case "iam-service":
		return []string{"/api/v1/auth", "/api/v1/users"}
	default:
		return []string{"/api/v1"}
	}
}

func gatewayRouteBlocksFor(service string) []string {
	prefixes := routePrefixesFor(service)
	blocks := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		blocks = append(blocks, prefix+"/**")
	}
	return blocks
}

func firstOrDefault(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func serviceKafkaBlock(service, brokers, group string, produced, consumed []string) map[string]any {
	block := map[string]any{
		"bootstrapServers": brokers,
		"bootstrap_servers": brokers,
		"consumerGroup": group,
		"consumer_group": group,
		"consumedTopics": consumed,
		"consumed_topics": consumed,
		"producedTopics": producedTopicMap(service, produced),
		"produced_topics": producedTopicMap(service, produced),
	}

	if normalizeLookup(service) == "device-management-service" {
		block["kafkaTopics"] = map[string]string{
			"deviceRegistered":           "device.registered",
			"deviceStatusUpdated":        "device.status.updated",
			"deviceLinked":               "device.linked",
			"deviceUnlinked":             "device.unlinked",
			"deviceConfigurationUpdated": "device.configuration.updated",
			"deviceEventRecorded":        "device.event.recorded",
		}
	}

	return block
}

func producedTopicMap(service string, topics []string) map[string]string {
	out := map[string]string{}
	for _, topic := range topics {
		out[topic] = topic
	}

	switch normalizeLookup(service) {
	case "analytics-service":
		out["billPredictionGenerated"] = "analytics.bill_prediction.generated"
		out["recommendationGenerated"] = "analytics.recommendation.generated"
		out["anomalyDetected"] = "analytics.anomaly.detected"
		out["deviceIdentified"] = "analytics.device_identified"
		out["consumptionRankingGenerated"] = "analytics.consumption_ranking.generated"
	case "subscriptions-service":
		out["subscriptionCreated"] = "subscription.created"
		out["subscriptionCancelled"] = "subscription.cancelled"
		out["subscriptionPlanChanged"] = "subscription.plan.changed"
		out["subscriptionExpired"] = "subscription.expired"
		out["subscriptionUpdated"] = "subscription.updated"
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

func gatewayHealthPathFor(service string) string {
	switch normalizeLookup(service) {
	case "iam-service":
		return "/health"
	case "analytics-service":
		return "/api/v1/analytics/health"
	case "device-management-service":
		return "/api/v1/device-management/health"
	case "alert-service":
		return "/api/v1/health"
	case "subscriptions-service":
		return "/api/v1/health"
	case "payments-service":
		return "/api/v1/health"
	case "energy-monitoring-service":
		return "/api/v1/health"
	default:
		return "/api/v1/health"
	}
}

func resolveKafkaBootstrap(profile string) string {
	switch normalizeProfile(profile) {
	case "docker", "container", "compose":
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_DOCKER", shared.EnvOrDefault("KAFKA_BROKERS", "kafka:9092"))
	case "azure", "prod", "production":
		return shared.EnvOrDefault("KAFKA_BROKERS_AZURE", shared.EnvOrDefault("KAFKA_BROKERS", "localhost:9092"))
	default:
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_LOCAL", shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS", shared.EnvOrDefault("KAFKA_BROKERS", "localhost:9092")))
	}
}

func resolveEnergyKafkaBootstrap(profile string) string {
	switch normalizeProfile(profile) {
	case "docker", "container", "compose":
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_DOCKER", shared.EnvOrDefault("KAFKA_BROKERS", "kafka:9092"))
	case "azure", "prod", "production":
		return shared.EnvOrDefault("KAFKA_BROKERS_AZURE", shared.EnvOrDefault("KAFKA_BROKERS", "localhost:9092"))
	default:
		return shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS_LOCAL", shared.EnvOrDefault("KAFKA_BOOTSTRAP_SERVERS", shared.EnvOrDefault("KAFKA_BROKERS", "localhost:9092")))
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

func resolveServiceURL(localKey, azureKey, fallback string, azure bool) string {
	if azure {
		return shared.EnvOrDefault(azureKey, shared.EnvOrDefault(localKey, fallback))
	}
	return shared.EnvOrDefault(localKey, fallback)
}

func serviceLocalURL(services []model.ServiceConfig, name, fallback string) string {
	for _, svc := range services {
		if normalizeLookup(svc.Name) == normalizeLookup(name) && strings.TrimSpace(svc.BaseURLLocal) != "" {
			return svc.BaseURLLocal
		}
	}
	return fallback
}
