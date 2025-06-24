package service

import (
	"tezos-delegation-service/internal/model"
	"tezos-delegation-service/internal/repository"
	"tezos-delegation-service/internal/transport"
	"time"
)

type XtzService interface {
	GetDelegations(year int, offset int) ([]model.Delegation, error)
	StoreDelegations(offset int, startFrom string) ([]model.Delegation, error)
	GetLatestDelegation() (model.Delegation, error)
}

type XtzFetcherService struct {
	repo       repository.DelegationRepository
	tzklClient transport.TzktClientInterface
}

func NewXtzFetcherService(repo repository.DelegationRepository, client transport.TzktClientInterface) XtzService {
	return &XtzFetcherService{
		repo:       repo,
		tzklClient: client,
	}
}

func (s *XtzFetcherService) GetDelegations(year int, offset int) ([]model.Delegation, error) {
	return s.repo.GetDelegations(year, offset)
}

func (s *XtzFetcherService) GetLatestDelegation() (model.Delegation, error) {
	return s.repo.GetLatestDelegation(time.Now().Year())
}

func (s *XtzFetcherService) StoreDelegations(offset int, startFrom string) ([]model.Delegation, error) {
	results, err := s.tzklClient.GetDelegations(offset, startFrom)
	if err != nil {
		return nil, err
	}

	var delegations []model.Delegation
	for _, result := range *results {
		parsedTimestamp, err := time.Parse(time.RFC3339, result.Timestamp)
		if err != nil {
			return nil, err
		}

		delegations = append(delegations, model.Delegation{
			ID:        result.ID,
			Timestamp: result.Timestamp,
			Amount:    result.Amount,
			Delegator: result.Sender.Address,
			Level:     result.Level,
			Year:      parsedTimestamp.Year(),
		})
	}

	return delegations, s.repo.SaveBatch(delegations)
}
