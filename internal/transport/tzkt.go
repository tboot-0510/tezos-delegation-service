package transport

// This package provides the transport layer for the Tezos delegation service.
// It handles the communication with the Tezos API to fetch delegation data.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type DelegationResponse struct {
	ID        int    `json:"id"`
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

type TzktClientInterface interface {
	GetDelegations(offset int, fromTimestamp string) (*[]DelegationResponse, error)
}

func NewTzktClient(apiURL string) *TzktClient {
	return &TzktClient{
		apiURL: apiURL,
	}
}

func (c *TzktClient) GetDelegations(offset int, fromTimestamp string) (*[]DelegationResponse, error) {
	u, err := url.Parse(c.apiURL)
	if err != nil {
		return nil, err
	}

	query := u.Query()
	if fromTimestamp != "" {
		query.Add("timestamp.gt", fromTimestamp)
	}
	if offset > 0 {
		query.Add("offset", fmt.Sprintf("%d", offset))
	}

	u.RawQuery = query.Encode()

	baseUrl := u.String()

	resp, err := http.Get(baseUrl)
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

var _ TzktClientInterface = (*TzktClient)(nil)
