package repository

import (
	"os"
	"testing"
	"time"

	"tezos-delegation-service/internal/model"

	"github.com/stretchr/testify/assert"
)

type TestDatabase struct {
	*Database
	tempPath string
}

func NewTestDatabase(t *testing.T) *TestDatabase {
	tempFile, err := os.CreateTemp("", "test_db_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()

	db, err := NewDatabase(tempFile.Name())
	if err != nil {
		os.Remove(tempFile.Name())
		t.Fatalf("Failed to create test database: %v", err)
	}

	return &TestDatabase{
		Database: db,
		tempPath: tempFile.Name(),
	}
}

func (td *TestDatabase) Cleanup() {
	os.Remove(td.tempPath)
}

func TestNewDatabase(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid path",
			path:        ":memory:",
			expectError: false,
		},
		{
			name:        "invalid path",
			path:        "/invalid/path/that/does/not/exist/test.db",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDatabase(tt.path)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				assert.NotNil(t, db.db)
			}
		})
	}
}

func TestDatabase_GetDelegations(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	testDelegations := []model.Delegation{
		{
			ID:        1,
			Timestamp: "2023-01-01T00:00:00Z",
			Amount:    1000,
			Delegator: "addr1",
			Level:     100,
			Year:      2023,
		},
		{
			ID:        2,
			Timestamp: "2023-01-02T00:00:00Z",
			Amount:    2000,
			Delegator: "addr2",
			Level:     101,
			Year:      2023,
		},
		{
			ID:        3,
			Timestamp: "2023-01-03T00:00:00Z",
			Amount:    3000,
			Delegator: "addr3",
			Level:     102,
			Year:      2023,
		},
		{
			ID:        4,
			Timestamp: "2024-01-01T00:00:00Z",
			Amount:    4000,
			Delegator: "addr4",
			Level:     200,
			Year:      2024,
		},
	}

	for _, delegation := range testDelegations {
		err := testDB.db.Create(&delegation).Error
		assert.NoError(t, err)
	}

	tests := []struct {
		name          string
		year          int
		offset        int
		expectedCount int
		expectedFirst *model.Delegation
		expectedLast  *model.Delegation
		expectError   bool
	}{
		{
			name:          "get all delegations for 2023",
			year:          2023,
			offset:        0,
			expectedCount: 3,
			expectedFirst: &testDelegations[2], // ordered by timestamp DESC
			expectedLast:  &testDelegations[0],
			expectError:   false,
		},
		{
			name:          "get delegations for 2023 with offset",
			year:          2023,
			offset:        1,
			expectedCount: 2,
			expectedFirst: &testDelegations[1],
			expectedLast:  &testDelegations[0],
			expectError:   false,
		},
		{
			name:          "get delegations for 2024",
			year:          2024,
			offset:        0,
			expectedCount: 1,
			expectedFirst: &testDelegations[3],
			expectedLast:  &testDelegations[3],
			expectError:   false,
		},
		{
			name:          "get delegations for non-existent year",
			year:          2025,
			offset:        0,
			expectedCount: 0,
			expectedFirst: nil,
			expectedLast:  nil,
			expectError:   false,
		},
		{
			name:          "get delegations with large offset",
			year:          2023,
			offset:        10,
			expectedCount: 0,
			expectedFirst: nil,
			expectedLast:  nil,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delegations, err := testDB.GetDelegations(tt.year, tt.offset)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Len(t, delegations, tt.expectedCount)

			if tt.expectedFirst != nil && len(delegations) > 0 {
				assert.Equal(t, tt.expectedFirst.ID, delegations[0].ID)
				assert.Equal(t, tt.expectedFirst.Timestamp, delegations[0].Timestamp)
			}

			if tt.expectedLast != nil && len(delegations) > 0 {
				lastIndex := len(delegations) - 1
				assert.Equal(t, tt.expectedLast.ID, delegations[lastIndex].ID)
				assert.Equal(t, tt.expectedLast.Timestamp, delegations[lastIndex].Timestamp)
			}

			// verify ordering - should be DESC
			for i := 1; i < len(delegations); i++ {
				prevTime, _ := time.Parse(time.RFC3339, delegations[i-1].Timestamp)
				currTime, _ := time.Parse(time.RFC3339, delegations[i].Timestamp)
				assert.True(t, prevTime.After(currTime) || prevTime.Equal(currTime))
			}
		})
	}
}

func TestDatabase_GetLatestDelegation(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	testDelegations := []model.Delegation{
		{
			ID:        1,
			Timestamp: "2023-01-01T00:00:00Z",
			Amount:    1000,
			Delegator: "addr1",
			Level:     100,
			Year:      2023,
		},
		{
			ID:        2,
			Timestamp: "2023-01-02T00:00:00Z",
			Amount:    2000,
			Delegator: "addr2",
			Level:     101,
			Year:      2023,
		},
		{
			ID:        3,
			Timestamp: "2023-01-03T00:00:00Z",
			Amount:    3000,
			Delegator: "addr3",
			Level:     102,
			Year:      2023,
		},
	}

	for _, delegation := range testDelegations {
		err := testDB.db.Create(&delegation).Error
		assert.NoError(t, err)
	}

	tests := []struct {
		name        string
		year        int
		expected    *model.Delegation
		expectError bool
	}{
		{
			name:        "get latest delegation for 2023",
			year:        2023,
			expected:    &testDelegations[2],
			expectError: false,
		},
		{
			name:        "get latest delegation for non-existent year",
			year:        2025,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delegation, err := testDB.GetLatestDelegation(tt.year)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, delegation)
				assert.Equal(t, tt.expected.ID, delegation.ID)
				assert.Equal(t, tt.expected.Timestamp, delegation.Timestamp)

				assert.Equal(t, 0, delegation.Amount)
				assert.Equal(t, "", delegation.Delegator)
				assert.Equal(t, 0, delegation.Level)
				assert.Equal(t, 0, delegation.Year)
			}
		})
	}
}

