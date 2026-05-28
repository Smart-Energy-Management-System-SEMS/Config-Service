# Config Service (SEMS)

Microservicio tecnico de configuracion centralizada para SEMS. Expone configuracion publica para API Gateway y microservicios, sin exponer secretos.

## Arquitectura

Arquitectura modular ligera estilo Clean Architecture:

- `application/services`: casos de uso del servicio de configuracion.
- `domain/model`: modelos de dominio de configuracion.
- `infrastructure/configuration`: carga y parseo de archivos en `Config/`.
- `infrastructure/http`: utilidades HTTP de respuesta.
- `interfaces/rest`: handlers y rutas REST.
- `shared`: utilidades compartidas.

## Fuente de configuracion

El servicio usa `Config/*.txt` como fuente inicial:

- `Alert-C.txt`
- `Analytics-C.txt`
- `Device-C.txt`
- `Energy-C.txt`
- `IAM-C.txt`
- `Payments-C.txt`
- `Subs-C.txt`

Si un dato no existe, se usa `""` o `"PENDING_CONFIGURATION"`.

## Endpoints

- `GET /api/v1/config/health`
- `GET /api/v1/config/services`
- `GET /api/v1/config/services/{serviceName}`
- `GET /api/v1/config/kafka`
- `GET /api/v1/config/api-gateway`

## Seguridad

Este servicio NO expone secretos. Cualquier valor sensible detectado se reemplaza por:

- `***SECRET_NOT_EXPOSED***`

No guardar claves reales en codigo ni en `.env.example`.

## Variables de entorno

Usar `.env.example` como plantilla:

- `CONFIG_SERVICE_PORT`
- `CONFIG_SOURCE_PATH`
- `ENVIRONMENT`
- `API_GATEWAY_*`
- `KAFKA_*`

## Ejecucion local

1. Configurar variables de entorno (o un `.env` propio para desarrollo).
2. Ejecutar:

```bash
go run main.go
```

3. Probar health:

```bash
curl http://localhost:8090/api/v1/config/health
```

## Uso con API Gateway

- Consumir `GET /api/v1/config/services` para discovery de rutas/servicios.
- Consumir `GET /api/v1/config/api-gateway` para placeholders de configuracion del gateway.
- Consumir `GET /api/v1/config/kafka` para metadatos centralizados de mensajeria.
- Para bootstrap de microservicios por lenguaje (Go/Python/Java), usar:
  - `GET /api/v1/config/runtime/{serviceName}/{profile}`
  - Ejemplos listos en `integration-examples/`.



```bash
  -H "Content-Type: application/json" \
  -d "{\"topic\":\"analytics.anomaly.detected\",\"key\":\"test-key\",\"payload\":{\"message\":\"hello kafka\"}}"
```

- `KAFKA_BOOTSTRAP_SERVERS` correcto (ej. `localhost:9092`)

## Deploy en Azure Container Apps

1. Construir imagen Docker del servicio.
2. Publicar imagen en Azure Container Registry (ACR).
3. Crear Container App con variables:
   - `CONFIG_SERVICE_PORT`
   - `CONFIG_SOURCE_PATH`
   - `ENVIRONMENT`
   - `API_GATEWAY_*`
   - `KAFKA_*`
4. Configurar probes apuntando a:
   - `/api/v1/config/health`
5. Gestionar secretos reales con Azure Key Vault o secretos de Container Apps, no en codigo.

## Advertencia

No exponer `DATABASE_URL`, tokens, passwords, JWT secrets, Stripe/Twilio/Gmail/OAuth keys, ni credenciales Kafka desde este servicio.
