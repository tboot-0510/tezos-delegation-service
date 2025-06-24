package service

import (
	"context"
	"fmt"
	"log/slog"
	"tezos-delegation-service/internal/repository"
	"time"
)

type Poller struct {
	ctx            context.Context
	cancel         context.CancelFunc
	repo           repository.DelegationRepository
	client         XtzService
	lastFetched    string
	offset         int
	started        bool
	logger         *slog.Logger
	tickerInterval time.Duration
}

func NewPoller(ctx context.Context, repo repository.DelegationRepository, fetcher XtzService, logger *slog.Logger) *Poller {
	ctx, cancel := context.WithCancel(ctx)
	return &Poller{
		ctx:            ctx,
		cancel:         cancel,
		repo:           repo,
		client:         fetcher,
		lastFetched:    "",
		offset:         0,
		logger:         logger,
		tickerInterval: 1 * time.Minute,
	}
}

func (p *Poller) Stop() {
	p.cancel()
}

func (p *Poller) backfill() {
	p.logger.Info("Starting backfill...")

	// get the latest stored delegation
	latest, err := p.repo.GetLatestDelegation(time.Now().Year())
	fmt.Println("Latest delegation:", latest)
	if err == nil && latest.Timestamp != "" {
		p.lastFetched = latest.Timestamp
	}

	for {
		results, err := p.client.StoreDelegations(0, p.lastFetched)
		if err != nil {
			p.logger.Error("Failed to fetch delegations", "error", err)
			return
		}
		if len(results) == 0 {
			p.logger.Info("No more delegations to fetch, stopping backfill")
			return
		}

		p.logger.Info("Fetched delegations", "count", len(results), "offset", p.offset)
		p.offset += len(results)
		p.lastFetched = (results)[len(results)-1].Timestamp
		p.logger.Info("Updated last fetched level", "timestamp", p.lastFetched)
	}
}

func (p *Poller) Start() {
	if p.started {
		return
	}
	p.started = true
	go func() {
		p.backfill()

		timer := time.NewTicker(p.tickerInterval)
		defer timer.Stop()

		for {
			select {
			case <-p.ctx.Done():
				p.logger.Info("Polling stopped")
				return
			case <-timer.C:
				p.logger.Info("Polling for new delegations...")
				results, err := p.client.StoreDelegations(p.offset, p.lastFetched)
				if err != nil {
					p.logger.Error("Failed to fetch delegations", "error", err)
					p.Stop()
					return
				}
				if len(results) == 0 {
					p.logger.Info("No new delegations found, continuing to poll")
					continue
				}
				p.logger.Info("Fetched new delegations", "count", len(results))
				p.offset += len(results)
				p.lastFetched = (results)[len(results)-1].Timestamp
				p.logger.Info("Updated last fetched level", "timestamp", p.lastFetched)
			}
		}
	}()

}
