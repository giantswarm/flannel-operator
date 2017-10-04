package etcdstorage

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/microstorage"
)

// Config represents the configuration used to create a etcd service.
type Config struct {
	// Dependencies.
	EtcdClient *clientv3.Client

	// Settings.
	Prefix  string
	Timeout time.Duration
}

// DefaultConfig provides a default configuration to create a new etcd service
// by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		EtcdClient: nil, // Required.

		// Settings.
		Prefix:  "",
		Timeout: 5 * time.Second,
	}
}

// New creates a new configured etcd service.
func New(config Config) (*Service, error) {
	// Dependencies.
	if config.EtcdClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "etcd client must not be empty")
	}
	if config.Prefix != "" {
		p, err := microstorage.SanitizeKey(config.Prefix)
		if err != nil {
			return nil, microerror.Maskf(invalidConfigError, "prefix must be valid: %s", err)
		}
		config.Prefix = p
	}

	newService := &Service{
		// Dependencies.
		etcdClient: config.EtcdClient,

		// Internals.
		keyClient: clientv3.NewKV(config.EtcdClient),

		// Settings.
		prefix:  config.Prefix,
		timeout: config.Timeout,
	}

	return newService, nil
}

// Service is the etcd service.
type Service struct {
	// Dependencies.
	etcdClient *clientv3.Client

	// Internals.
	keyClient clientv3.KV

	// Settings.
	prefix  string
	timeout time.Duration
}

func (s *Service) Put(ctx context.Context, kv microstorage.KV) error {
	key, value := kv.Key(), kv.Val()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.keyClient.Put(ctx, key, value)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, k microstorage.K) error {
	key := k.Key()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.keyClient.Delete(ctx, key)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (s *Service) Exists(ctx context.Context, k microstorage.K) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.Search(ctx, k)
	if microstorage.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (s *Service) List(ctx context.Context, k microstorage.K) ([]microstorage.KV, error) {
	key := k.Key()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	res, err := s.keyClient.Get(ctx, key, opts...)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if res.Count == 0 {
		return nil, microerror.Maskf(microstorage.NotFoundError, key)
	}

	var list []microstorage.KV

	// Special case.
	if key == "/" {

		for _, kv := range res.Kvs {
			// Skip the leading slash '/'.
			list = append(list, microstorage.MustKV(microstorage.NewKV(string(kv.Key)[1:], string(kv.Value))))
		}
		return list, nil
	}

	i := len(key)
	for _, kv := range res.Kvs {
		k := string(kv.Key)

		if len(k) <= i+1 {
			continue
		}

		if k[i] != '/' {
			// We want to ignore all keys that are not separated by slash. When there
			// is a key stored like "foo/bar/baz", listing keys using "foo/ba" should
			// not succeed.
			continue
		}

		list = append(list, microstorage.MustKV(microstorage.NewKV(k[i+1:], string(kv.Value))))
	}

	return list, nil
}

func (s *Service) Search(ctx context.Context, k microstorage.K) (microstorage.KV, error) {
	key := k.Key()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	res, err := s.keyClient.Get(ctx, key)

	if err != nil {
		return microstorage.KV{}, microerror.Mask(err)
	}

	if res.Count == 0 {
		return microstorage.KV{}, microerror.Maskf(microstorage.NotFoundError, key)
	}

	if res.Count > 1 {
		return microstorage.KV{}, microerror.Maskf(multipleValuesError, key)
	}

	return microstorage.MustKV(microstorage.NewKV(key, string(res.Kvs[0].Value))), nil
}
