package configuration

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"config-service/config-service/domain/model"
)

const pending = "PENDING_CONFIGURATION"

var (
	portRegex        = regexp.MustCompile(`\d{2,5}`)
	serviceKeyRegex  = regexp.MustCompile(`[^a-z0-9-]+`)
	endpointFmtRegex = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE)\s+/`)
)

type Loader struct {
	sourcePath string
}

func NewLoader(sourcePath string) *Loader {
	if strings.TrimSpace(sourcePath) == "" {
		sourcePath = "Config"
	}
	return &Loader{sourcePath: sourcePath}
}

func (l *Loader) LoadServices() ([]model.ServiceConfig, error) {
	entries, err := os.ReadDir(l.sourcePath)
	if err != nil {
		return nil, err
	}

	services := make([]model.ServiceConfig, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			continue
		}
		path := filepath.Join(l.sourcePath, e.Name())
		svc, err := parseServiceFile(path, e.Name())
		if err != nil {
			continue
		}
		services = append(services, svc)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	return services, nil
}

func parseServiceFile(path, fileName string) (model.ServiceConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.ServiceConfig{}, err
	}
	defer file.Close()

	svc := model.ServiceConfig{
		Name:             normalizeServiceName(""),
		BoundedContext:   pending,
		LocalPort:        "",
		BaseURLLocal:     "",
		BaseURLDeploy:    "",
		RoutePrefix:      "",
		MainEndpoints:    []string{},
		Dependencies:     []string{},
		ExternalServices: []string{},
		SourceFile:       fileName,
	}

	scanner := bufio.NewScanner(file)
	section := ""
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := normalizeText(rawLine)
		if line == "" {
			continue
		}
		lc := strings.ToLower(line)

		switch {
		case strings.Contains(lc, "nombre del microservicio"):
			section = "name"
			continue
		case strings.Contains(lc, "bounded context"):
			section = "context"
			continue
		case strings.Contains(lc, "puerto local"):
			section = "port"
			continue
		case strings.Contains(lc, "base url local"):
			section = "base_local"
			continue
		case strings.Contains(lc, "base url esperada para deploy"):
			section = "base_deploy"
			continue
		case strings.Contains(lc, "endpoints principales"):
			section = "endpoints"
			continue
		case strings.Contains(lc, "prefijo base de rutas"):
			section = "prefix"
			continue
		case strings.Contains(lc, "servicios externos"):
			section = "external"
			continue
		case strings.Contains(lc, "dependencias de otros microservicios"):
			section = "deps"
			continue
		}

		switch section {
		case "name":
			if svc.Name == pending {
				svc.Name = normalizeServiceName(takeToken(line))
			}
		case "context":
			if svc.BoundedContext == pending {
				svc.BoundedContext = line
			}
		case "port":
			if svc.LocalPort == "" {
				svc.LocalPort = extractPort(line)
			}
		case "base_local":
			if svc.BaseURLLocal == "" {
				svc.BaseURLLocal = normalizeLocalURL(line, svc.LocalPort)
			}
		case "base_deploy":
			if svc.BaseURLDeploy == "" && strings.Contains(strings.ToLower(line), "http") {
				svc.BaseURLDeploy = line
			}
		case "endpoints":
			ep := parseListItem(rawLine)
			if strings.HasPrefix(strings.TrimSpace(rawLine), "-") && endpointFmtRegex.MatchString(ep) {
				svc.MainEndpoints = append(svc.MainEndpoints, ep)
			}
		case "prefix":
			if svc.RoutePrefix == "" && strings.HasPrefix(line, "/") {
				svc.RoutePrefix = strings.Fields(line)[0]
			}
		case "external":
			val := parseListItem(rawLine)
			if strings.HasPrefix(strings.TrimSpace(rawLine), "-") && val != "" {
				svc.ExternalServices = append(svc.ExternalServices, val)
			}
		case "deps":
			val := parseListItem(rawLine)
			if strings.HasPrefix(strings.TrimSpace(rawLine), "-") && val != "" {
				svc.Dependencies = append(svc.Dependencies, val)
			}
		}
	}

	if svc.Name == pending {
		svc.Name = normalizeNameFromFile(fileName)
	}
	if svc.LocalPort == "" {
		svc.LocalPort = defaultPortForService(svc.Name)
	}
	if svc.BaseURLLocal == "" {
		svc.BaseURLLocal = "http://localhost:" + svc.LocalPort
	}
	if svc.RoutePrefix == "" {
		svc.RoutePrefix = defaultPrefixForService(svc.Name)
	}
	if len(svc.MainEndpoints) == 0 {
		svc.MainEndpoints = []string{pending}
	}
	if len(svc.Dependencies) == 0 {
		svc.Dependencies = []string{pending}
	}
	if len(svc.ExternalServices) == 0 {
		svc.ExternalServices = []string{pending}
	}

	ensureUniqueLocalPort(&svc)
	return svc, scanner.Err()
}

func normalizeText(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "`", "")
	return strings.TrimSpace(v)
}

