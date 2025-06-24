package mocks

import (
	"tezos-delegation-service/internal/transport"
)

type MockTzktClient struct {
	Delegations *[]transport.DelegationResponse
	Err         error
}

func (m *MockTzktClient) GetDelegations(offset int, fromTimestamp string) (*[]transport.DelegationResponse, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Delegations, nil
}
