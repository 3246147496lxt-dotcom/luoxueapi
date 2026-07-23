package main

//go:generate go run github.com/google/wire/cmd/wire@v0.7.0

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/setup"
	"github.com/Wei-Shaw/sub2api/internal/web"

	"github.com/gin-gonic/gin"
)

//go:embed VERSION
var embeddedVersion string

// Build-time variables (can be set by ldflags)
var (
	Version   = ""
	Commit    = "unknown"
	Date      = "unknown"
	BuildType = "source" // "source" for manual builds, "release" for CI builds (set by ldflags)
)

func init() {
	// 如果 Version 已通过 ldflags 注入（例如 -X main.Version=...），则不要覆盖。
	if strings.TrimSpace(Version) != "" {
		return
	}

	// 默认从 embedded VERSION 文件读取版本号（编译期打包进二进制）。
	Version = strings.TrimSpace(embeddedVersion)
	if Version == "" {
		Version = "0.0.0-dev"
	}
}

func main() {
	if err := run(); err != nil {
		log.Printf("LuoxueAPI exited with error: %v", err)
		os.Exit(1)
	}
}

// run owns process-scoped resources so every error path executes deferred
// logger and application cleanup before main chooses the exit status.
func run() error {
	logger.InitBootstrap()
	defer logger.Sync()

	// Parse command line flags
	setupMode := flag.Bool("setup", false, "Run setup wizard in CLI mode")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		log.Printf("LuoxueAPI %s (commit: %s, built: %s)\n", Version, Commit, Date)
		return nil
	}

	// CLI setup mode
	if *setupMode {
		if err := setup.RunCLI(); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		return nil
	}

	// Check if setup is needed
	if setup.NeedsSetup() {
		// Check if auto-setup is enabled (for Docker deployment)
		if setup.AutoSetupEnabled() {
			log.Println("Auto setup mode enabled...")
			if err := setup.AutoSetupFromEnv(); err != nil {
				return fmt.Errorf("auto setup failed: %w", err)
			}
			// Continue to main server after auto-setup
		} else {
			log.Println("First run detected, starting setup wizard...")
			return runSetupServer()
		}
	}

	// Normal server mode
	return runMainServer()
}

func runSetupServer() error {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(config.CORSConfig{}))
	r.Use(middleware.SecurityHeaders(config.CSPConfig{Enabled: true, Policy: config.DefaultCSPPolicy}, nil))

	// Register setup routes
	setup.RegisterRoutes(r)

	// Serve embedded frontend if available
	if web.HasEmbeddedFrontend() {
		r.Use(web.ServeEmbeddedFrontend())
	}

	// Get server address from config.yaml or environment variables (SERVER_HOST, SERVER_PORT)
	// This allows users to run setup on a different address if needed
	addr := config.GetServerAddress()
	log.Printf("Setup wizard available at http://%s", addr)
	log.Println("Complete the setup wizard to configure LuoxueAPI")

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       120 * time.Second,
		Protocols:         protocols,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("start setup server: %w", err)
		}
		return nil
	case <-signalCtx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown setup server: %w", err)
	}
	return nil
}

func runMainServer() error {
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := logger.Init(logger.OptionsFromConfig(cfg.Log)); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	// Gin mode is process-global state, so configure it explicitly in the
	// runtime entrypoint instead of mutating globals from a Wire provider.
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	if cfg.RunMode == config.RunModeSimple {
		log.Println("⚠️  WARNING: Running in SIMPLE mode - billing and quota checks are DISABLED")
	}

	buildInfo := handler.BuildInfo{
		Version:   Version,
		BuildType: BuildType,
	}

	entClient, sqlDB, err := repository.InitEnt(cfg)
	if err != nil {
		return fmt.Errorf("bootstrap database: %w", err)
	}
	redisClient := repository.InitRedis(cfg)
	app, err := initializeApplication(buildInfo, cfg, entClient, sqlDB, redisClient)
	if err != nil {
		if redisClient != nil {
			_ = redisClient.Close()
		}
		_ = entClient.Close()
		return fmt.Errorf("initialize application: %w", err)
	}
	defer cleanupApplicationInfrastructure(app)
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	if app.Supervisor != nil {
		if err := app.Supervisor.Start(signalCtx); err != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			stopErr := app.Supervisor.Stop(stopCtx)
			cancel()
			startErr := fmt.Errorf("start application components: %w", err)
			if stopErr != nil {
				return errors.Join(startErr, fmt.Errorf("retry application component rollback: %w", stopErr))
			}
			return startErr
		}
		defer func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := app.Supervisor.Stop(stopCtx); err != nil {
				log.Printf("Application component shutdown failed: %v", err)
			}
		}()
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- app.Server.ListenAndServe()
	}()

	log.Printf("Server started on %s", app.Server.Addr)

	// 等待中断信号
	var serveErr error
	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr = fmt.Errorf("serve HTTP: %w", err)
		}
	case <-signalCtx.Done():
	}

	log.Println("Shutting down server...")
	drainStartedAt := time.Now()
	if app.Supervisor != nil {
		// Readiness flips before HTTP Shutdown starts draining active ingress.
		app.Supervisor.BeginDrain()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := app.Server.Shutdown(ctx)
	if shutdownErr != nil {
		shutdownErr = fmt.Errorf("shutdown HTTP server: %w", shutdownErr)
	}

	var componentErr error
	if app.Supervisor != nil {
		componentCtx, componentCancel := context.WithTimeout(context.Background(), 10*time.Second)
		componentErr = app.Supervisor.Stop(componentCtx)
		componentCancel()
		if componentErr != nil {
			componentErr = fmt.Errorf("shutdown application components: %w", componentErr)
		}
	}

	log.Printf("Server exited after shutdown drain duration=%s", time.Since(drainStartedAt))
	return errors.Join(serveErr, shutdownErr, componentErr)
}

func cleanupApplicationInfrastructure(app *Application) {
	if app == nil || app.Cleanup == nil {
		return
	}
	if app.Supervisor != nil && app.Supervisor.HasPendingComponents() {
		// A timed-out component can still be using Redis or PostgreSQL. Do not
		// close them underneath that worker; process exit will release them after
		// the deferred retry has had its final opportunity to join.
		log.Printf("Skipping explicit infrastructure cleanup: application components remain pending")
		return
	}
	app.Cleanup()
}
