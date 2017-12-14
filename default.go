package main

import (
	"strings"
	
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	 "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func SetDefaults_MasterConfiguration(obj *kubeadm.MasterConfiguration) {
	if obj.KubernetesVersion == "" {
		obj.KubernetesVersion = v1alpha1.DefaultKubernetesVersion
	}

	if obj.API.BindPort == 0 {
		obj.API.BindPort = v1alpha1.DefaultAPIBindPort
	}

	if obj.Networking.ServiceSubnet == "" {
		obj.Networking.ServiceSubnet = v1alpha1.DefaultServicesSubnet
	}

	if obj.Networking.DNSDomain == "" {
		obj.Networking.DNSDomain = v1alpha1.DefaultServiceDNSDomain
	}

	if len(obj.AuthorizationModes) == 0 {
		obj.AuthorizationModes = strings.Split(v1alpha1.DefaultAuthorizationModes, ",")
	}

	if obj.CertificatesDir == "" {
		obj.CertificatesDir = v1alpha1.DefaultCertificatesDir
	}

	if obj.TokenTTL == nil {
		obj.TokenTTL = &metav1.Duration{
			Duration: constants.DefaultTokenDuration,
		}
	}

	if obj.ImageRepository == "" {
		obj.ImageRepository = v1alpha1.DefaultImageRepository
	}

	if obj.Etcd.DataDir == "" {
		obj.Etcd.DataDir = v1alpha1.DefaultEtcdDataDir
	}
}
