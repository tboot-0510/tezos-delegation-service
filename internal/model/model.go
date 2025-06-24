package model

type Delegation struct {
	ID        int    `gorm:"primaryKey" json:"id"`
	Timestamp string `gorm:"index:idx_year_timestamp" json:"timestamp"`
	Amount    int    `json:"amount"`
	Delegator string `json:"address"`
	Level     int    `json:"level"`
	Year      int    `gorm:"index:idx_year_timestamp" json:"year"`
}
