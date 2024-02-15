package history

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type TaskHistoryRepo interface {
	Append(ctx context.Context, report TaskReport) error
	Retrieve(ctx context.Context, query Query) ([]TaskReport, error)
}

type TaskReport struct {
	ID        uint      `gorm:"primaryKey;not null;unique;autoIncrement" json:"id"`
	Type      string    `gorm:"not null" json:"type"`
	Status    string    `gorm:"not null" json:"status"`
	Channel   string    `gorm:"not null" json:"channel"`
	Submitted time.Time `json:"submitted"`
	CreatedAt time.Time `json:"completed,omitempty"`
}

type Query struct {
	Limit   int
	Offset  int
	Status  string
	Type    string
	Channel string
}

func (query Query) BuildGormQuery(ctx context.Context, db *gorm.DB) *gorm.DB {
	queryBuilder := db.WithContext(ctx).Model(&TaskReport{})

	if query.Limit > 0 {
		queryBuilder = queryBuilder.Limit(query.Limit)
	}

	if query.Offset > 0 {
		queryBuilder = queryBuilder.Offset(query.Offset)
	}

	if query.Status != "" {
		queryBuilder = queryBuilder.Where(&TaskReport{Status: query.Status})
	}

	if query.Type != "" {
		queryBuilder = queryBuilder.Where(&TaskReport{Type: query.Type})
	}

	if query.Channel != "" {
		queryBuilder = queryBuilder.Where(&TaskReport{Channel: query.Channel})
	}

	queryBuilder = queryBuilder.Order("created_at DESC")

	return queryBuilder
}

type DummyTaskHistoryRepo struct{}

func (dummy *DummyTaskHistoryRepo) Append(ctx context.Context, report TaskReport) error {
	return nil
}

func (dummy *DummyTaskHistoryRepo) Retrieve(ctx context.Context, query Query) ([]TaskReport, error) {
	return nil, nil
}
