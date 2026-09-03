//go:build wireinject
// +build wireinject

package main

import (
	"database/sql"
	"net/http"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	skillimportapp "github.com/Wei-Shaw/sub2api/internal/modules/skillimport/application"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

type Application struct {
	Server         *http.Server
	RequestDrainer *server.RequestDrainer
	Supervisor     *lifecycle.Supervisor
	Cleanup        func()
}

func initializeApplication(
	buildInfo handler.BuildInfo,
	cfg *config.Config,
	entClient *ent.Client,
	db *sql.DB,
	redisClient *redis.Client,
) (*Application, error) {
	wire.Build(
		// Business layer ProviderSets
		repository.ProviderSet,
		service.ProviderSet,
		skillimportapp.ProviderSet,
		payment.ProviderSet,
		middleware.ProviderSet,
		handler.ProviderSet,

		// Server layer ProviderSet
		server.ProviderSet,

		// Privacy client factory for OpenAI training opt-out
		providePrivacyClientFactory,

		// BuildInfo provider
		provideServiceBuildInfo,

		// Cleanup function provider
		provideCleanup,
		buildApplicationSupervisor,

		// Application struct
		wire.Struct(new(Application), "Server", "RequestDrainer", "Supervisor", "Cleanup"),
	)
	return nil, nil
}
