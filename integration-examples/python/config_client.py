import os
import requests


def env(name: str, default: str) -> str:
    value = os.getenv(name)
    return value if value else default


def str_to_bool(value: str) -> bool:
    return value.strip().lower() == "true"


def load_bootstrap() -> dict:
    return {
        "config_service_url": env("CONFIG_SERVICE_URL", "http://localhost:8090"),
        "service_name": env("SERVICE_NAME", "PENDING_CONFIGURATION"),
        "profile": env("CONFIG_PROFILE", "local"),
        "timeout": int(env("CONFIG_FETCH_TIMEOUT_SECONDS", "5")),
        "fail_fast": str_to_bool(env("CONFIG_FAIL_FAST", "false")),
    }


def fallback_from_env() -> dict:
    return {
        "service": env("SERVICE_NAME", "PENDING_CONFIGURATION"),
        "profile": env("CONFIG_PROFILE", "local"),
        "common": {
            "ENVIRONMENT": env("ENVIRONMENT", "local"),
            "KAFKA_BOOTSTRAP_SERVERS": env("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
        },
        "service_config": {
            "name": env("SERVICE_NAME", "PENDING_CONFIGURATION"),
            "local_port": env("PORT", env("SERVER_PORT", "PENDING_CONFIGURATION")),
            "base_url_local": env("BASE_URL_LOCAL", "PENDING_CONFIGURATION"),
            "route_prefix": env("ROUTE_PREFIX", "PENDING_CONFIGURATION"),
            "main_endpoints": [],
        },
    }


def fetch_runtime_config() -> dict:
    cfg = load_bootstrap()
    url = f"{cfg['config_service_url'].rstrip('/')}/api/v1/config/runtime/{cfg['service_name']}/{cfg['profile']}"

    try:
        response = requests.get(url, timeout=cfg["timeout"])
        response.raise_for_status()
        return response.json()
    except Exception:
        if cfg["fail_fast"]:
            raise
        return fallback_from_env()


if __name__ == "__main__":
    runtime_cfg = fetch_runtime_config()
    print(f"Loaded config for {runtime_cfg.get('service', 'unknown')}")
