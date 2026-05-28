package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	"config-service/config-service/application/services"
	"config-service/config-service/domain/model"
	httpx "config-service/config-service/infrastructure/http"
)

type Handler struct {
	service *services.ConfigService
}

func NewHandler(service *services.ConfigService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/config/health", h.health)
	mux.HandleFunc("/api/v1/config/services", h.services)
	mux.HandleFunc("/api/v1/config/services/", h.serviceByName)
	mux.HandleFunc("/api/v1/config/runtime/", h.runtimeConfig)
	mux.HandleFunc("/api/v1/config/kafka", h.kafka)
	mux.HandleFunc("/api/v1/config/kafka/publish", h.publishKafka)
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
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"services": h.service.GetAllServices()})
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
	service, err := h.service.GetServiceByName(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, service)
}

func (h *Handler) kafka(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.service.KafkaConfig())
}

func (h *Handler) gateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.service.GatewayConfig())
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

func (h *Handler) publishKafka(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req model.KafkaPublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	if strings.TrimSpace(req.Topic) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "topic is required")
		return
	}

	if err := h.service.PublishTestEvent(r.Context(), req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]string{
		"status": "published",
		"topic":  req.Topic,
	})
}
