package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"aidevclub/internal/app"
	"aidevclub/internal/mcpserver"
	"aidevclub/internal/platform"
)

func main() {
	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	infra, err := app.OpenInfrastructure(cfg)
	if err != nil {
		logger.Error("open infrastructure", "err", err)
		return
	}
	defer func() {
		if err := infra.Close(); err != nil {
			logger.Error("close infrastructure", "err", err)
		}
	}()

	if err := app.Migrate(infra.DB); err != nil {
		logger.Error("migrate", "err", err)
		return
	}

	services := app.NewServices(infra, cfg)

	deps := mcpserver.Dependencies{
		Public: mcpserver.PublicDependencies{
			Search:     services.Search,
			Articles:   services.Articles,
			Skills:     services.Skills,
			MCPServers: services.MCPServers,
			Ranking:    services.Ranking,
			Tags:       services.Tags,
		},
		Account: mcpserver.AccountDependencies{
			Profile:       services.Users,
			Articles:      services.Articles,
			Skills:        services.Skills,
			MCPServers:    services.MCPServers,
			Notifications: services.Notifications,
		},
	}

	limiter := platform.NewRateLimiter(infra.Redis, cfg.MCPRateLimitPerMin, 60000000000)
	handler := mcpserver.NewHandler(deps, cfg, limiter, infra, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("mcp server starting", "addr", cfg.MCPAddr)
	if err := app.ServeHTTP(ctx, app.NewHTTPServer(cfg.MCPAddr, handler)); err != nil {
		logger.Error("mcp server exited", "err", err)
	}
}
