package mocks

import (
	"tezos-delegation-service/internal/model"
)

type MockDelegationRepository struct {
	Delegations []model.Delegation
	Latest      model.Delegation
	Err         error
	SaveErr     error
}

func (m *MockDelegationRepository) GetDelegations(year int, offset int) ([]model.Delegation, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Delegations, nil
}

func (m *MockDelegationRepository) GetLatestDelegation(year int) (model.Delegation, error) {
	if m.Err != nil {
		return model.Delegation{}, m.Err
	}
	return m.Latest, nil
}

func (m *MockDelegationRepository) SaveBatch(delegations []model.Delegation) error {
	return m.SaveErr
}
