package repository

import (
	"tezos-delegation-service/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	db *gorm.DB
}

type DelegationRepository interface {
	GetDelegations(year int, offset int) ([]model.Delegation, error)
	SaveBatch([]model.Delegation) error
	GetLatestDelegation(year int) (model.Delegation, error)
}

func NewDatabase(path string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Delegation{}); err != nil {
		return nil, err
	}

	rawIndex := `
		CREATE INDEX IF NOT EXISTS idx_year_timestamp_desc 
		ON delegations (year, timestamp DESC);
	`
	if err := db.Exec(rawIndex).Error; err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

func (d *Database) GetDelegations(year int, offset int) ([]model.Delegation, error) {
	db := d.db
	var delegations []model.Delegation

	limit := 50
	err := db.Where("year = ?", year).
		Order("timestamp DESC").
		Offset(offset).
		Limit(limit).
		Find(&delegations).Error

	if err == gorm.ErrRecordNotFound {
		return []model.Delegation{}, nil
	}

	return delegations, err
}

func (d *Database) GetLatestDelegation(year int) (model.Delegation, error) {
	db := d.db
	var delegation model.Delegation

	err := db.Select("id", "timestamp").
		Where("year = ?", year).
		Order("timestamp DESC").
		Limit(1).
		First(&delegation).Error

	return delegation, err
}

func (d *Database) SaveBatch(delegations []model.Delegation) error {
	if len(delegations) == 0 {
		return nil
	}

	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&delegations).Error; err != nil {
			return err
		}
		return nil
	})
}
