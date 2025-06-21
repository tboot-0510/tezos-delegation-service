package transport

// This package provides the transport layer for the Tezos delegation service.
// It handles the communication with the Tezos API to fetch delegation data.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type DelegationResponse struct {
	Timestamp string `json:"timestamp"`
	Amount    int    `json:"amount"`
	Sender    struct {
		Address string `json:"address"`
	} `json:"sender"`
	Level int `json:"level"`
}

type TzktClient struct {
	apiURL string
}

func NewTzktClient(apiURL string) *TzktClient {
	return &TzktClient{
		apiURL: apiURL,
	}
}

func (c *TzktClient) GetDelegations(ctx context.Context, year string) (*[]DelegationResponse, error) {
	resp, err := http.Get(c.apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var entry []DelegationResponse
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, err
	}

	return &entry, nil
}
