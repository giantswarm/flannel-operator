package etcdstorage

import (
	"testing"

	"github.com/coreos/etcd/clientv3"
	"github.com/giantswarm/microerror"
)

func TestPrefix(t *testing.T) {
	tests := []struct {
		prefix    string
		outPrefix string
		err       error
	}{
		{
			prefix:    "/a",
			outPrefix: "/a",
		},
		{
			prefix:    "b",
			outPrefix: "/b",
		},
		{
			prefix: "/",
			err:    invalidConfigError,
		},
		{
			prefix: "c//",
			err:    invalidConfigError,
		},
	}

	for _, tc := range tests {
		etcdConfig := clientv3.Config{
			Endpoints: []string{"http://127.0.0.1:2379"},
		}
		etcdClient, err := clientv3.New(etcdConfig)
		if err != nil {
			t.Fatalf("expected nil, got %#v prefix=%s", err, tc.prefix)
		}

		config := DefaultConfig()
		config.EtcdClient = etcdClient

		config.Prefix = tc.prefix

		storage, err := New(config)
		if tc.err != microerror.Cause(err) {
			t.Fatalf("expected %#v, got %#v prefix=%s", tc.err, err, tc.prefix)
		}

		if tc.err != nil {
			continue
		}

		if storage.prefix != tc.outPrefix {
			t.Fatalf("expected %s, got %#v prefix=%s", tc.outPrefix, storage.prefix, tc.prefix)
		}
	}
}
