package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ZutrixPog/dispatcher/history"
	"github.com/ZutrixPog/dispatcher/history/postgres"
	"github.com/stretchr/testify/require"
)

var (
	report1 = history.TaskReport{
		Type:   "test",
		Status: "success",
	}

	report2 = history.TaskReport{
		Type:   "test",
		Status: "failed",
	}

	report3 = history.TaskReport{
		Type:   "mest",
		Status: "success",
	}
)

func TestSave(t *testing.T) {
	repo := InitRepo()

	cases := []struct {
		desc   string
		report history.TaskReport
		err    error
	}{
		{
			desc:   "store valid event",
			report: report1,
			err:    nil,
		},
	}

	for _, c := range cases {
		err := repo.Append(context.Background(), c.report)
		require.Equal(t, c.err, err, c.desc)
	}
}

func TestRetrieve(t *testing.T) {
	repo := InitRepo()
	ctx := context.Background()

	require.Nil(t, repo.Append(ctx, report1))
	require.Nil(t, repo.Append(ctx, report2))
	require.Nil(t, repo.Append(ctx, report3))

	cases := []struct {
		desc  string
		query history.Query
		res   []history.TaskReport
		err   error
	}{
		{
			desc:  "query all data",
			query: history.Query{},
			res:   []history.TaskReport{report1, report2, report3},
			err:   nil,
		},
		{
			desc: "query using offset and limit",
			query: history.Query{
				Offset: 1,
				Limit:  1,
			},
			res: []history.TaskReport{report2},
			err: nil,
		},
		{
			desc: "query by type",
			query: history.Query{
				Type: report1.Type,
			},
			res: []history.TaskReport{report1, report2},
			err: nil,
		},
		{
			desc: "query by status",
			query: history.Query{
				Status: "success",
			},
			res: []history.TaskReport{report1, report3},
			err: nil,
		},
	}

	for _, c := range cases {
		res, err := repo.Retrieve(ctx, history.Query{Type: c.query.Type, Status: c.query.Status, Limit: c.query.Limit, Offset: c.query.Offset})
		fmt.Println(res)
		require.Equal(t, c.err, err, c.desc)
		require.Equal(t, len(c.res), len(res), c.desc)
		for i, record := range res {
			require.Equal(t, c.res[i].Status, record.Status)
			require.Equal(t, c.res[i].Type, record.Type)
		}
	}
}

func InitRepo() history.TaskHistoryRepo {
	return postgres.NewHistoryRepo(db)
}
