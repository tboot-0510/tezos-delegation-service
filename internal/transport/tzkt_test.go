package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewTzktClient(t *testing.T) {
	apiURL := "https://api.tzkt.io/v1/operations/delegations"
	client := NewTzktClient(apiURL)

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.apiURL != apiURL {
		t.Errorf("Expected apiURL %s, got %s", apiURL, client.apiURL)
	}
}

func TestTzktClient_GetDelegations_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []DelegationResponse{
			{
				ID:        1,
				Timestamp: "2023-01-01T01:00:00Z",
				Amount:    1000,
				Sender: struct {
					Address string `json:"address"`
				}{Address: "addr1"},
				Level: 100,
			},
			{
				ID:        2,
				Timestamp: "2023-01-01T02:00:00Z",
				Amount:    2000,
				Sender: struct {
					Address string `json:"address"`
				}{Address: "addr2"},
				Level: 101,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewTzktClient(server.URL)

	results, err := client.GetDelegations(10, "2023-01-01T00:00:00Z")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	if len(*results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(*results))
	}

	first := (*results)[0]
	if first.ID != 1 {
		t.Errorf("Expected ID 1, got %d", first.ID)
	}
	if first.Timestamp != "2023-01-01T01:00:00Z" {
		t.Errorf("Expected timestamp '2023-01-01T01:00:00Z', got %s", first.Timestamp)
	}
	if first.Amount != 1000 {
		t.Errorf("Expected amount 1000, got %d", first.Amount)
	}
	if first.Sender.Address != "addr1" {
		t.Errorf("Expected sender address 'addr1', got %s", first.Sender.Address)
	}
	if first.Level != 100 {
		t.Errorf("Expected level 100, got %d", first.Level)
	}

	second := (*results)[1]
	if second.ID != 2 {
		t.Errorf("Expected ID 2, got %d", second.ID)
	}
	if second.Timestamp != "2023-01-01T02:00:00Z" {
		t.Errorf("Expected timestamp '2023-01-01T02:00:00Z', got %s", second.Timestamp)
	}
	if second.Amount != 2000 {
		t.Errorf("Expected amount 2000, got %d", second.Amount)
	}
	if second.Sender.Address != "addr2" {
		t.Errorf("Expected sender address 'addr2', got %s", second.Sender.Address)
	}
	if second.Level != 101 {
		t.Errorf("Expected level 101, got %d", second.Level)
	}
}

func TestTzktClient_GetDelegations_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewTzktClient(server.URL)

	results, err := client.GetDelegations(0, "")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedError := "unexpected status code: 500"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	if results != nil {
		t.Errorf("Expected nil results, got %v", results)
	}
}

func TestTzktClient_GetDelegations_NetworkError(t *testing.T) {
	client := NewTzktClient("http://invalid-url-that-does-not-exist.com")

	results, err := client.GetDelegations(0, "")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if results != nil {
		t.Errorf("Expected nil results, got %v", results)
	}
}

func TestTzktClient_GetDelegations_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewTzktClient(server.URL)

	results, err := client.GetDelegations(0, "")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if results != nil {
		t.Errorf("Expected nil results, got %v", results)
	}
}

func TestTzktClient_URLConstruction(t *testing.T) {
	tests := []struct {
		name          string
		offset        int
		timestamp     string
		expectedQuery string
	}{
		{
			name:          "no parameters",
			offset:        0,
			timestamp:     "",
			expectedQuery: "/v1/operations/delegations",
		},
		{
			name:          "only offset",
			offset:        10,
			timestamp:     "",
			expectedQuery: "/v1/operations/delegations?offset=10",
		},
		{
			name:          "only timestamp",
			offset:        0,
			timestamp:     "2023-01-01T00:00:00Z",
			expectedQuery: "/v1/operations/delegations?timestamp.gt=2023-01-01T00%3A00%3A00Z",
		},
		{
			name:          "both parameters",
			offset:        5,
			timestamp:     "2023-01-01T00:00:00Z",
			expectedQuery: "/v1/operations/delegations?offset=5&timestamp.gt=2023-01-01T00%3A00%3A00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedQuery string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedQuery = r.URL.String()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]DelegationResponse{})
			}))
			defer server.Close()

			testClient := NewTzktClient(server.URL + "/v1/operations/delegations")

			_, err := testClient.GetDelegations(tt.offset, tt.timestamp)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if capturedQuery != tt.expectedQuery {
				t.Errorf("expected query '%s', got '%s'", tt.expectedQuery, capturedQuery)
			}
		})
	}
}
