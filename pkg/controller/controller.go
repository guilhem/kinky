package controller

import (
	"fmt"

	api "github.com/barpilot/kinky/pkg/apis/kinky/v1alpha1"
	"github.com/barpilot/kinky/pkg/client/clientset/versioned"
	kinkycluster "github.com/barpilot/kinky/pkg/cluster"

	informers "github.com/barpilot/kinky/pkg/client/informers/externalversions"
	etcdclientset "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	kwatch "k8s.io/apimachinery/pkg/watch"
)

type Event struct {
	Type   kwatch.EventType
	Object *api.Kinky
}

type Controller struct {
	Config

	clusters map[string]*kinkycluster.Cluster

	kinkyInformer informers.SharedInformerFactory
}

type Config struct {
	// Namespace      string
	// ClusterWide    bool
	BaseHost string

	ServiceAccount string

	KinkyClient  versioned.Interface
	K8sClient    kubernetes.Interface
	EtcdClient   etcdclientset.Interface
	APIExtClient apiextensionsclientset.Interface

	CreateCRD bool
}

func New(cfg Config) *Controller {
	return &Controller{
		Config:   cfg,
		clusters: make(map[string]*kinkycluster.Cluster),

		kinkyInformer: informers.NewSharedInformerFactory(cfg.KinkyClient, 0),
	}
}

func (c *Controller) handleClusterEvent(event *Event) error {
	clus := event.Object

	if clus.Status.IsFailed() {
		clustersFailed.Inc()
		if event.Type == kwatch.Deleted {
			delete(c.clusters, clus.Name)
			return nil
		}
		return fmt.Errorf("ignore failed cluster (%s). Please delete its CR", clus.Name)
	}

	clus.SetDefaults()

	if err := clus.Spec.Validate(); err != nil {
		return fmt.Errorf("invalid cluster spec. please fix the following problem with the cluster spec: %v", err)
	}

	switch event.Type {
	case kwatch.Added:
		if _, ok := c.clusters[clus.Name]; ok {
			return fmt.Errorf("unsafe state. cluster (%s) was created before but we received event (%s)", clus.Name, event.Type)
		}

		nc, err := kinkycluster.New(c.makeClusterConfig(), clus)
		if err != nil {
			return fmt.Errorf("error in creating cluster: %v", err)
		}

		c.clusters[clus.Name] = nc

		clustersCreated.Inc()
		clustersTotal.Inc()

	case kwatch.Modified:
		if _, ok := c.clusters[clus.Name]; !ok {
			return fmt.Errorf("unsafe state. cluster (%s) was never created but we received event (%s)", clus.Name, event.Type)
		}
		c.clusters[clus.Name].Update(clus)
		clustersModified.Inc()

	case kwatch.Deleted:
		if _, ok := c.clusters[clus.Name]; !ok {
			return fmt.Errorf("unsafe state. cluster (%s) was never created but we received event (%s)", clus.Name, event.Type)
		}
		c.clusters[clus.Name].Delete()
		delete(c.clusters, clus.Name)
		clustersDeleted.Inc()
		clustersTotal.Dec()
	}
	return nil
}

func (c *Controller) makeClusterConfig() kinkycluster.Config {
	return kinkycluster.Config{
		BaseHost:     c.Config.BaseHost,
		KinkyClient:  c.Config.KinkyClient,
		K8sClient:    c.Config.K8sClient,
		EtcdClient:   c.Config.EtcdClient,
		APIExtClient: c.Config.APIExtClient,
	}
}
