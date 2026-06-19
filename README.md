# Config Service (SEMS)

Servicio central de configuracion para la arquitectura SEMS. Expone URLs de microservicios, configuracion Kafka para local y Azure, topics agrupados y el mapeo publish/consume por microservicio.

## Endpoints

- GET `/health`
- GET `/api/v1/health`
- GET `/api/v1/config/health`
- GET `/api/v1/config/services`
- GET `/api/v1/config/services/{serviceName}`
- GET `/api/v1/config/kafka`
- GET `/api/v1/config/topics`
- GET `/api/v1/config/api-gateway`
- GET `/api/v1/config/runtime/{serviceName}/{profile}`

## Microservicios soportados

- IAM Service: `http://localhost:8082`
- Device Management Service: `http://localhost:8083`
- Energy Monitoring Service: `http://localhost:8001`
- Analytics Service: `http://localhost:8004`
- Alerts Service: `http://localhost:8085`
- Payments Service: `http://localhost:8086`
- Subscriptions Service: `http://localhost:18083`

## Topics agrupados oficiales

- `iam.events`
- `device.events`
- `energy.events`
- `analytics.events`
- `alerts.events`
- `payments.events`
- `subscriptions.events`
- `billing.events`

Los eventos especificos ya no deben usarse como topic principal. Deben viajar dentro del payload usando `eventType`, por ejemplo:

```json
{
  "eventType": "energy.consumption.recorded",
  "eventId": "uuid",
  "occurredAt": "ISO_DATE",
  "data": {}
}
```

## Variables locales recomendadas

```env
PORT=8090
CONFIG_SOURCE_PATH=Rutas
ENVIRONMENT=local
IAM_SERVICE_URL=http://localhost:8082
DEVICE_SERVICE_URL=http://localhost:8083
ENERGY_SERVICE_URL=http://localhost:8001
ANALYTICS_SERVICE_URL=http://localhost:8004
ALERTS_SERVICE_URL=http://localhost:8085
PAYMENTS_SERVICE_URL=http://localhost:8086
SUBSCRIPTIONS_SERVICE_URL=http://localhost:18083
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
KAFKA_BROKERS=localhost:9092
KAFKA_SECURITY_PROTOCOL=PLAINTEXT
KAFKA_SASL_MECHANISM=
```

## Variables Azure recomendadas

```env
PORT=8080
CONFIG_SOURCE_PATH=Rutas
ENVIRONMENT=azure
IAM_SERVICE_URL=https://iam-service2.<azure-domain>
DEVICE_SERVICE_URL=https://device-service.<azure-domain>
ENERGY_SERVICE_URL=https://energy-service.<azure-domain>
ANALYTICS_SERVICE_URL=https://analytics-service.<azure-domain>
ALERTS_SERVICE_URL=https://alerts-service.<azure-domain>
PAYMENTS_SERVICE_URL=https://payments-service.<azure-domain>
SUBSCRIPTIONS_SERVICE_URL=https://subscriptions-service.<azure-domain>
KAFKA_BOOTSTRAP_SERVERS=sems-kafka-ns.servicebus.windows.net:9093
KAFKA_BROKERS=sems-kafka-ns.servicebus.windows.net:9093
KAFKA_SECURITY_PROTOCOL=SASL_SSL
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SASL_USERNAME=$ConnectionString
KAFKA_SASL_PASSWORD=<AZURE_EVENT_HUB_CONNECTION_STRING>
```

## Perfiles

- `local`: usa `KAFKA_BOOTSTRAP_SERVERS_LOCAL` si existe; si no, `KAFKA_BOOTSTRAP_SERVERS` o `KAFKA_BROKERS`
- `docker`: usa `KAFKA_BOOTSTRAP_SERVERS_DOCKER` si existe
- `azure`: usa `KAFKA_BOOTSTRAP_SERVERS_AZURE` o `KAFKA_BROKERS_AZURE`; si no existen, reutiliza `KAFKA_BOOTSTRAP_SERVERS`

## Ejecucion local

```bash
go run main.go
```

Health check:

```bash
curl http://localhost:8090/health
curl http://localhost:8090/api/v1/config/kafka
curl http://localhost:8090/api/v1/config/topics
```

## Notas de compatibilidad

- Se mantienen los endpoints existentes.
- Las URLs de servicios pueden resolverse por variables de entorno y ya no dependen rigidamente de `localhost`.
- El endpoint `/api/v1/config/kafka` devuelve `enabled`, `bootstrapServers`, `brokers`, `securityProtocol`, `saslMechanism`, `username`, `topics`, `publishTopics` y `consumeTopics`.
- Los secretos SASL se enmascaran en las respuestas HTTP.
