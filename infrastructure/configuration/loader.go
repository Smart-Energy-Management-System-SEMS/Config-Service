package configuration

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"config-service/domain/model"
)

const pending = "PENDING_CONFIGURATION"

var secretPattern = regexp.MustCompile(`(?i)(password|secret|token|apikey|api[_-]?key|connection\s*string|jwt|stripe|twilio|gmail|oauth|credential|db_url|database_url|mongodb_uri|kafka_username|kafka_password)`)

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
		Name:             pending,
		BoundedContext:   pending,
		LocalPort:        "",
		BaseURLLocal:     "",
		BaseURLDeploy:    pending,
		RoutePrefix:      pending,
		MainEndpoints:    []string{},
		Dependencies:     []string{},
		ExternalServices: []string{},
		SourceFile:       fileName,
	}

	scanner := bufio.NewScanner(file)
	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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

		clean := sanitize(strings.Trim(line, "` "))
		switch section {
		case "name":
			if svc.Name == pending && clean != "" {
				svc.Name = takeToken(clean)
			}
		case "context":
			if svc.BoundedContext == pending && clean != "" {
				svc.BoundedContext = clean
			}
		case "port":
			if svc.LocalPort == "" {
				svc.LocalPort = extractPort(clean)
			}
		case "base_local":
			if svc.BaseURLLocal == "" && strings.HasPrefix(clean, "http") {
				svc.BaseURLLocal = clean
			}
		case "base_deploy":
			if svc.BaseURLDeploy == pending && clean != "" {
				svc.BaseURLDeploy = clean
			}
		case "endpoints":
			if strings.HasPrefix(clean, "-") {
				ep := strings.TrimSpace(strings.TrimPrefix(clean, "-"))
				if ep != "" {
					svc.MainEndpoints = append(svc.MainEndpoints, ep)
				}
			}
		case "prefix":
			if svc.RoutePrefix == pending && strings.HasPrefix(clean, "/") {
				svc.RoutePrefix = clean
			}
		case "external":
			if strings.HasPrefix(clean, "-") {
				val := strings.TrimSpace(strings.TrimPrefix(clean, "-"))
				if val != "" {
					svc.ExternalServices = append(svc.ExternalServices, val)
				}
			}
		case "deps":
			if strings.HasPrefix(clean, "-") {
				val := strings.TrimSpace(strings.TrimPrefix(clean, "-"))
				if val != "" {
					svc.Dependencies = append(svc.Dependencies, val)
				}
			}
		}
	}

	if svc.Name == pending {
		svc.Name = normalizeName(fileName)
	}
	if svc.BaseURLDeploy == pending {
		svc.BaseURLDeploy = ""
	}
	if svc.RoutePrefix == pending {
		svc.RoutePrefix = ""
	}
	if len(svc.Dependencies) == 0 {
		svc.Dependencies = []string{pending}
	}
	if len(svc.ExternalServices) == 0 {
		svc.ExternalServices = []string{pending}
	}
	if len(svc.MainEndpoints) == 0 {
		svc.MainEndpoints = []string{pending}
	}
	return svc, scanner.Err()
}

func sanitize(value string) string {
	if secretPattern.MatchString(value) {
		return "***SECRET_NOT_EXPOSED***"
	}
	return value
}

func takeToken(v string) string {
	if strings.Contains(v, "(") {
		v = strings.Split(v, "(")[0]
	}
	return strings.TrimSpace(strings.Trim(v, "`"))
}

func normalizeName(fileName string) string {
	v := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	v = strings.ReplaceAll(v, "-C", "")
	v = strings.ReplaceAll(v, "_", "-")
	return strings.ToLower(strings.TrimSpace(v))
}

func extractPort(v string) string {
	re := regexp.MustCompile(`\d{2,5}`)
	if p := re.FindString(v); p != "" {
		return p
	}
	return ""
}
