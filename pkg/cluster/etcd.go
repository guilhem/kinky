package cluster

import (
	"fmt"

	"github.com/barpilot/kinky/pkg/util/etcd"
	"github.com/barpilot/kinky/pkg/util/k8sutil"
	etcdv1beta2 "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	"github.com/coreos/etcd-operator/pkg/util/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *Cluster) etcdname() string {
	return c.cluster.Name + "-etcd"
}

func (c *Cluster) CreateEtcdCluster() (*etcdv1beta2.EtcdCluster, error) {
	if err := etcd.WaitForETCDCRD(c.config.APIExtClient); err != nil {
		return nil, err
	}

	name := c.etcdname()

	etcdCl := &etcdv1beta2.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.cluster.Namespace,
			Labels:    k8sutil.LabelsForCluster(name),
			Annotations: map[string]string{
				constants.AnnotationScope: constants.AnnotationClusterWide,
			},
		},
		Spec: etcdv1beta2.ClusterSpec{
			Size: 1,
		},
	}

	if _, err := c.config.EtcdClient.EtcdV1beta2().EtcdClusters(etcdCl.Namespace).Create(etcdCl); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("unable to create etcd cluster: %v", err)
		}

		cl, err := c.config.EtcdClient.EtcdV1beta2().EtcdClusters(etcdCl.Namespace).Get(etcdCl.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		cl.DeepCopyInto(etcdCl)
		//etcdCl.ObjectMeta.ResourceVersion = cl.ObjectMeta.ResourceVersion

		if _, err := c.config.EtcdClient.EtcdV1beta2().EtcdClusters(etcdCl.Namespace).Update(etcdCl); err != nil {
			return cl, fmt.Errorf("unable to update etcd cluster: %v", err)
		}
	}

	etcd.WaitForEtcdAvailable(c.config.EtcdClient, etcdCl)

	return c.config.EtcdClient.EtcdV1beta2().EtcdClusters(etcdCl.Namespace).Get(name, metav1.GetOptions{})
}

func (c *Cluster) GetEtcdCluster() (*etcdv1beta2.EtcdCluster, error) {
	return c.config.EtcdClient.EtcdV1beta2().EtcdClusters(c.cluster.Namespace).Get(c.etcdname(), metav1.GetOptions{})
}
