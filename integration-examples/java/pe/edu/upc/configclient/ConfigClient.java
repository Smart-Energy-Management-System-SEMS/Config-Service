package pe.edu.upc.configclient;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
/**
 * Cliente para obtener configuración en tiempo de ejecución desde el servicio de configuración.
 * 
 * Este cliente se conecta a un servicio de configuración remoto para recuperar parámetros
 * de configuración específicos del servicio y perfil. Soporta fallback a configuración
 * por defecto en caso de fallo.
 * 
 * Variables de entorno requeridas:
 * - CONFIG_SERVICE_URL: URL del servicio de configuración (default: http://localhost:8090)
 * - SERVICE_NAME: Nombre del servicio (default: PENDING_CONFIGURATION)
 * - CONFIG_PROFILE: Perfil de configuración (default: local)
 * - CONFIG_FETCH_TIMEOUT_SECONDS: Tiempo de espera en segundos (default: 5)
 * - CONFIG_FAIL_FAST: Si es true, lanza excepción en caso de error (default: false)
 */
public class ConfigClient {

    public record Bootstrap(String configServiceUrl, String serviceName, String profile, int timeoutSeconds, boolean failFast) {}

    public static Bootstrap loadBootstrap() {
        String configServiceUrl = env("CONFIG_SERVICE_URL", "http://localhost:8090");
        String serviceName = env("SERVICE_NAME", "PENDING_CONFIGURATION");
        String profile = env("CONFIG_PROFILE", "local");
        int timeout = Integer.parseInt(env("CONFIG_FETCH_TIMEOUT_SECONDS", "5"));
        boolean failFast = Boolean.parseBoolean(env("CONFIG_FAIL_FAST", "false"));
        return new Bootstrap(configServiceUrl, serviceName, profile, timeout, failFast);
    }

    public static String fetchRuntimeConfigJson(Bootstrap cfg) throws IOException, InterruptedException {
        String url = cfg.configServiceUrl().replaceAll("/$", "") +
                "/api/v1/config/runtime/" + cfg.serviceName() + "/" + cfg.profile();

        HttpClient client = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(cfg.timeoutSeconds()))
                .build();

        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .timeout(Duration.ofSeconds(cfg.timeoutSeconds()))
                .GET()
                .build();

        try {
            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() == 200) {
                return response.body();
            }
            if (cfg.failFast()) {
                throw new IOException("config-service status: " + response.statusCode());
            }
            return fallbackJson();
        } catch (Exception ex) {
            if (cfg.failFast()) {
                throw ex;
            }
            return fallbackJson();
        }
    }

    private static String fallbackJson() {
        return "{" +
                "\"service\":\"" + env("SERVICE_NAME", "PENDING_CONFIGURATION") + "\"," +
                "\"profile\":\"" + env("CONFIG_PROFILE", "local") + "\"," +
                "\"common\":{" +
                "\"ENVIRONMENT\":\"" + env("ENVIRONMENT", "local") + "\"," +
                "\"KAFKA_BOOTSTRAP_SERVERS\":\"" + env("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092") + "\"}" +
                "}";
    }
     /**
     * Obtiene el valor de una variable de entorno con fallback a valor por defecto.
     * 
     * Busca la variable de entorno especificada. Si no existe o está vacía,
     * retorna el valor por defecto proporcionado.
     * 
     * @param key Nombre de la variable de entorno a buscar
     * @param fallback Valor por defecto si la variable no existe o está vacía
     * @return Valor de la variable de entorno o fallback si no está disponible
     */
    private static String env(String key, String fallback) {
        String value = System.getenv(key);
        return (value == null || value.isBlank()) ? fallback : value;
    }
    /**
     * Método principal para probar el cliente de configuración.
     * 
     * Realiza el flujo completo:
     * 1. Carga configuración de bootstrap desde variables de entorno
     * 2. Obtiene configuración en tiempo de ejecución del servicio remoto
     * 3. Imprime la configuración obtenida en consola
     * 
     * @param args Argumentos de línea de comandos (no utilizados)
     * @throws Exception Si ocurre cualquier error durante la ejecución
     */
    public static void main(String[] args) throws Exception {
        Bootstrap cfg = loadBootstrap();
        String json = fetchRuntimeConfigJson(cfg);
        System.out.println("Loaded runtime config JSON: " + json);
    }
}