func parseListItem(v string) string {
	v = normalizeText(v)
	v = strings.TrimPrefix(v, "- ")
	v = strings.TrimPrefix(v, "-")
	return strings.TrimSpace(v)
}

func takeToken(v string) string {
	if i := strings.Index(v, "("); i > 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

func extractPort(v string) string {
	if p := portRegex.FindString(v); p != "" {
		return p
	}
	return ""
}

func normalizeLocalURL(raw, knownPort string) string {
	candidate := strings.Fields(raw)
	for _, t := range candidate {
		if strings.HasPrefix(strings.ToLower(t), "http://") {
			u, err := url.Parse(strings.TrimSpace(t))
			if err == nil && u.Host != "" {
				if u.Scheme == "http" && strings.Contains(u.Host, "localhost") {
					if p := u.Port(); p != "" {
						return "http://localhost:" + p
					}
				}
			}
		}
	}
	if knownPort != "" {
		return "http://localhost:" + knownPort
	}
	return ""
}

func normalizeServiceName(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "_", "-")
	v = strings.ReplaceAll(v, " ", "-")
	v = strings.ReplaceAll(v, "microservice-", "")
	v = serviceKeyRegex.ReplaceAllString(v, "")
	if v == "" {
		return pending
	}
	if !strings.HasSuffix(v, "-service") {
		if strings.Contains(v, "iam") {
			return "iam-service"
		}
		if strings.Contains(v, "analytics") {
			return "analytics-service"
		}
		if strings.Contains(v, "energy") {
			return "energy-monitoring-service"
		}
		if strings.Contains(v, "device") {
			return "device-management-service"
		}
		if strings.Contains(v, "payment") {
			return "payments-service"
		}
		if strings.Contains(v, "sub") {
			return "subscriptions-service"
		}
		if strings.Contains(v, "alert") {
			return "alert-service"
		}
	}
	return v
}

func normalizeNameFromFile(fileName string) string {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	base = strings.ReplaceAll(base, "-C", "")
	return normalizeServiceName(base)
}

func defaultPortForService(service string) string {
	switch service {
	case "analytics-service":
		return "8004"
	case "energy-monitoring-service":
		return "8001"
	case "alert-service":
		return "8085"
	case "device-management-service":
		return "8083"
	case "iam-service":
		return "8080"
	case "subscriptions-service":
		return "8082"
	case "payments-service":
		return "8086"
	default:
		return ""
	}
}

func defaultPrefixForService(service string) string {
	switch service {
	case "analytics-service":
		return "/api/v1/analytics"
	case "device-management-service":
		return "/api/v1/device-management"
	default:
		return "/api/v1"
	}
}

func ensureUniqueLocalPort(svc *model.ServiceConfig) {
	// Normaliza puertos locales para evitar conflicto conocido alert/payments en 8085.
	if svc.Name == "payments-service" && svc.LocalPort == "8085" {
		svc.LocalPort = "8086"
		svc.BaseURLLocal = "http://localhost:8086"
	}
}
