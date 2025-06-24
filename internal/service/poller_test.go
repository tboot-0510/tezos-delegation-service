package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"tezos-delegation-service/internal/model"
	"tezos-delegation-service/internal/repository"
)

type MockPollerRepository struct {
	delegations []model.Delegation
	latest      model.Delegation
	err         error
	saveErr     error
}

func (m *MockPollerRepository) GetDelegations(year int, offset int) ([]model.Delegation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.delegations, nil
}

func (m *MockPollerRepository) GetLatestDelegation(year int) (model.Delegation, error) {
	if m.err != nil {
		return model.Delegation{}, m.err
	}
	return m.latest, nil
}

func (m *MockPollerRepository) SaveBatch(delegations []model.Delegation) error {
	return m.saveErr
}

type MockPollerService struct {
	storeResults [][]model.Delegation
	storeErrors  []error
	callCount    int
	mu           sync.Mutex
}

func (m *MockPollerService) GetDelegations(year int, offset int) ([]model.Delegation, error) {
	return nil, nil
}

func (m *MockPollerService) StoreDelegations(offset int, startFrom string) ([]model.Delegation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.callCount >= len(m.storeResults) {
		return []model.Delegation{}, nil
	}

	result := m.storeResults[m.callCount]
	err := m.storeErrors[m.callCount]
	m.callCount++

	return result, err
}

func (m *MockPollerService) GetLatestDelegation() (model.Delegation, error) {
	return model.Delegation{}, nil
}

func TestNewPoller(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{}
	service := &MockPollerService{}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	if poller == nil {
		t.Fatal("Expected poller to be created, got nil")
	}

	if poller.repo != repo {
		t.Error("Expected repository to be set correctly")
	}

	if poller.client != service {
		t.Error("Expected service to be set correctly")
	}

	if poller.logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if poller.offset != 0 {
		t.Error("Expected initial offset to be 0")
	}

	if poller.lastFetched != "" {
		t.Error("Expected initial lastFetched to be empty")
	}

	if poller.started {
		t.Error("Expected poller to not be started initially")
	}

	select {
	case <-poller.ctx.Done():
		t.Error("Expected context to not be cancelled initially")
	default:
		// context is not cancelled, which is correct
	}
}

func TestPoller_Stop(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{}
	service := &MockPollerService{}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	select {
	case <-poller.ctx.Done():
		t.Error("Expected context to not be cancelled initially")
	default:
		// context is not cancelled, which is correct
	}

	poller.Stop()

	select {
	case <-poller.ctx.Done():
		// context is cancelled, which is correct
	default:
		t.Error("Expected context to be cancelled after Stop()")
	}
}

func TestPoller_Start(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
				{ID: 2, Timestamp: "2023-01-01T01:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
			},
			{}, // stop backfill
		},
		storeErrors: []error{nil, nil},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	if poller.started {
		t.Error("Expected poller to not be started initially")
	}

	poller.Start()

	// wait a bit for the goroutine to start
	time.Sleep(100 * time.Millisecond)

	if !poller.started {
		t.Error("Expected poller to be marked as started")
	}

	poller.Start()
	time.Sleep(100 * time.Millisecond)

	// should only have called StoreDelegations twice
	if service.callCount != 2 {
		t.Errorf("Expected 2 calls to StoreDelegations, got %d", service.callCount)
	}
}

func TestPoller_Backfill(t *testing.T) {
	tests := []struct {
		name           string
		storeResults   [][]model.Delegation
		storeErrors    []error
		expectedOffset int
		expectedLast   string
		shouldStop     bool
	}{
		{
			name: "successful backfill with multiple batches",
			storeResults: [][]model.Delegation{
				{
					{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
					{ID: 2, Timestamp: "2023-01-01T01:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
				},
				{
					{ID: 3, Timestamp: "2023-01-01T02:00:00Z", Amount: 3000, Delegator: "addr3", Level: 102, Year: 2023},
				},
				{}, // stop backfill
			},
			storeErrors:    []error{nil, nil, nil},
			expectedOffset: 3,
			expectedLast:   "2023-01-01T02:00:00Z",
			shouldStop:     true,
		},
		{
			name: "backfill stops on error",
			storeResults: [][]model.Delegation{
				{
					{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
				},
			},
			storeErrors:    []error{errors.New("API error")},
			expectedOffset: 0,
			expectedLast:   "",
			shouldStop:     true,
		},
		{
			name: "backfill with single batch",
			storeResults: [][]model.Delegation{
				{
					{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
				},
				{}, // stop backfill
			},
			storeErrors:    []error{nil, nil},
			expectedOffset: 1,
			expectedLast:   "2023-01-01T00:00:00Z",
			shouldStop:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := &MockPollerRepository{}
			service := &MockPollerService{
				storeResults: tt.storeResults,
				storeErrors:  tt.storeErrors,
			}
			logger := slog.Default()

			poller := NewPoller(ctx, repo, service, logger)

			// run backfill in a goroutine
			go poller.backfill()
			time.Sleep(200 * time.Millisecond)

			if poller.offset != tt.expectedOffset {
				t.Errorf("Expected offset %d, got %d", tt.expectedOffset, poller.offset)
			}

			if poller.lastFetched != tt.expectedLast {
				t.Errorf("Expected lastFetched %s, got %s", tt.expectedLast, poller.lastFetched)
			}
		})
	}
}

func TestPoller_Polling(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{}, // backfill
			{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
			},
			{}, // empty result for first poll
			{
				{ID: 2, Timestamp: "2023-01-01T01:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
			},
		},
		storeErrors: []error{nil, nil, nil, nil},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)
	poller.tickerInterval = 50 * time.Millisecond

	poller.Start()
	time.Sleep(2 * time.Second)

	// should have called StoreDelegations multiple times
	if service.callCount < 3 {
		t.Errorf("Expected at least 3 calls to StoreDelegations, got %d", service.callCount)
	}

	poller.Stop()
	time.Sleep(100 * time.Millisecond)

	select {
	case <-poller.ctx.Done():
		// context is cancelled, which is correct
	default:
		t.Error("Expected context to be cancelled after Stop()")
	}
}

func TestPoller_PollingWithError(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{}, // backfill
			{}, // first poll
		},
		storeErrors: []error{nil, errors.New("API error")},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)
	poller.tickerInterval = 50 * time.Millisecond

	poller.Start()
	time.Sleep(2 * time.Second)

	// should have called StoreDelegations twice (backfill + one poll)
	if service.callCount != 2 {
		t.Errorf("Expected 2 calls to StoreDelegations, got %d", service.callCount)
	}

	select {
	case <-poller.ctx.Done():
		// context is cancelled, which is correct
	default:
		t.Error("Expected context to be cancelled after error")
	}
}

