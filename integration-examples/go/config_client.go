package configclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type RuntimeConfig struct {
	Service       string            `json:"service"`
	Profile       string            `json:"profile"`
	ConfigVersion string            `json:"config_version"`
	Common        map[string]string `json:"common"`
	ServiceConfig struct {
		Name         string   `json:"name"`
		LocalPort    string   `json:"local_port"`
		BaseURLLocal string   `json:"base_url_local"`
		RoutePrefix  string   `json:"route_prefix"`
		MainEndpoints []string `json:"main_endpoints"`
	} `json:"service_config"`
}

type Bootstrap struct {
	ConfigServiceURL string
	ServiceName      string
	Profile          string
	Timeout          time.Duration
	FailFast         bool
}

func LoadBootstrap() Bootstrap {
	timeoutSec := 5
	if raw := os.Getenv("CONFIG_FETCH_TIMEOUT_SECONDS"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			timeoutSec = v
		}
	}
	return Bootstrap{
		ConfigServiceURL: getenv("CONFIG_SERVICE_URL", "http://localhost:8090"),
		ServiceName:      getenv("SERVICE_NAME", "PENDING_CONFIGURATION"),
		Profile:          getenv("CONFIG_PROFILE", "local"),
		Timeout:          time.Duration(timeoutSec) * time.Second,
		FailFast:         strings.EqualFold(getenv("CONFIG_FAIL_FAST", "false"), "true"),
	}
}

func FetchRuntimeConfig(ctx context.Context, cfg Bootstrap) (*RuntimeConfig, error) {
	url := fmt.Sprintf("%s/api/v1/config/runtime/%s/%s", strings.TrimRight(cfg.ConfigServiceURL, "/"), cfg.ServiceName, cfg.Profile)
	client := &http.Client{Timeout: cfg.Timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		if cfg.FailFast {
			return nil, err
		}
		return fallbackFromEnv(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if cfg.FailFast {
			return nil, fmt.Errorf("config-service returned status %d", resp.StatusCode)
		}
		return fallbackFromEnv(), nil
	}

	var out RuntimeConfig
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		if cfg.FailFast {
			return nil, err
		}
		return fallbackFromEnv(), nil
	}
	return &out, nil
}

func fallbackFromEnv() *RuntimeConfig {
	return &RuntimeConfig{
		Service: getenv("SERVICE_NAME", "PENDING_CONFIGURATION"),
		Profile: getenv("CONFIG_PROFILE", "local"),
		Common: map[string]string{
			"ENVIRONMENT":              getenv("ENVIRONMENT", "local"),
			"KAFKA_BOOTSTRAP_SERVERS": getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		},
		ServiceConfig: struct {
			Name          string   `json:"name"`
			LocalPort     string   `json:"local_port"`
			BaseURLLocal  string   `json:"base_url_local"`
			RoutePrefix   string   `json:"route_prefix"`
			MainEndpoints []string `json:"main_endpoints"`
		}{
			Name:          getenv("SERVICE_NAME", "PENDING_CONFIGURATION"),
			LocalPort:     getenv("PORT", getenv("SERVER_PORT", "PENDING_CONFIGURATION")),
			BaseURLLocal:  getenv("BASE_URL_LOCAL", "PENDING_CONFIGURATION"),
			RoutePrefix:   getenv("ROUTE_PREFIX", "PENDING_CONFIGURATION"),
			MainEndpoints: []string{},
		},
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
