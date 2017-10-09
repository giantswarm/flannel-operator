package etcdv2

import (
	"context"
)

type Store interface {
	Create(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, key string) ([]string, error)
	Search(ctx context.Context, key string) (string, error)
}
