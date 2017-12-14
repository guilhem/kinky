package main

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strconv"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"

	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs/pkiutil"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	certutil "k8s.io/client-go/util/cert"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	certsphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
)

type certsWallet struct {
	CaCert                   *x509.Certificate
	CaKey                    *rsa.PrivateKey
	ApiCert                  *x509.Certificate
	ApiKey                   *rsa.PrivateKey
	ApiClientCert            *x509.Certificate
	ApiClientKey             *rsa.PrivateKey
	ServiceAccountPrivateKey *rsa.PrivateKey
	FrontProxyCACert         *x509.Certificate
	FrontProxyCAKey          *rsa.PrivateKey
	FrontProxyClientCert     *x509.Certificate
	FrontProxyClientKey      *rsa.PrivateKey
}

func certsPhase(k8sClient *kubernetes.Clientset, cfg *kubeadmapi.MasterConfiguration, ns string, ips []net.IP, hostname string) error {
	if !certificatesSecretExists(k8sClient, ns) {
		wallet, err := createCerts(ips, hostname)
		if err != nil {
			return err
		}
		if err := createCertificatesSecret(k8sClient, ns, wallet); err != nil {
			return err
		}
		if err := createKubeconfigSecret(k8sClient, cfg, ns, wallet); err != nil {
			return err
		}
	}
	return nil
}

