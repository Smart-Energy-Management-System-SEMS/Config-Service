package pe.edu.upc.configclient;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

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

    private static String env(String key, String fallback) {
        String value = System.getenv(key);
        return (value == null || value.isBlank()) ? fallback : value;
    }

    public static void main(String[] args) throws Exception {
        Bootstrap cfg = loadBootstrap();
        String json = fetchRuntimeConfigJson(cfg);
        System.out.println("Loaded runtime config JSON: " + json);
    }
}
