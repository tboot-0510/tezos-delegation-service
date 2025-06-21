package service

import (
	"context"
	"tezos-delegation-service/internal/model"
	"tezos-delegation-service/internal/repository"
	"tezos-delegation-service/internal/transport"
)

type XtzService interface {
	GetDelegations(ctx context.Context, year int) ([]model.Delegation, error)
}

type XtzFetcherService struct {
	repo       repository.DelegationRepository
	tzklClient *transport.TzktClient
}

func NewXtzFetcherService(repo repository.DelegationRepository, client *transport.TzktClient) XtzService {
	return &XtzFetcherService{
		repo:       repo,
		tzklClient: client,
	}
}

func (s *XtzFetcherService) GetDelegations(ctx context.Context, year int) ([]model.Delegation, error) {
	return s.repo.GetDelegations(year, 1)
}

func (s *XtzFetcherService) StoreDelegations(ctx context.Context) error {
	results, err := s.tzklClient.GetDelegations(ctx, "")
	if err != nil {
		return err
	}

	var delegations []model.Delegation
	for _, result := range *results {
		delegations = append(delegations, model.Delegation{
			Timestamp: result.Timestamp,
			Amount:    result.Amount,
			Delegator: result.Sender.Address,
			Level:     result.Level,
		})
	}

	return s.repo.SaveBatch(delegations)
}
