package setup

import (
	"context"
	"github.com/circleci/ex/config/o11y"
	"github.com/circleci/ex/config/secret"
	"os"
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

func LoadO11y(ctx context.Context, mode string, cfg Setup) (context.Context, func(context.Context), error) {
	o11ycfg := o11y.Config{
		RollbarDisabled:  true,
		HoneycombEnabled: cfg.O11yHoneycombEnabled,
		HoneycombHost:    cfg.O11yHoneycombHost,
		HoneycombKey:     cfg.O11yHoneycombKey,
		HoneycombDataset: cfg.O11yHoneycombDataset,
		Service:          cfg.O11yService,
		Format:           cfg.O11yFormat,
		Mode:             mode,
		StatsNamespace:   cfg.StatsNamespace,
	}
	return o11y.Setup(ctx, addSampling(o11ycfg))
}

func addSampling(cfg o11y.Config) o11y.Config {
	cfg.SampleTraces = true
	cfg.SampleRates = map[string]int{
		"/api/private/list_tables 200":            1000,
		"/api/private/create_table 200":           1000,
		"/api/private/delete_table 200":           1000,
		"/api/private/update_table_with_user 200": 1000,
		"/api/private/get_current_score 200":      1000,
		"/api/private/update_user_score 200":      1000,
		"/api/private/get_pokemon 200":            1000,
		"/api/private/put_answer 200":             1000,
		"/api/private/get_answer 200":             1000,
		"/api/private/leaderboard 200":            1000,
	}
	return cfg
}
