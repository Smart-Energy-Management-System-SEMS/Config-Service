# Integration Examples

Plantillas minimas para que cada microservicio (Go, Python, Java) consuma:

- `GET /api/v1/config/runtime/{serviceName}/{profile}`

Todas incluyen:

- timeout HTTP
- fallback a variables de entorno
- validacion minima de respuesta

Variables esperadas:

- `CONFIG_SERVICE_URL` (ejemplo: `http://localhost:8090`)
- `SERVICE_NAME` (ejemplo: `iam-service`)
- `CONFIG_PROFILE` (ejemplo: `local`)
- `CONFIG_FETCH_TIMEOUT_SECONDS` (ejemplo: `5`)
- `CONFIG_FAIL_FAST` (`true`/`false`)
