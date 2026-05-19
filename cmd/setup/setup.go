package setup

import (
	"context"
	"os"

	"github.com/circleci/ex/config/o11y"
	"github.com/circleci/ex/config/secret"
)

type Setup struct {
	O11yStatsd           string
	O11yHoneycombEnabled bool
	O11yHoneycombHost    string
	O11yHoneycombKey     secret.String
	O11yHoneycombDataset string
	O11yService          string
	O11yFormat           string
	StatsNamespace       string
}

func O11ySetup() *Setup {
	hcKey := os.Getenv("HC_TOKEN")
	cfg := &Setup{
		O11yHoneycombKey:     secret.String(hcKey),
		O11yHoneycombEnabled: true,
		O11yHoneycombHost:    "https://api.honeycomb.io",
		O11yHoneycombDataset: "mickrok8s",
		O11yService:          "api-service",
		O11yFormat:           "json",
		StatsNamespace:       "api-service",
	}
	return cfg
}

func LoadO11y(ctx context.Context, mode string, cfg Setup, version string) (context.Context, func(context.Context), error) {
	o11ycfg := o11y.OtelConfig{
		RollbarDisabled:   true,
		HTTPTracesURL:     cfg.O11yHoneycombHost,
		HTTPAuthorization: cfg.O11yHoneycombKey,
		Dataset:           cfg.O11yHoneycombDataset,
		Service:           cfg.O11yService,
		Mode:              mode,
		StatsNamespace:    cfg.StatsNamespace,
		Version:           version,
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
	cfg.SampleRates = map[string]uint{
		includeAll:  1,
		includeMany: 100,
		includeSome: 1000,
		includeNone: o11y.SampleOut,
	}
	return cfg
}
