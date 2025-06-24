package service

import (
	"errors"
	"testing"

	"tezos-delegation-service/internal/model"
	"tezos-delegation-service/internal/transport"
	"tezos-delegation-service/mocks"
)

func TestNewXtzFetcherService(t *testing.T) {
	repo := &mocks.MockDelegationRepository{}
	client := &mocks.MockTzktClient{}

	service := NewXtzFetcherService(repo, client)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	_, ok := service.(*XtzFetcherService)
	if !ok {
		t.Fatal("Expected service to be of type *XtzFetcherService")
	}
}

func TestGetDelegations(t *testing.T) {
	tests := []struct {
		name            string
		year            int
		offset          int
		mockDelegations []model.Delegation
		mockErr         error
		expectedResult  []model.Delegation
		expectedErr     error
	}{
		{
			name:   "successful retrieval",
			year:   2023,
			offset: 10,
			mockDelegations: []model.Delegation{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
				{ID: 2, Timestamp: "2023-01-02T00:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
			},
			mockErr: nil,
			expectedResult: []model.Delegation{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
				{ID: 2, Timestamp: "2023-01-02T00:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
			},
			expectedErr: nil,
		},
		{
			name:            "repository error",
			year:            2023,
			offset:          10,
			mockDelegations: nil,
			mockErr:         errors.New("database error"),
			expectedResult:  nil,
			expectedErr:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockDelegationRepository{
				Delegations: tt.mockDelegations,
				Err:         tt.mockErr,
			}
			client := &mocks.MockTzktClient{}

			service := NewXtzFetcherService(repo, client)

			result, err := service.GetDelegations(tt.year, tt.offset)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.expectedErr)
				} else if err.Error() != tt.expectedErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}

			if len(result) != len(tt.expectedResult) {
				t.Errorf("Expected %d delegations, got %d", len(tt.expectedResult), len(result))
			}

			for i, expected := range tt.expectedResult {
				if i >= len(result) {
					t.Errorf("Expected delegation at index %d, but result has only %d items", i, len(result))
					continue
				}
				if result[i] != expected {
					t.Errorf("Expected delegation %+v, got %+v", expected, result[i])
				}
			}
		})
	}
}

