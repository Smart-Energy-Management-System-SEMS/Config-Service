package main

import (
	"log"
	"net/http"

	"config-service/config-service/application/services"
	"config-service/config-service/infrastructure/configuration"
	"config-service/config-service/interfaces/rest"
	"config-service/shared"
)

func main() {
	port := shared.EnvOrDefault("PORT", shared.EnvOrDefault("CONFIG_SERVICE_PORT", "8090"))
	sourcePath := shared.EnvOrDefault("CONFIG_SOURCE_PATH", "Rutas")

	loader := configuration.NewLoader(sourcePath)
	configService, err := services.NewConfigService(loader)
	if err != nil {
		log.Fatalf("failed to load config source: %v", err)
	}

	mux := http.NewServeMux()
	handler := rest.NewHandler(configService)
	handler.Register(mux)

	addr := ":" + port
	log.Printf("config-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