func createCerts(ips []net.IP, hostname string) (*certsWallet, error) {
	wallet := certsWallet{}

	var err error

	wallet.CaCert, wallet.CaKey, err = certsphase.NewCACertAndKey()
	if err != nil {
		return nil, err
	}

	altNames := &certutil.AltNames{
		DNSNames: []string{
			"Default",
			"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			fmt.Sprintf("kubernetes.default.svc.%s", "apiserver"),
			hostname,
		},
		IPs: ips,
	}

	config := certutil.Config{
		CommonName: kubeadmconstants.APIServerCertCommonName,
		AltNames:   *altNames,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	wallet.ApiCert, wallet.ApiKey, err = pkiutil.NewCertAndKey(wallet.CaCert, wallet.CaKey, config)
	if err != nil {
		return nil, err
	}

	config = certutil.Config{
		CommonName:   kubeadmconstants.APIServerKubeletClientCertCommonName,
		Organization: []string{kubeadmconstants.MastersGroup},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	wallet.ApiClientCert, wallet.ApiClientKey, err = pkiutil.NewCertAndKey(wallet.CaCert, wallet.CaKey, config)
	if err != nil {
		return nil, err
	}

	wallet.ServiceAccountPrivateKey, err = certutil.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	wallet.FrontProxyCACert, wallet.FrontProxyCAKey, err = pkiutil.NewCertificateAuthority()
	if err != nil {
		return nil, err
	}

	config = certutil.Config{
		CommonName: kubeadmconstants.FrontProxyClientCertCommonName,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	wallet.FrontProxyClientCert, wallet.FrontProxyClientKey, err = pkiutil.NewCertAndKey(wallet.FrontProxyCACert, wallet.FrontProxyCAKey, config)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("\nfrontProxyClientCert: %v, %v\n", frontProxyClientCert, frontProxyClientKey)
	// // PHASE 1: Generate certificates
	// if err := certsphase.CreatePKIAssets(i.cfg); err != nil {
	// 	return err
	// }
	//
	// // PHASE 2: Generate kubeconfig files for the admin and the kubelet
	// if err := kubeconfigphase.CreateInitKubeConfigFiles(kubeConfigDir, i.cfg); err != nil {
	// 	return err
	// }

	return &wallet, nil
}

func certificatesSecretExists(k8sClient *kubernetes.Clientset, ns string) bool {
	_, err := k8sClient.CoreV1().Secrets(ns).Get(kubeadmconstants.KubeCertificatesVolumeName, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

func createCertificatesSecret(k8sClient *kubernetes.Clientset, ns string, wallet *certsWallet) error {

	serviceAccountPublicKey, err := certutil.EncodePublicKeyPEM(&wallet.ServiceAccountPrivateKey.PublicKey)
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      kubeadmconstants.KubeCertificatesVolumeName,
		},
		Data: map[string][]byte{
			kubeadmconstants.CACertName:                     certutil.EncodeCertPEM(wallet.CaCert),
			kubeadmconstants.CAKeyName:                      certutil.EncodePrivateKeyPEM(wallet.CaKey),
			kubeadmconstants.APIServerCertName:              certutil.EncodeCertPEM(wallet.ApiCert),
			kubeadmconstants.APIServerKeyName:               certutil.EncodePrivateKeyPEM(wallet.ApiKey),
			kubeadmconstants.APIServerKubeletClientCertName: certutil.EncodeCertPEM(wallet.ApiClientCert),
			kubeadmconstants.APIServerKubeletClientKeyName:  certutil.EncodePrivateKeyPEM(wallet.ApiClientKey),
			kubeadmconstants.ServiceAccountPublicKeyName:    serviceAccountPublicKey,
			kubeadmconstants.ServiceAccountPrivateKeyName:   certutil.EncodePrivateKeyPEM(wallet.ServiceAccountPrivateKey),
			kubeadmconstants.FrontProxyCAKeyName:            certutil.EncodePrivateKeyPEM(wallet.FrontProxyCAKey),
			kubeadmconstants.FrontProxyCACertName:           certutil.EncodeCertPEM(wallet.FrontProxyCACert),
			kubeadmconstants.FrontProxyClientKeyName:        certutil.EncodePrivateKeyPEM(wallet.FrontProxyClientKey),
			kubeadmconstants.FrontProxyClientCertName:       certutil.EncodeCertPEM(wallet.FrontProxyClientCert),
		},
	}

	if err := apiclient.CreateOrUpdateSecret(k8sClient, secret); err != nil {
		return err
	}
	return nil
}

func kubeconfigSecretExists(k8sClient *kubernetes.Clientset, ns string) bool {
	_, err := k8sClient.CoreV1().Secrets(ns).Get(kubeconfigSecret, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

func createKubeconfigSecret(k8sClient *kubernetes.Clientset, cfg *kubeadmapi.MasterConfiguration, ns string, wallet *certsWallet) error {

	kubeConfigs, err := createKubeConfigFiles(cfg, wallet.CaCert, wallet.CaKey)
	if err != nil {
		return err
	}

	schedulerConfig, ok := kubeConfigs[kubeadmconstants.SchedulerKubeConfigFileName]
	if !ok {
		return errors.New("No Scheduler Kubeconfig found")
	}
	controllerConfig, ok := kubeConfigs[kubeadmconstants.ControllerManagerKubeConfigFileName]
	if !ok {
		return errors.New("No Controller Kubeconfig found")
	}

	// TODO own secret
	adminConfig, ok := kubeConfigs[kubeadmconstants.AdminKubeConfigFileName]
	if !ok {
		return errors.New("No Admin Kubeconfig found")
	}

	schedulerFile, err := clientcmd.Write(*schedulerConfig)
	if err != nil {
		return err
	}
	controllerFile, err := clientcmd.Write(*controllerConfig)
	if err != nil {
		return err
	}
	adminFile, err := clientcmd.Write(*adminConfig)
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      kubeconfigSecret,
		},
		Data: map[string][]byte{
			kubeadmconstants.SchedulerKubeConfigFileName:         schedulerFile,
			kubeadmconstants.ControllerManagerKubeConfigFileName: controllerFile,
			kubeadmconstants.AdminKubeConfigFileName:             adminFile,
		},
	}

	if err := apiclient.CreateOrUpdateSecret(k8sClient, secret); err != nil {
		return err
	}

	return nil
}

func getSecretString(secret *v1.Secret, key string) string {
	if secret.Data == nil {
		return ""
	}
	if val, ok := secret.Data[key]; ok {
		return string(val)
	}
	return ""
}

func createKubeConfigFiles(cfg *kubeadmapi.MasterConfiguration, caCert *x509.Certificate, caKey *rsa.PrivateKey) (map[string]*clientcmdapi.Config, error) {
	configs := make(map[string]*clientcmdapi.Config)
	// gets the KubeConfigSpecs, actualized for the current MasterConfiguration
	specs, err := getKubeConfigSpecs(cfg, caCert, caKey)
	if err != nil {
		return configs, err
	}

	for key, spec := range specs {
		// builds the KubeConfig object
		config, err := buildKubeConfigFromSpec(spec)
		if err != nil {
			return configs, err
		}
		configs[key] = config
	}

	return configs, nil
}

/// Copy of kubeadm code

// clientCertAuth struct holds info required to build a client certificate to provide authentication info in a kubeconfig object
type clientCertAuth struct {
	CAKey         *rsa.PrivateKey
	Organizations []string
}

// tokenAuth struct holds info required to use a token to provide authentication info in a kubeconfig object
type tokenAuth struct {
	Token string
}

// kubeConfigSpec struct holds info required to build a KubeConfig object
type kubeConfigSpec struct {
	CACert         *x509.Certificate
	APIServer      string
	ClientName     string
	TokenAuth      *tokenAuth
	ClientCertAuth *clientCertAuth
}

// buildKubeConfigFromSpec creates a kubeconfig object for the given kubeConfigSpec
func buildKubeConfigFromSpec(spec *kubeConfigSpec) (*clientcmdapi.Config, error) {

	// If this kubeconfig should use token
	if spec.TokenAuth != nil {
		// create a kubeconfig with a token
		return kubeconfigutil.CreateWithToken(
			spec.APIServer,
			"kubernetes",
			spec.ClientName,
			certutil.EncodeCertPEM(spec.CACert),
			spec.TokenAuth.Token,
		), nil
	}

	// otherwise, create a client certs
	clientCertConfig := certutil.Config{
		CommonName:   spec.ClientName,
		Organization: spec.ClientCertAuth.Organizations,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientCert, clientKey, err := pkiutil.NewCertAndKey(spec.CACert, spec.ClientCertAuth.CAKey, clientCertConfig)
	if err != nil {
		return nil, fmt.Errorf("failure while creating %s client certificate: %v", spec.ClientName, err)
	}

	// create a kubeconfig with the client certs
	return kubeconfigutil.CreateWithCerts(
		spec.APIServer,
		"kubernetes",
		spec.ClientName,
		certutil.EncodeCertPEM(spec.CACert),
		certutil.EncodePrivateKeyPEM(clientKey),
		certutil.EncodeCertPEM(clientCert),
	), nil
}

// getKubeConfigSpecs returns all KubeConfigSpecs actualized to the context of the current MasterConfiguration
// NB. this methods holds the information about how kubeadm creates kubeconfig files.
func getKubeConfigSpecs(cfg *kubeadmapi.MasterConfiguration, caCert *x509.Certificate, caKey *rsa.PrivateKey) (map[string]*kubeConfigSpec, error) {

	masterEndpoint := "https://" + net.JoinHostPort(cfg.API.AdvertiseAddress, strconv.Itoa(int(cfg.API.BindPort)))

	var kubeConfigSpec = map[string]*kubeConfigSpec{
		kubeadmconstants.AdminKubeConfigFileName: {
			CACert:     caCert,
			APIServer:  masterEndpoint,
			ClientName: "kubernetes-admin",
			ClientCertAuth: &clientCertAuth{
				CAKey:         caKey,
				Organizations: []string{kubeadmconstants.MastersGroup},
			},
		},
		kubeadmconstants.KubeletKubeConfigFileName: {
			CACert:     caCert,
			APIServer:  masterEndpoint,
			ClientName: fmt.Sprintf("system:node:%s", cfg.NodeName),
			ClientCertAuth: &clientCertAuth{
				CAKey:         caKey,
				Organizations: []string{kubeadmconstants.NodesGroup},
			},
		},
		kubeadmconstants.ControllerManagerKubeConfigFileName: {
			CACert:     caCert,
			APIServer:  masterEndpoint,
			ClientName: kubeadmconstants.ControllerManagerUser,
			ClientCertAuth: &clientCertAuth{
				CAKey: caKey,
			},
		},
		kubeadmconstants.SchedulerKubeConfigFileName: {
			CACert:     caCert,
			APIServer:  masterEndpoint,
			ClientName: kubeadmconstants.SchedulerUser,
			ClientCertAuth: &clientCertAuth{
				CAKey: caKey,
			},
		},
	}

	return kubeConfigSpec, nil
}
