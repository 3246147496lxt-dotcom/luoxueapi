package application

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
	"github.com/google/wire"
)

func ProvideHTTPFetcher(cfg *config.Config) (core.HTTPFetcher, error) {
	settings := config.SkillImportConfig{}
	if cfg != nil {
		settings = cfg.SkillImport
	}
	globalConcurrency := settings.WorkerConcurrency * 4
	if globalConcurrency < 4 {
		globalConcurrency = 4
	}
	if globalConcurrency > 32 {
		globalConcurrency = 32
	}
	perHost := settings.PerHostConcurrency
	if perHost <= 0 {
		perHost = 1
	}
	timeout := time.Duration(settings.HTTPTimeoutSeconds) * time.Second
	return core.NewSecureFetcher(core.SecureFetcherOptions{
		Timeout: timeout, MaxResponseBytes: settings.MaxHTTPResponseBytes,
		MaxConcurrent: globalConcurrency, MaxConnsPerHost: perHost,
		MinIntervalPerHost: 100 * time.Millisecond,
	})
}

func ProvideAdapterRegistry(fetcher core.HTTPFetcher, cfg *config.Config) *AdapterRegistry {
	token := ""
	if cfg != nil {
		token = cfg.SkillImport.GitHubToken
	}
	return NewAdapterRegistry(core.NewBuiltInAdapters(fetcher, token)...)
}

func ProvideNormalizer() *core.Normalizer { return core.NewNormalizer() }

var ProviderSet = wire.NewSet(
	ProvideHTTPFetcher,
	ProvideAdapterRegistry,
	ProvideNormalizer,
	NewService,
	NewWorkerRuntime,
)
