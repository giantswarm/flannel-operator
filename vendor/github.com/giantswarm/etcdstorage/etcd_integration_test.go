// +build integration

package etcdstorage

import (
	"testing"

	"github.com/coreos/etcd/clientv3"
	"github.com/giantswarm/microstorage/storagetest"
)

func Test_Storage(t *testing.T) {
	etcdConfig := clientv3.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
	}
	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		t.Fatal("expected", nil, "got", err)
	}

	config := DefaultConfig()
	config.EtcdClient = etcdClient

	storage, err := New(config)
	if err != nil {
		t.Fatal("expected", nil, "got", err)
	}

	storagetest.Test(t, storage)
}

func Test_Storage_Prefix(t *testing.T) {
	etcdConfig := clientv3.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
	}
	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		t.Fatal("expected", nil, "got", err)
	}

	config := DefaultConfig()
	config.EtcdClient = etcdClient
	config.Prefix = "test-etcd-prefix"

	storage, err := New(config)
	if err != nil {
		t.Fatal("expected", nil, "got", err)
	}

	storagetest.Test(t, storage)
}