func TestPoller_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
			},
			{}, // empty result to stop backfill
		},
		storeErrors: []error{nil, nil},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			poller.Start()
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// should only have called StoreDelegations twice (once for each result set)
	if service.callCount != 2 {
		t.Errorf("Expected 2 calls to StoreDelegations, got %d", service.callCount)
	}

	if !poller.started {
		t.Error("Expected poller to be marked as started")
	}
}

func TestPoller_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repo := &MockPollerRepository{}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{}, // empty result for backfill
		},
		storeErrors: []error{nil},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	poller.Start()
	time.Sleep(200 * time.Millisecond)

	cancel()
	time.Sleep(200 * time.Millisecond)

	// context should be cancelled
	select {
	case <-poller.ctx.Done():
		// context is cancelled, which is correct
	default:
		t.Error("Expected context to be cancelled after parent context cancellation")
	}
}

func TestPoller_OffsetTracking(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
				{ID: 2, Timestamp: "2023-01-01T01:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
			},
			{
				{ID: 3, Timestamp: "2023-01-01T02:00:00Z", Amount: 3000, Delegator: "addr3", Level: 102, Year: 2023},
			},
			{}, // empty result to stop backfill
		},
		storeErrors: []error{nil, nil, nil},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	// initial offset should be 0
	if poller.offset != 0 {
		t.Errorf("Expected initial offset 0, got %d", poller.offset)
	}

	go poller.backfill()
	time.Sleep(200 * time.Millisecond)

	// offset should be updated to 3 (2 + 1)
	if poller.offset != 3 {
		t.Errorf("Expected offset 3, got %d", poller.offset)
	}

	// last fetched should be updated
	if poller.lastFetched != "2023-01-01T02:00:00Z" {
		t.Errorf("Expected lastFetched '2023-01-01T02:00:00Z', got %s", poller.lastFetched)
	}
}

func TestPoller_BackfillWithLatestDelegation(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{
		latest: model.Delegation{
			ID:        1,
			Timestamp: "2023-01-01T00:00:00Z",
			Amount:    1000,
			Delegator: "addr1",
			Level:     100,
			Year:      2023,
		},
		err: nil,
	}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{
				{ID: 2, Timestamp: "2023-01-01T01:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
			},
			{}, // empty result to stop backfill
		},
		storeErrors: []error{nil, nil},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	go poller.backfill()
	time.Sleep(200 * time.Millisecond)

	// should have used the latest delegation timestamp as startFrom
	if poller.lastFetched != "2023-01-01T01:00:00Z" {
		t.Errorf("Expected lastFetched '2023-01-01T01:00:00Z', got %s", poller.lastFetched)
	}

	if poller.offset != 1 {
		t.Errorf("Expected offset 1, got %d", poller.offset)
	}
}

func TestPoller_BackfillWithRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := &MockPollerRepository{
		err: errors.New("database error"),
	}
	service := &MockPollerService{
		storeResults: [][]model.Delegation{
			{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
			},
			{}, // empty result to stop backfill
		},
		storeErrors: []error{nil, nil},
	}
	logger := slog.Default()

	poller := NewPoller(ctx, repo, service, logger)

	go poller.backfill()
	time.Sleep(200 * time.Millisecond)

	// should start from empty string when repository error occurs
	if poller.lastFetched != "2023-01-01T00:00:00Z" {
		t.Errorf("Expected lastFetched '2023-01-01T00:00:00Z', got %s", poller.lastFetched)
	}

	if poller.offset != 1 {
		t.Errorf("Expected offset 1, got %d", poller.offset)
	}
}

func TestPoller_InterfaceCompliance(t *testing.T) {
	var _ repository.DelegationRepository = (*MockPollerRepository)(nil)
	var _ XtzService = (*MockPollerService)(nil)
}
