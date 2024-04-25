package models

import (
	"context"
)

type RepoSearchPaginator interface {
	HasNext() bool
	Next(ctx context.Context) ([]*Repo, error)
}
