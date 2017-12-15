package controller

import (
	"fmt"
	"time"

	api "github.com/barpilot/kinky/pkg/apis/kinky/v1alpha1"
	"github.com/barpilot/kinky/pkg/client/clientset/versioned"
	kinkyinformer "github.com/barpilot/kinky/pkg/client/informers/externalversions/kinky/v1alpha1"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var pt *panicTimer

func init() {
	pt = newPanicTimer(time.Minute, "unexpected long blocking (> 1 Minute) when handling cluster event")
}

func (c *Controller) Run(stop <-chan struct{}) {
	pt = newPanicTimer(time.Minute, "unexpected long blocking (> 1 Minute) when handling cluster event")

	ca := c.kinkyInformer.InformerFor(&api.Kinky{}, func(client versioned.Interface, t time.Duration) cache.SharedIndexInformer {
		return kinkyinformer.NewKinkyInformer(
			client,
			metav1.NamespaceAll,
			t,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)
	})

	ca.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAddClus,
		UpdateFunc: c.onUpdateClus,
		DeleteFunc: c.onDeleteClus,
	})

	c.kinkyInformer.Start(stop)
}

func (c *Controller) onAddClus(obj interface{}) {
	glog.Infof("cluster added: %v", obj)
	c.syncClus(obj.(*api.Kinky))
}

func (c *Controller) onUpdateClus(oldObj, newObj interface{}) {
	glog.Infof("cluster updated: %v", newObj)
	c.syncClus(newObj.(*api.Kinky))
}

func (c *Controller) onDeleteClus(obj interface{}) {
	glog.Infof("cluster deleted: %v", obj)

	clus, ok := obj.(*api.Kinky)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			panic(fmt.Sprintf("unknown object from EtcdCluster delete event: %#v", obj))
		}
		clus, ok = tombstone.Obj.(*api.Kinky)
		if !ok {
			panic(fmt.Sprintf("Tombstone contained object that is not an EtcdCluster: %#v", obj))
		}
	}
	ev := &Event{
		Type:   kwatch.Deleted,
		Object: clus,
	}

	pt.start()
	err := c.handleClusterEvent(ev)
	if err != nil {
		glog.Warningf("fail to handle event: %v", err)
	}
	pt.stop()
}

func (c *Controller) syncClus(clus *api.Kinky) {
	ev := &Event{
		Type:   kwatch.Added,
		Object: clus,
	}
	// re-watch or restart could give ADD event.
	// If for an ADD event the cluster spec is invalid then it is not added to the local cache
	// so modifying that cluster will result in another ADD event
	if _, ok := c.clusters[clus.Name]; ok {
		ev.Type = kwatch.Modified
	}

	pt.start()
	err := c.handleClusterEvent(ev)
	if err != nil {
		glog.Warningf("fail to handle event: %v", err)
	}
	pt.stop()
}
