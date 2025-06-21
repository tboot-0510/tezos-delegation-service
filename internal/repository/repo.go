package repository

import (
	"tezos-delegation-service/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

type DelegationRepository interface {
	GetDelegations(year int, page int) ([]model.Delegation, error)
	SaveBatch([]model.Delegation) error
}

func NewDatabase(path string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Delegation{}); err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

func (d *Database) GetDelegations(year int, page int) ([]model.Delegation, error) {
	db := d.db
	var delegations []model.Delegation

	// handle offset and limit for pagination
	err := db.Where("year = ?", year).Offset((page - 1) * 50).Find(&delegations).Error

	return delegations, err
}

func (d *Database) SaveBatch(delegations []model.Delegation) error {
	// no need to chunk as the default limit is 100
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&delegations).Error; err != nil {
			return err
		}
		return nil
	})
}
