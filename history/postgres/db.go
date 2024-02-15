package postgres

import (
	"errors"

	"github.com/ZutrixPog/dispatcher/history"
	ps "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ErrMigration          = errors.New("database migration failed")
	ErrDbInitiationFailed = errors.New("couldn't connect to database")
)

func InitDB(dsn string, config *gorm.Config) (*gorm.DB, error) {
	db, err := gorm.Open(ps.Open(dsn), config)
	if err != nil {
		return nil, err
	}

	if err := Migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(history.TaskReport{}); err != nil {
		return ErrMigration
	}
	return nil
}
