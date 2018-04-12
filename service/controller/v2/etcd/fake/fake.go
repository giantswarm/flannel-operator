package fake

import (
	"context"
)

type Fake struct {
}

func New() *Fake {
	return &Fake{}
}

func (s *Fake) Create(ctx context.Context, key, value string) error {
	return nil
}

func (s *Fake) Delete(ctx context.Context, key string) error {
	return nil
}

func (s *Fake) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (s *Fake) List(ctx context.Context, key string) ([]string, error) {
	return nil, nil
}

func (s *Fake) Search(ctx context.Context, key string) (string, error) {
	return "", nil
}
