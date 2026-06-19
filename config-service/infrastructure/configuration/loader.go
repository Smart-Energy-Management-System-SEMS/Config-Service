package configuration

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"config-service/config-service/domain/model"
)

var firstBacktickRegex = regexp.MustCompile("`([^`]*)`")

type Loader struct {
	sourcePath string
}

func NewLoader(sourcePath string) *Loader {
	if strings.TrimSpace(sourcePath) == "" {
		sourcePath = "Rutas"
	}
	return &Loader{sourcePath: sourcePath}
}

func (l *Loader) LoadServices() ([]model.ServiceConfig, error) {
	entries, err := os.ReadDir(l.sourcePath)
	if err != nil {
		return nil, err
	}

	services := make([]model.ServiceConfig, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			continue
		}

		service, err := l.parseServiceFile(filepath.Join(l.sourcePath, entry.Name()), entry.Name())
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

func (l *Loader) parseServiceFile(path, fileName string) (model.ServiceConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return model.ServiceConfig{}, err
	}

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	service := model.ServiceConfig{
		SourceFile:       fileName,
		Routes:           []string{},
		TopicsPublished:  []string{},
		TopicsConsumed:   []string{},
		HealthAliases:    []string{},
		MainEndpoints:    []string{},
		Dependencies:     []string{},
		ExternalServices: []string{},
	}

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "- Nombre del servicio:"):
			service.Name = firstBacktickValue(line)
		case strings.HasPrefix(line, "- URL local:"):
			service.URL = firstBacktickValue(line)
		case strings.HasPrefix(line, "- Puerto:"):
			service.Port = firstBacktickValue(line)
		case strings.HasPrefix(line, "- Prefix:"):
			service.Prefix = normalizePrefixValue(firstBacktickValue(line))
		case strings.HasPrefix(line, "- Health endpoint:"):
			service.Health = firstBacktickValue(line)
		case strings.HasPrefix(line, "- Topics que publica:"):
			service.TopicsPublished = splitBacktickList(line)
		case strings.HasPrefix(line, "- Topics que consume:"):
			service.TopicsConsumed = splitBacktickList(line)
		case strings.Contains(line, "| Servicio |"):
			if service.Name == "" {
				service.Name = firstBacktickValue(line)
			}
		case strings.Contains(line, "| Base URL local sugerida |"):
			if service.URL == "" {
				service.URL = firstBacktickValue(line)
			}
		case strings.Contains(line, "| Puerto |"):
			if service.Port == "" {
				service.Port = firstBacktickValue(line)
			}
		case strings.Contains(line, "| Prefix |"):
			if service.Prefix == "" {
				service.Prefix = normalizePrefixValue(firstBacktickValue(line))
			}
		case strings.Contains(line, "| Health |"):
			healths := splitBacktickList(line)
			if len(healths) > 0 {
				if service.Health == "" {
					service.Health = healths[0]
				}
				service.HealthAliases = uniqueStrings(append(service.HealthAliases, healths...))
			}
		case strings.Contains(line, "| Topics publica |"):
			if len(service.TopicsPublished) == 0 {
				service.TopicsPublished = splitBacktickList(line)
			}
		case strings.Contains(line, "| Topics consume |"):
			if len(service.TopicsConsumed) == 0 {
				service.TopicsConsumed = splitBacktickList(line)
			}
		case strings.Contains(line, "| Rutas principales |"):
			service.MainEndpoints = splitBacktickList(line)
		case strings.Contains(strings.ToLower(line), "path predicates") && strings.Contains(strings.ToLower(line), "recomendados"):
			service.Routes = splitBacktickList(line)
		case strings.Contains(strings.ToLower(line), "health path") && strings.Contains(strings.ToLower(line), "recomendado"):
			if health := firstBacktickValue(line); health != "" {
				service.Health = health
			}
		}
	}

	service.Prefix = normalizePrefixValue(service.Prefix)
	service.TopicsPublished = normalizeNotFoundList(service.TopicsPublished)
	service.TopicsConsumed = normalizeNotFoundList(service.TopicsConsumed)
	service.MainEndpoints = normalizeNotFoundList(service.MainEndpoints)
	service.HealthAliases = uniqueStrings(removeString(normalizeNotFoundList(service.HealthAliases), service.Health))

	service.Routes = normalizeRoutes(service)

	service.LocalPort = service.Port
	service.BaseURLLocal = service.URL
	service.RoutePrefix = service.Prefix

	return service, nil
}

func normalizeRoutes(service model.ServiceConfig) []string {
	if len(service.Routes) > 0 {
		routes := make([]string, 0, len(service.Routes))
		for _, route := range service.Routes {
			if route == service.Health || isHealthAlias(service.HealthAliases, route) || route == "" || route == "No encontrado" || !strings.HasPrefix(route, "/") {
				continue
			}
			routes = append(routes, route)
		}
		routes = replaceBroadWebhook(routes, service.MainEndpoints)
		return uniqueStrings(routes)
	}

	routes := make([]string, 0, len(service.MainEndpoints))
	for _, endpoint := range service.MainEndpoints {
		normalized := strings.TrimSpace(endpoint)
		if normalized == "" || normalized == "No encontrado" || normalized == service.Health {
			continue
		}
		if strings.HasPrefix(normalized, "/") {
			routes = append(routes, normalized)
			continue
		}
		if service.Prefix != "" && service.Prefix != "No encontrado" {
			routes = append(routes, joinRoute(service.Prefix, normalized))
		}
	}
	return uniqueStrings(routes)
}

func replaceBroadWebhook(routes, mainEndpoints []string) []string {
	hasExactWebhook := false
	exactWebhook := ""
	for _, endpoint := range mainEndpoints {
		if strings.Contains(endpoint, "/webhooks/stripe") {
			hasExactWebhook = true
			exactWebhook = endpoint
			break
		}
	}
	if !hasExactWebhook {
		return routes
	}

	out := make([]string, 0, len(routes))
	for _, route := range routes {
		if strings.HasSuffix(route, "/webhooks/**") {
			out = append(out, exactWebhook)
			continue
		}
		out = append(out, route)
	}
	return out
}

func joinRoute(prefix, route string) string {
	prefix = strings.TrimSuffix(strings.TrimSpace(prefix), "/")
	route = strings.TrimPrefix(strings.TrimSpace(route), "/")
	if prefix == "" {
		return "/" + route
	}
	return prefix + "/" + route
}

func firstBacktickValue(line string) string {
	matches := firstBacktickRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func splitBacktickList(line string) []string {
	matches := firstBacktickRegex.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value := strings.TrimSpace(match[1])
		if value == "" {
			continue
		}
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func removeString(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func normalizeNotFoundList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	if len(values) == 1 && strings.EqualFold(strings.TrimSpace(values[0]), "No encontrado") {
		return []string{}
	}
	return values
}

func normalizePrefixValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.Contains(strings.ToLower(value), "no hay prefix global") {
		return "/api/v1"
	}
	return value
}

func isHealthAlias(aliases []string, route string) bool {
	for _, alias := range aliases {
		if alias == route {
			return true
		}
	}
	return false
}