func TestDatabase_SaveBatch(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	tests := []struct {
		name        string
		delegations []model.Delegation
		expectError bool
	}{
		{
			name: "save single delegation",
			delegations: []model.Delegation{
				{
					ID:        1,
					Timestamp: "2023-01-01T00:00:00Z",
					Amount:    1000,
					Delegator: "addr1",
					Level:     100,
					Year:      2023,
				},
			},
			expectError: false,
		},
		{
			name: "save multiple delegations",
			delegations: []model.Delegation{
				{
					ID:        1,
					Timestamp: "2023-01-01T00:00:00Z",
					Amount:    1000,
					Delegator: "addr1",
					Level:     100,
					Year:      2023,
				},
				{
					ID:        2,
					Timestamp: "2023-01-02T00:00:00Z",
					Amount:    2000,
					Delegator: "addr2",
					Level:     101,
					Year:      2023,
				},
			},
			expectError: false,
		},
		{
			name:        "save empty batch",
			delegations: []model.Delegation{},
			expectError: false,
		},
		{
			name: "save delegation with duplicate ID (should ignore)",
			delegations: []model.Delegation{
				{
					ID:        1,
					Timestamp: "2023-01-01T00:00:00Z",
					Amount:    1000,
					Delegator: "addr1",
					Level:     100,
					Year:      2023,
				},
				{
					ID:        1,
					Timestamp: "2023-01-02T00:00:00Z",
					Amount:    2000,
					Delegator: "addr2",
					Level:     101,
					Year:      2023,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testDB.SaveBatch(tt.delegations)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if len(tt.delegations) > 0 {
				var count int64
				testDB.db.Model(&model.Delegation{}).Count(&count)
				assert.GreaterOrEqual(t, count, int64(len(tt.delegations)))
			}
		})
	}
}

func TestDatabase_SaveBatch_Transaction(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	delegations := []model.Delegation{
		{
			ID:        1,
			Timestamp: "2023-01-01T00:00:00Z",
			Amount:    1000,
			Delegator: "addr1",
			Level:     100,
			Year:      2023,
		},
		{
			ID:        2,
			Timestamp: "2023-01-02T00:00:00Z",
			Amount:    2000,
			Delegator: "addr2",
			Level:     101,
			Year:      2023,
		},
	}

	err := testDB.SaveBatch(delegations)
	assert.NoError(t, err)

	var savedDelegations []model.Delegation
	err = testDB.db.Find(&savedDelegations).Error
	assert.NoError(t, err)
	assert.Len(t, savedDelegations, 2)
}

func TestDatabase_IndexCreation(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	// verify that the index was created
	var indexes []struct {
		Name string
	}
	err := testDB.db.Raw("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_year_timestamp_desc'").Scan(&indexes).Error
	assert.NoError(t, err)
	assert.Len(t, indexes, 1)
	assert.Equal(t, "idx_year_timestamp_desc", indexes[0].Name)
}

func TestDatabase_GetDelegations_Limit(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	for i := 1; i <= 150; i++ {
		delegation := model.Delegation{
			ID:        i,
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			Amount:    i * 1000,
			Delegator: "addr" + string(rune(i)),
			Level:     i,
			Year:      2023,
		}
		err := testDB.db.Create(&delegation).Error
		assert.NoError(t, err)
	}

	// limit is 50
	delegations, err := testDB.GetDelegations(2023, 0)
	assert.NoError(t, err)
	assert.Len(t, delegations, 50)

	// test offset works correctly
	delegations, err = testDB.GetDelegations(2023, 100)
	assert.NoError(t, err)
	assert.Len(t, delegations, 50)
}

func TestDatabase_GetLatestDelegation_EmptyDatabase(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	delegation, err := testDB.GetLatestDelegation(2023)
	assert.Error(t, err) // should return error when no records found
	assert.Equal(t, model.Delegation{}, delegation)
}

func TestDatabase_SaveBatch_DuplicateHandling(t *testing.T) {
	testDB := NewTestDatabase(t)
	defer testDB.Cleanup()

	delegation1 := model.Delegation{
		ID:        1,
		Timestamp: "2023-01-01T00:00:00Z",
		Amount:    1000,
		Delegator: "addr1",
		Level:     100,
		Year:      2023,
	}

	err := testDB.SaveBatch([]model.Delegation{delegation1})
	assert.NoError(t, err)

	// try to save the same delegation again (should be ignored due to ON CONFLICT DO NOTHING)
	delegation2 := model.Delegation{
		ID:        1,                      // same ID
		Timestamp: "2023-01-02T00:00:00Z", // different timestamp
		Amount:    2000,                   // different amount
		Delegator: "addr2",                // different delegator
		Level:     101,                    // different level
		Year:      2024,                   // different year
	}

	err = testDB.SaveBatch([]model.Delegation{delegation2})
	assert.NoError(t, err)

	var delegations []model.Delegation
	err = testDB.db.Find(&delegations).Error
	assert.NoError(t, err)
	assert.Len(t, delegations, 1)
	assert.Equal(t, delegation1.ID, delegations[0].ID)
	assert.Equal(t, delegation1.Timestamp, delegations[0].Timestamp)
	assert.Equal(t, delegation1.Amount, delegations[0].Amount)
	assert.Equal(t, delegation1.Delegator, delegations[0].Delegator)
	assert.Equal(t, delegation1.Level, delegations[0].Level)
	assert.Equal(t, delegation1.Year, delegations[0].Year)
}

func TestDatabase_InterfaceCompliance(t *testing.T) {
	var _ DelegationRepository = (*Database)(nil)
}
