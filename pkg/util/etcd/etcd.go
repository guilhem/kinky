package etcd

import (
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	etcdv1beta2 "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etcdclientset "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

func WaitForETCDCRD(apiExtClient apiextensionsclientset.Interface) error {
	return wait.Poll(5*time.Second, 30*time.Minute, func() (bool, error) {
		_, err := apiExtClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(etcdv1beta2.EtcdClusterCRDName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
}

func WaitForEtcdAvailable(client etcdclientset.Interface, cluster *etcdv1beta2.EtcdCluster) error {
	return wait.Poll(5*time.Second, 30*time.Minute, func() (bool, error) {
		cl, err := client.EtcdV1beta2().EtcdClusters(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if cl.Status.Phase != etcdv1beta2.ClusterPhaseRunning {
			return false, nil
		}
		return true, nil
	})
}
