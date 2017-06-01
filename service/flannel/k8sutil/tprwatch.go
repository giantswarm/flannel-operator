package k8sutil

import (
	"encoding/json"
	"io"

	"github.com/giantswarm/operatorkit/tpr"

	microerror "github.com/giantswarm/microkit/error"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// ZeroObjectFuncs provides zero values of an object and objects' list ready to
// be decoded. The provided zero values must not be reused by zeroObjectFactory.
type ZeroObjectFactory interface {
	NewObject() runtime.Object
	NewObjectList() runtime.Object
}

// ZeroObjectFactoryFuncs implements ZeroObjectFactory.
type ZeroObjectFactoryFuncs struct {
	NewObjectFunc     func() runtime.Object
	NewObjectListFunc func() runtime.Object
}

func (z ZeroObjectFactoryFuncs) NewObject() runtime.Object     { return z.NewObjectFunc() }
func (z ZeroObjectFactoryFuncs) NewObjectList() runtime.Object { return z.NewObjectListFunc() }

// Observer functions are called when the cache.ListWatch List and Watch
// functions are called.
type Observer interface {
	OnList()
	OnWatch()
}

// ObserverFuncs implements Observer interface.
type ObserverFuncs struct {
	OnListFunc  func()
	OnWatchFunc func()
}

func (o ObserverFuncs) OnList()  { o.OnList() }
func (o ObserverFuncs) OnWatch() { o.OnWatch() }

// NewInformer returns a configured informer for the TPR.
func NewInformer(
	k8sClient kubernetes.Interface,
	t *tpr.TPR,
	zeroObjectFactory ZeroObjectFactory,
	observer Observer,
	handler cache.ResourceEventHandler,
) (informer *cache.Controller) {
	listWatch := &cache.ListWatch{
		ListFunc: func(options api.ListOptions) (runtime.Object, error) {
			observer.OnList()

			req := k8sClient.Core().RESTClient().Get().AbsPath(t.Endpoint(""))
			b, err := req.DoRaw()
			if err != nil {
				return nil, microerror.MaskAny(err)
			}

			v := zeroObjectFactory.NewObjectList()
			if err := json.Unmarshal(b, v); err != nil {
				return nil, microerror.MaskAny(err)
			}

			return v, nil
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			observer.OnWatch()

			req := k8sClient.CoreV1().RESTClient().Get().AbsPath(t.WatchEndpoint(""))
			stream, err := req.Stream()
			if err != nil {
				return nil, microerror.MaskAny(err)
			}

			watcher := watch.NewStreamWatcher(&decoder{
				stream: stream,
				obj:    zeroObjectFactory,
			})
			return watcher, nil
		},
	}

	_, informer = cache.NewInformer(listWatch, zeroObjectFactory.NewObject(), tpr.ResyncPeriod, handler)
	return informer
}

type decoder struct {
	stream io.ReadCloser
	obj    ZeroObjectFactory
}

func (d *decoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	dec := json.NewDecoder(d.stream)
	var e struct {
		Type   watch.EventType
		Object runtime.Object
	}
	e.Object = d.obj.NewObject()
	if err := dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, e.Object, nil
}

func (d *decoder) Close() {
	d.stream.Close()
}
