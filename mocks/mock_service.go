package mocks

import "tezos-delegation-service/internal/model"

type MockXtzService struct {
	Delegations []model.Delegation
	Err         error
}

func (m *MockXtzService) GetDelegations(year int, offset int) ([]model.Delegation, error) {
	return m.Delegations, m.Err
}

func (m *MockXtzService) StoreDelegations(offset int, startFrom string) ([]model.Delegation, error) {
	return m.Delegations, m.Err
}

func (m *MockXtzService) GetLatestDelegation() (model.Delegation, error) {
	if len(m.Delegations) > 0 {
		return m.Delegations[0], m.Err
	}
	return model.Delegation{}, m.Err
}
