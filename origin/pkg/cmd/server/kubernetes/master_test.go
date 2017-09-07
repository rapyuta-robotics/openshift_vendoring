package kubernetes

import (
	"testing"
	"time"

	"github.com/openshift/github.com/coreos/etcd/client"
	"github.com/openshift/golang.org/x/net/context"
	"github.com/openshift/kubernetes/pkg/api/testapi"
	"github.com/openshift/kubernetes/pkg/registry/generic"
	"github.com/openshift/kubernetes/pkg/storage/etcd/etcdtest"
	etcdtesting "github.com/openshift/kubernetes/pkg/storage/etcd/testing"
	"github.com/openshift/kubernetes/pkg/storage/storagebackend"
)

func TestNewMasterLeasesHasCorrectTTL(t *testing.T) {
	server := etcdtesting.NewUnsecuredEtcdTestClientServer(t)
	etcdStorage := &storagebackend.Config{
		Type:                     "etcd2",
		Prefix:                   etcdtest.PathPrefix(),
		ServerList:               server.Client.Endpoints(),
		DeserializationCacheSize: etcdtest.DeserializationCacheSize,
		Codec: testapi.Groups[""].StorageCodec(),
	}

	restOptions := generic.RESTOptions{StorageConfig: etcdStorage, Decorator: generic.UndecoratedStorage, DeleteCollectionWorkers: 1}
	storageInterface, _ := restOptions.Decorator(restOptions.StorageConfig, 0, nil, "masterleases", nil, nil, nil)
	defer server.Terminate(t)

	masterLeases := newMasterLeases(storageInterface)
	if err := masterLeases.UpdateLease("1.2.3.4"); err != nil {
		t.Fatalf("error updating lease: %v", err)
	}

	etcdClient := server.Client
	keys := client.NewKeysAPI(etcdClient)
	resp, err := keys.Get(context.Background(), etcdtest.PathPrefix()+"/masterleases/1.2.3.4", nil)
	if err != nil {
		t.Fatalf("error getting key: %v", err)
	}
	ttl := resp.Node.TTLDuration()
	if ttl > 15*time.Second || ttl < 10*time.Second {
		t.Errorf("ttl %v should be ~ 15s", ttl)
	}
}
