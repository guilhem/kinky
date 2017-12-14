package main

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"

	etcdv1beta2 "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	etcdclientset "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

func createEtcdOperator(client *kubernetes.Clientset, ns string) error {

	deployment := &extv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd-operator",
			Namespace: ns,
		},
		Spec: extv1beta1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": "etcd-operator",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "etcd-operator",
							Image:   "quay.io/coreos/etcd-operator:v0.7.0",
							Command: []string{"etcd-operator"},
							Env: []apiv1.EnvVar{
								{
									Name: "MY_POD_NAMESPACE",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "MY_POD_NAME",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createEtcdCluster(client *etcdclientset.Clientset, apiExtClient *apiextensionsclientset.Clientset, name string, ns string) (*etcdv1beta2.EtcdCluster, error) {
	if err := waitForETCDCRD(apiExtClient); err != nil {
		return nil, err
	}

	etcdCl := &etcdv1beta2.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"captaincy": "kinky",
			},
		},
		Spec: etcdv1beta2.ClusterSpec{
			Size: 1,
		},
	}

	if _, err := client.EtcdV1beta2().EtcdClusters(etcdCl.Namespace).Create(etcdCl); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("unable to create etcd cluster: %v", err)
		}

		cl, err := client.EtcdV1beta2().EtcdClusters(etcdCl.Namespace).Get(etcdCl.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		cl.DeepCopyInto(etcdCl)
		//etcdCl.ObjectMeta.ResourceVersion = cl.ObjectMeta.ResourceVersion

		if _, err := client.EtcdV1beta2().EtcdClusters(etcdCl.Namespace).Update(etcdCl); err != nil {
			return cl, fmt.Errorf("unable to update etcd cluster: %v", err)
		}
	}

	waitForEtcdAvailable(client, etcdCl)

	return client.EtcdV1beta2().EtcdClusters(ns).Get(name, metav1.GetOptions{})
}

func waitForETCDCRD(apiExtClient *apiextensionsclientset.Clientset) error {
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

func waitForEtcdAvailable(client *etcdclientset.Clientset, cluster *etcdv1beta2.EtcdCluster) error {
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
