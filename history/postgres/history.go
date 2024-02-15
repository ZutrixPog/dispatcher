package postgres

import (
	"context"

	"github.com/ZutrixPog/dispatcher"
	"github.com/ZutrixPog/dispatcher/history"
	"gorm.io/gorm"
)

type PostgresHistoryRepo struct {
	db *gorm.DB
}

func NewHistoryRepo(db *gorm.DB) history.TaskHistoryRepo {
	return &PostgresHistoryRepo{
		db,
	}
}

func (repo *PostgresHistoryRepo) Append(ctx context.Context, report history.TaskReport) error {
	if err := repo.db.WithContext(ctx).Create(&report).Error; err != nil {
		return dispatcher.ErrCreateEntity
	}
	return nil
}

func (repo *PostgresHistoryRepo) Retrieve(ctx context.Context, query history.Query) ([]history.TaskReport, error) {
	var report []history.TaskReport

	if err := query.BuildGormQuery(ctx, repo.db).Find(&report).Error; err != nil {
		return nil, dispatcher.ErrRetrieveEntity
	}

	return report, nil
}