func TestGetLatestDelegation(t *testing.T) {
	tests := []struct {
		name        string
		mockLatest  model.Delegation
		mockErr     error
		expected    model.Delegation
		expectedErr error
	}{
		{
			name: "successful retrieval",
			mockLatest: model.Delegation{
				ID:        1,
				Timestamp: "2023-12-31T23:59:59Z",
				Amount:    5000,
				Delegator: "addr1",
				Level:     1000,
				Year:      2023,
			},
			mockErr: nil,
			expected: model.Delegation{
				ID:        1,
				Timestamp: "2023-12-31T23:59:59Z",
				Amount:    5000,
				Delegator: "addr1",
				Level:     1000,
				Year:      2023,
			},
			expectedErr: nil,
		},
		{
			name:        "repository error",
			mockLatest:  model.Delegation{},
			mockErr:     errors.New("database error"),
			expected:    model.Delegation{},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockDelegationRepository{
				Latest: tt.mockLatest,
				Err:    tt.mockErr,
			}
			client := &mocks.MockTzktClient{}

			service := NewXtzFetcherService(repo, client)

			result, err := service.GetLatestDelegation()

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.expectedErr)
				} else if err.Error() != tt.expectedErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}

			if result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestStoreDelegations(t *testing.T) {
	tests := []struct {
		name              string
		offset            int
		mockLatest        model.Delegation
		mockLatestErr     error
		mockClientResults *[]transport.DelegationResponse
		mockClientErr     error
		mockSaveErr       error
		expectedResult    []model.Delegation
		expectedErr       error
	}{
		{
			name:   "successful store with existing latest delegation",
			offset: 10,
			mockLatest: model.Delegation{
				ID:        1,
				Timestamp: "2023-12-31T23:59:59Z",
				Amount:    5000,
				Delegator: "addr1",
				Level:     1000,
				Year:      2023,
			},
			mockLatestErr: nil,
			mockClientResults: &[]transport.DelegationResponse{
				{
					ID:        2,
					Timestamp: "2024-01-01T00:00:00Z",
					Amount:    1000,
					Sender: struct {
						Address string `json:"address"`
					}{Address: "addr2"},
					Level: 1001,
				},
				{
					ID:        3,
					Timestamp: "2024-01-01T01:00:00Z",
					Amount:    2000,
					Sender: struct {
						Address string `json:"address"`
					}{Address: "addr3"},
					Level: 1002,
				},
			},
			mockClientErr: nil,
			mockSaveErr:   nil,
			expectedResult: []model.Delegation{
				{
					ID:        2,
					Timestamp: "2024-01-01T00:00:00Z",
					Amount:    1000,
					Delegator: "addr2",
					Level:     1001,
					Year:      2024,
				},
				{
					ID:        3,
					Timestamp: "2024-01-01T01:00:00Z",
					Amount:    2000,
					Delegator: "addr3",
					Level:     1002,
					Year:      2024,
				},
			},
			expectedErr: nil,
		},
		{
			name:          "successful store without existing latest delegation",
			offset:        10,
			mockLatest:    model.Delegation{},
			mockLatestErr: errors.New("not found"),
			mockClientResults: &[]transport.DelegationResponse{
				{
					ID:        1,
					Timestamp: "2024-01-01T00:00:00Z",
					Amount:    1000,
					Sender: struct {
						Address string `json:"address"`
					}{Address: "addr1"},
					Level: 1000,
				},
			},
			mockClientErr: nil,
			mockSaveErr:   nil,
			expectedResult: []model.Delegation{
				{
					ID:        1,
					Timestamp: "2024-01-01T00:00:00Z",
					Amount:    1000,
					Delegator: "addr1",
					Level:     1000,
					Year:      2024,
				},
			},
			expectedErr: nil,
		},
		{
			name:              "client error",
			offset:            10,
			mockLatest:        model.Delegation{},
			mockLatestErr:     errors.New("not found"),
			mockClientResults: nil,
			mockClientErr:     errors.New("API error"),
			mockSaveErr:       nil,
			expectedResult:    nil,
			expectedErr:       errors.New("API error"),
		},
		{
			name:   "save error",
			offset: 10,
			mockLatest: model.Delegation{
				ID:        1,
				Timestamp: "2023-12-31T23:59:59Z",
				Amount:    5000,
				Delegator: "addr1",
				Level:     1000,
				Year:      2023,
			},
			mockLatestErr: nil,
			mockClientResults: &[]transport.DelegationResponse{
				{
					ID:        2,
					Timestamp: "2024-01-01T00:00:00Z",
					Amount:    1000,
					Sender: struct {
						Address string `json:"address"`
					}{Address: "addr2"},
					Level: 1001,
				},
			},
			mockClientErr: nil,
			mockSaveErr:   errors.New("save error"),
			expectedResult: []model.Delegation{
				{
					ID:        2,
					Timestamp: "2024-01-01T00:00:00Z",
					Amount:    1000,
					Delegator: "addr2",
					Level:     1001,
					Year:      2024,
				},
			},
			expectedErr: errors.New("save error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockDelegationRepository{
				Latest:  tt.mockLatest,
				Err:     tt.mockLatestErr,
				SaveErr: tt.mockSaveErr,
			}
			client := &mocks.MockTzktClient{
				Delegations: tt.mockClientResults,
				Err:         tt.mockClientErr,
			}

			service := NewXtzFetcherService(repo, client)

			result, err := service.StoreDelegations(tt.offset, "")

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.expectedErr)
				} else if err.Error() != tt.expectedErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}

			if len(result) != len(tt.expectedResult) {
				t.Errorf("Expected %d delegations, got %d", len(tt.expectedResult), len(result))
			}

			for i, expected := range tt.expectedResult {
				if i >= len(result) {
					t.Errorf("Expected delegation at index %d, but result has only %d items", i, len(result))
					continue
				}
				if result[i] != expected {
					t.Errorf("Expected delegation %+v, got %+v", expected, result[i])
				}
			}
		})
	}
}

func TestStoreDelegations_StartFromLogic(t *testing.T) {
	// test that startFrom is correctly set based on latest delegation
	repo := &mocks.MockDelegationRepository{
		Latest: model.Delegation{
			ID:        1,
			Timestamp: "2023-12-31T23:59:59Z",
			Amount:    5000,
			Delegator: "addr1",
			Level:     1000,
			Year:      2023,
		},
		Err:     nil,
		SaveErr: nil,
	}

	client := &mocks.MockTzktClient{
		Delegations: &[]transport.DelegationResponse{},
		Err:         nil,
	}

	service := NewXtzFetcherService(repo, client)

	_, err := service.StoreDelegations(10, "2023-12-31T23:59:59Z")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestStoreDelegations_EmptyResults(t *testing.T) {
	repo := &mocks.MockDelegationRepository{
		Latest:  model.Delegation{},
		Err:     errors.New("not found"),
		SaveErr: nil,
	}

	client := &mocks.MockTzktClient{
		Delegations: &[]transport.DelegationResponse{},
		Err:         nil,
	}

	service := NewXtzFetcherService(repo, client)

	result, err := service.StoreDelegations(10, "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d delegations", len(result))
	}
}
