package mocks

import (
	"context"

	"github.com/ZutrixPog/dispatcher/history"
)

var _ history.TaskHistoryRepo = (*MockHistoryRepo)(nil)

type MockHistoryRepo struct {
	history []history.TaskReport
}

func NewMockHistoryRepo() history.TaskHistoryRepo {
	return &MockHistoryRepo{
		history: make([]history.TaskReport, 0),
	}
}

func (repo *MockHistoryRepo) Append(ctx context.Context, report history.TaskReport) error {
	repo.history = append(repo.history, report)
	return nil
}

func (repo *MockHistoryRepo) Retrieve(ctx context.Context, query history.Query) ([]history.TaskReport, error) {
	if len(repo.history) == 0 {
		return nil, nil
	}
	res := make([]history.TaskReport, 0)
	for i := len(repo.history) - 1; i >= max(0, len(repo.history)-int(query.Limit)); i-- {
		cond := true
		if query.Type != "" {
			cond = cond && repo.history[i].Type == query.Type
		}
		if query.Status != "" {
			cond = cond && repo.history[i].Status == query.Status
		}
		if query.Queue != "" {
			cond = cond && repo.history[i].Queue == query.Queue
		}

		if cond {
			res = append(res, repo.history[i])
		}
	}

	return res, nil
}

func max(a, b int) int {
	if a >= b {
		return a
	}

	return b
}
