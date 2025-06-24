package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"tezos-delegation-service/internal/middleware"
	"tezos-delegation-service/internal/model"
	"tezos-delegation-service/mocks"

	"github.com/gorilla/mux"
)

func TestNewApiServer(t *testing.T) {
	service := &mocks.MockXtzService{}
	server := NewApiServer(service)

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	if server.svc != service {
		t.Error("Expected service to be set correctly")
	}
}

func TestVerifyYear(t *testing.T) {
	tests := []struct {
		name        string
		year        int
		parseErr    error
		expected    int
		expectedErr error
	}{
		{
			name:        "valid year 2023",
			year:        2023,
			parseErr:    nil,
			expected:    2023,
			expectedErr: nil,
		},
		{
			name:        "valid year 2018",
			year:        2018,
			parseErr:    nil,
			expected:    2018,
			expectedErr: nil,
		},
		{
			name:        "valid current year",
			year:        time.Now().Year(),
			parseErr:    nil,
			expected:    time.Now().Year(),
			expectedErr: nil,
		},
		{
			name:        "year too old",
			year:        2017,
			parseErr:    nil,
			expected:    0,
			expectedErr: &InvalidYearError{Year: 2017},
		},
		{
			name:        "year too far in future",
			year:        time.Now().Year() + 1,
			parseErr:    nil,
			expected:    0,
			expectedErr: &InvalidYearError{Year: time.Now().Year() + 1},
		},
		{
			name:        "parse error",
			year:        0,
			parseErr:    errors.New("parse error"),
			expected:    0,
			expectedErr: errors.New("parse error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := verifyYear(tt.year, tt.parseErr)

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
				t.Errorf("Expected year %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestInvalidYearError(t *testing.T) {
	err := &InvalidYearError{Year: 2017}
	expected := "Invalid year: 2017"

	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}
func TestHandleGetDelegations(t *testing.T) {
	tests := []struct {
		name            string
		queryParams     string
		mockDelegations []model.Delegation
		mockErr         error
		expectedStatus  int
		expectedBody    string
	}{
		{
			name:        "successful request with year and offset",
			queryParams: "?year=2023&offset=10",
			mockDelegations: []model.Delegation{
				{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
				{ID: 2, Timestamp: "2023-01-02T00:00:00Z", Amount: 2000, Delegator: "addr2", Level: 101, Year: 2023},
			},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"timestamp":"2023-01-01T00:00:00Z","amount":"1000","delegator":"addr1","level":"100"},{"timestamp":"2023-01-02T00:00:00Z","amount":"2000","delegator":"addr2","level":"101"}],"offset":10,"limit":50}`,
		},
		{
			name:        "successful request without parameters (defaults)",
			queryParams: "",
			mockDelegations: []model.Delegation{
				{ID: 1, Timestamp: "2024-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2024},
			},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"timestamp":"2024-01-01T00:00:00Z","amount":"1000","delegator":"addr1","level":"100"}],"offset":0,"limit":50}`,
		},
		{
			name:            "invalid year parameter",
			queryParams:     "?year=2017",
			mockDelegations: nil,
			mockErr:         nil,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `{"error":"Invalid year parameter"}`,
		},
		{
			name:            "invalid year format",
			queryParams:     "?year=invalid",
			mockDelegations: nil,
			mockErr:         nil,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `{"error":"Invalid year parameter"}`,
		},
		{
			name:            "invalid offset format",
			queryParams:     "?offset=invalid",
			mockDelegations: nil,
			mockErr:         nil,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `{"error":"Invalid offset parameter"}`,
		},
		{
			name:            "service error",
			queryParams:     "?year=2023",
			mockDelegations: nil,
			mockErr:         errors.New("database error"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedBody:    `{"error":"database error"}`,
		},
		{
			name:            "empty results",
			queryParams:     "?year=2023",
			mockDelegations: []model.Delegation{},
			mockErr:         nil,
			expectedStatus:  http.StatusOK,
			expectedBody:    `{"data":null,"offset":0,"limit":50}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockXtzService{
				Delegations: tt.mockDelegations,
				Err:         tt.mockErr,
			}

			server := NewApiServer(mockService)

			req := httptest.NewRequest("GET", "/xtz/delegations"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			logger := middleware.Logger
			ctx := context.WithValue(req.Context(), middleware.LoggerKey, logger)
			req = req.WithContext(ctx)

			server.handleGetDelegations(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			var actual map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &actual); err != nil {
				t.Fatalf("Failed to unmarshal response body: %v", err)
			}

			var expected map[string]any
			if err := json.Unmarshal([]byte(tt.expectedBody), &expected); err != nil {
				t.Fatalf("Failed to unmarshal expected body: %v", err)
			}

			if !reflect.DeepEqual(expected, actual) {
				expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
				actualJSON, _ := json.MarshalIndent(actual, "", "  ")
				t.Errorf("Expected body:\n%s\nGot:\n%s", expectedJSON, actualJSON)
			}
		})
	}
}

func TestHandleGetDelegations_Integration(t *testing.T) {
	mockService := &mocks.MockXtzService{
		Delegations: []model.Delegation{
			{ID: 1, Timestamp: "2023-01-01T00:00:00Z", Amount: 1000, Delegator: "addr1", Level: 100, Year: 2023},
		},
		Err: nil,
	}

	server := NewApiServer(mockService)

	router := mux.NewRouter()
	router.HandleFunc("/xtz/delegations", server.handleGetDelegations).Methods("GET")

	req := httptest.NewRequest("GET", "/xtz/delegations?year=2023", nil)
	w := httptest.NewRecorder()

	logger := middleware.Logger
	ctx := context.WithValue(req.Context(), middleware.LoggerKey, logger)
	req = req.WithContext(ctx)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response WrappedResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Errorf("Expected 1 delegation, got %d", len(response.Data))
	}

	if response.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", response.Offset)
	}
}

func TestHandleGetDelegations_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		description    string
		expectedStatus int
	}{
		{
			name:           "year in future",
			queryParams:    "?year=" + strconv.Itoa(time.Now().Year()+1),
			description:    "Year parameter in the future should be invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative offset",
			queryParams:    "?offset=-10",
			description:    "Negative offset should be accepted (validation is in service layer)",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "large offset",
			queryParams:    "?offset=999999",
			description:    "Large offset should be accepted",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "zero year",
			queryParams:    "?year=0",
			description:    "Zero year should be invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockXtzService{
				Delegations: []model.Delegation{},
				Err:         nil,
			}

			server := NewApiServer(mockService)

			req := httptest.NewRequest("GET", "/xtz/delegations"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Add logger to context
			logger := middleware.Logger
			ctx := context.WithValue(req.Context(), middleware.LoggerKey, logger)
			req = req.WithContext(ctx)

			server.handleGetDelegations(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Test '%s': Expected status code %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestWrappedResponse_Serialization(t *testing.T) {
	response := WrappedResponse{
		Data: []DelegationAPIResponse{
			{Timestamp: "2023-01-01T00:00:00Z", Amount: "1000", Delegator: "addr1", Level: "100"},
			{Timestamp: "2023-01-02T00:00:00Z", Amount: "2000", Delegator: "addr2", Level: "101"},
		},
		Offset: 10,
		Limit:  50,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal WrappedResponse: %v", err)
	}

	expected := `{"data":[{"timestamp":"2023-01-01T00:00:00Z","amount":"1000","delegator":"addr1","level":"100"},{"timestamp":"2023-01-02T00:00:00Z","amount":"2000","delegator":"addr2","level":"101"}],"offset":10,"limit":50}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}
