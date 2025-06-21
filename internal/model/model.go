package model

type Delegation struct {
	Timestamp string `json:"timestamp"`
	Amount    int    `json:"amount"`
	Delegator string `json:"address"`
	Level     int    `json:"level"`
}
