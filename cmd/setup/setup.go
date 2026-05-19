package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/circleci/ex/config/o11y"
)

type Setup struct {
	O11yStatsd           string
	O11yHoneycombEnabled bool
	O11yHoneycombDataset string
	O11yService          string
	O11yFormat           string
	StatsNamespace       string
	O11yGrpcHostAndPort  string
}

func O11ySetup() *Setup {
	cfg := &Setup{
		O11yHoneycombEnabled: true,
		O11yGrpcHostAndPort:  "opentelementry-opentelemetry-collector.otel.svc.cluster.local:4317",
		O11yHoneycombDataset: "mickrok8s",
		O11yService:          "api-service",
		O11yFormat:           "json",
		StatsNamespace:       "api-service",
	}
	return cfg
}

func LoadO11y(ctx context.Context, mode string, cfg Setup, version string) (context.Context, func(context.Context), error) {
	o11ycfg := o11y.OtelConfig{
		RollbarDisabled: true,
		GrpcHostAndPort: cfg.O11yGrpcHostAndPort,
		Dataset:         cfg.O11yHoneycombDataset,
		Service:         cfg.O11yService,
		Mode:            mode,
		StatsNamespace:  cfg.StatsNamespace,
		Version:         version,
		UseEnvironments: true,
	}
	return o11y.Otel(ctx, addSampling(o11ycfg))
}

const (
	includeAll  = "includeAll"
	includeMany = "includeMany"
	includeSome = "includeSome"
	includeNone = "includeNone"
)

func addSampling(cfg o11y.OtelConfig) o11y.OtelConfig {
	cfg.SampleTraces = true
	cfg.SampleKeyFunc = defaultSampleFn

	cfg.SampleRates = map[string]uint{
		includeAll:  1,
		includeMany: 100,
		includeSome: 1000,
		includeNone: o11y.SampleOut,
	}
	return cfg
}

func defaultSampleFn(fields map[string]any) string {
	if _, ok := fields["error"]; ok {
		return includeAll
	}

	if intField(fields, "http.status_code") >= 500 {
		return includeAll
	}

	if _, ok := fields["warning"]; ok {
		return includeMany
	}

	if _, gotServer := fields["http.server_name"]; gotServer {
		return fmt.Sprintf("%s %v",
			fields["http.route"],
			fields["http.status_code"],
		)
	}

	name := stringField(fields, "name")
	if strings.HasPrefix(name, "db:") {
		return includeMany
	}

	return name
}

func stringField(fields map[string]any, key string) (val string) {
	if n, ok := fields[key]; ok {
		val, _ = n.(string)
	}
	return val
}

func intField(fields map[string]any, key string) (val int) {
	if n, ok := fields[key]; ok {
		val, _ = n.(int)
	}
	return val
}
