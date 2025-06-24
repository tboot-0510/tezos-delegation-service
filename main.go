package main

import (
	"context"
	"log/slog"
	"os"
	"tezos-delegation-service/internal/api"
	"tezos-delegation-service/internal/middleware"
	"tezos-delegation-service/internal/repository"
	"tezos-delegation-service/internal/service"
	"tezos-delegation-service/internal/transport"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	middleware.Logger = logger

	// init the transport layer - calls tzkt API
	tzkt := transport.NewTzktClient("https://api.tzkt.io/v1/operations/delegations?limit=1000")

	// init the repository layer - uses sqlite
	repo, err := repository.NewDatabase("delegations.db")
	if err != nil {
		logger.Error("❌❌❌ Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// init the service layer - uses tzkt client and repository
	// this is the business logic layer - it fetches data from the tzkt client and stores it in the repository
	svc := service.NewXtzFetcherService(repo, tzkt)

	// Get the delegations at startup
	go func() {
		ctx := context.Background()
		poller := service.NewPoller(ctx, repo, svc, logger)
		poller.Start()

	}()

	server := api.NewApiServer(svc)
	server.Start(":3000")
}
