# Config Service (SEMS)

Microservicio de configuracion centralizada para SEMS. Expone configuracion publica para API Gateway y microservicios sin exponer secretos.

## Endpoints principales

- GET `/api/v1/health`
- GET `/api/v1/config/health`
- GET `/api/v1/config/services`
- GET `/api/v1/config/services/{serviceName}`
- GET `/api/v1/config/kafka`
- GET `/api/v1/config/api-gateway`
- GET `/api/v1/config/runtime/{serviceName}/{profile}`

## Puerto y runtime

El servicio lee el puerto en este orden:
1. `PORT` (recomendado para Azure Container Apps)
2. `CONFIG_SERVICE_PORT`
3. fallback `8090`

## Variables requeridas

Minimas para contenedor/local:
- `PORT` (ejemplo: `8080`)
- `CONFIG_SOURCE_PATH` (ejemplo: `Config`)
- `ENVIRONMENT` (ejemplo: `local`, `staging`, `prod`)

Variables de integracion/config centralizada:
- `CONFIG_SERVICE_URL`
- `KAFKA_BROKERS`
- `KAFKA_SECURITY_PROTOCOL`
- `KAFKA_SASL_MECHANISM`
- `KAFKA_USERNAME`
- `KAFKA_PASSWORD`
- `DATABASE_URL`
- `GIN_MODE`

Compatibilidad local Kafka:
- `KAFKA_BROKERS=localhost:9092`
- `KAFKA_BOOTSTRAP_SERVERS=localhost:9092`
- `KAFKA_BOOTSTRAP_SERVERS_LOCAL=localhost:9092`

## Ejecutar local (sin Docker)

```bash
go run main.go
```

Health check local:

```bash
curl http://localhost:8090/api/v1/health
curl http://localhost:8090/api/v1/config/health
```

## Docker

### Build

```bash
docker build -t sems-config-service:latest .
```

### Run (ejemplo local)

```bash
docker run --rm -p 8080:8080 \
  -e PORT=8080 \
  -e CONFIG_SOURCE_PATH=Config \
  -e ENVIRONMENT=local \
  -e KAFKA_BROKERS=host.docker.internal:9092 \
  -e KAFKA_BOOTSTRAP_SERVERS_LOCAL=host.docker.internal:9092 \
  -e KAFKA_SECURITY_PROTOCOL=PLAINTEXT \
  -e KAFKA_SASL_MECHANISM=NONE \
  sems-config-service:latest
```

## Azure Container Apps (ejemplo)

Definir en la Container App:
- `PORT=8080`
- `CONFIG_SOURCE_PATH=Config`
- `ENVIRONMENT=prod`
- `CONFIG_SERVICE_URL` (si aplica para clientes)
- `KAFKA_BROKERS=<broker-azure:9092>`
- `KAFKA_SECURITY_PROTOCOL=SASL_SSL` (segun tu cluster)
- `KAFKA_SASL_MECHANISM=SCRAM-SHA-256` (segun tu cluster)
- `KAFKA_USERNAME=<secret>`
- `KAFKA_PASSWORD=<secret>`
- `DATABASE_URL=<secret>`
- `GIN_MODE=release`

Configurar health probe en:
- `/api/v1/health` (recomendado)

## Seguridad

No exponer ni versionar secretos reales (`DATABASE_URL`, passwords, tokens, credenciales Kafka). Usar secretos de Azure Container Apps y/o Azure Key Vault.
