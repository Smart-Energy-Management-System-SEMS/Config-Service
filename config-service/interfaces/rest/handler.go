package rest

import (
	"net/http"
	"sort"
	"strings"

	"config-service/config-service/application/services"
	httpx "config-service/config-service/infrastructure/http"
	"config-service/shared"
)

type Handler struct {
	service *services.ConfigService
}

func NewHandler(service *services.ConfigService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/health", h.health)
	mux.HandleFunc("/api/v1/config/health", h.health)
	mux.HandleFunc("/api/v1/config/services", h.services)
	mux.HandleFunc("/api/v1/config/services/", h.serviceByName)
	mux.HandleFunc("/api/v1/config/", h.configByName)
	mux.HandleFunc("/api/v1/config/runtime/", h.runtimeConfig)
	mux.HandleFunc("/api/v1/config/kafka", h.kafka)
	mux.HandleFunc("/api/v1/config/api-gateway", h.gateway)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.service.Health())
}

func (h *Handler) services(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	profile := strings.TrimSpace(r.URL.Query().Get("profile"))
	if profile == "" {
		profile = strings.TrimSpace(r.URL.Query().Get("env"))
	}
	if profile == "" {
		profile = "local"
	}
	payload := map[string]any{}
	for key, value := range h.service.ServiceEndpoints(profile) {
		payload[key] = value
	}
	payload["services"] = h.service.GetAllServices()
	payload["services_map"] = h.service.GetAllServicesMap(profile)

	httpx.WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) serviceByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/config/services/"))
	if name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "service name is required")
		return
	}
	profile := strings.TrimSpace(r.URL.Query().Get("profile"))
	if profile == "" {
		profile = strings.TrimSpace(r.URL.Query().Get("env"))
	}
	if profile == "" {
		profile = "local"
	}
	service, err := h.service.GetServiceCompatibilityConfig(name, profile)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, service)
}

func (h *Handler) configByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	name := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/config/"))
	if name == "" || strings.Contains(name, "/") {
		httpx.WriteError(w, http.StatusNotFound, "route not found")
		return
	}

	switch name {
	case "health", "services", "kafka", "api-gateway", "runtime":
		httpx.WriteError(w, http.StatusNotFound, "route not found")
		return
	}

	profile := strings.TrimSpace(r.URL.Query().Get("profile"))
	if profile == "" {
		profile = strings.TrimSpace(r.URL.Query().Get("env"))
	}
	if profile == "" {
		profile = "local"
	}

	payload, err := h.service.GetServiceCompatibilityConfig(name, profile)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) kafka(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	profile := strings.TrimSpace(r.URL.Query().Get("profile"))
	if profile == "" {
		profile = strings.TrimSpace(r.URL.Query().Get("env"))
	}
	if profile == "" {
		profile = "local"
	}
	cfg := h.service.KafkaConfig(profile)
	brokers := splitCSV(cfg.BootstrapServers)
	servicesMap := h.service.GetAllServicesMap(profile)
	officialTopics := map[string]any{}
	allTopicNames := map[string]struct{}{}
	for serviceName, serviceCfg := range servicesMap {
		shortName := serviceCfg["service"]
		if shortName == nil {
			shortName = serviceName
		}
		officialTopics[shortName.(string)] = serviceCfg["kafka"]
		kafkaCfg, _ := serviceCfg["kafka"].(map[string]any)
		for _, key := range []string{"consumerTopics", "producerTopics"} {
			topicMap, _ := kafkaCfg[key].(map[string]string)
			for _, topicName := range topicMap {
				allTopicNames[topicName] = struct{}{}
			}
		}
	}
	flatTopics := make([]string, 0, len(allTopicNames))
	for topicName := range allTopicNames {
		flatTopics = append(flatTopics, topicName)
	}
	sort.Strings(flatTopics)
	saslUsername := shared.EnvOrDefault("KAFKA_USERNAME", shared.EnvOrDefault("KAFKA_SASL_USERNAME", ""))
	saslPassword := shared.EnvOrDefault("KAFKA_PASSWORD", shared.EnvOrDefault("KAFKA_SASL_PASSWORD", ""))
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"kafkaBrokers":       cfg.BootstrapServers,
		"brokers":            cfg.BootstrapServers,
		"securityProtocol":   cfg.SecurityProtocol,
		"saslMechanism":      cfg.SASLMechanism,
		"saslUsername":       saslUsername,
		"saslPassword":       saslPassword,
		"bootstrapServers":   cfg.BootstrapServers,
		"bootstrap_servers":  brokers,
		"client_id":          "config-service",
		"security_protocol":  cfg.SecurityProtocol,
		"sasl_mechanism":     cfg.SASLMechanism,
		"officialTopics":     officialTopics,
		"officialTopicNames": flatTopics,
		"services":           officialTopics,
		"topics": map[string]string{
			"payment_processed":              "payment.processed",
			"payment_failed":                 "payment.failed",
			"invoice_generated":              "invoice.generated",
			"payment_method_added":           "payment.method.added",
			"subscription_created":           "subscription.created",
			"subscription_renewal_requested": "subscription.renewal.requested",
			"subscription_cancelled":         "subscription.cancelled",
		},
		"producedTopics":  cfg.ProducedTopics,
		"consumedTopics":  cfg.ConsumedTopics,
		"consumerGroups":  cfg.ConsumerGroups,
		"produced_topics": cfg.ProducedTopics,
		"consumed_topics": cfg.ConsumedTopics,
		"consumer_groups": cfg.ConsumerGroups,
	})
}

func (h *Handler) gateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.service.GatewayConfig())
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

func (h *Handler) runtimeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/config/runtime/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "use /api/v1/config/runtime/{serviceName}/{profile}")
		return
	}

	resp, err := h.service.GetRuntimeConfig(parts[0], parts[1])
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
