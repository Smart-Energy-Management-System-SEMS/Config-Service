package rest

import (
	"net/http"
	"strings"

	"config-service/config-service/application/services"
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
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"services":     h.service.GetAllServices(),
		"services_map": h.service.GetAllServicesMap(),
	})
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
	httpx.WriteJSON(w, http.StatusOK, h.service.KafkaConfig(profile))
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
