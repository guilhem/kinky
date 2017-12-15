package cluster

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"reflect"
	"strings"
	"time"

	kinkyv1alpha1 "github.com/barpilot/kinky/pkg/apis/kinky/v1alpha1"
	kinkyclientset "github.com/barpilot/kinky/pkg/client/clientset/versioned"
	"github.com/barpilot/kinky/pkg/cluster/certs"
	"github.com/barpilot/kinky/pkg/cluster/ingress"
	etcdclientset "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned"
	"github.com/coreos/etcd-operator/pkg/util/k8sutil"
	"github.com/coreos/etcd-operator/pkg/util/retryutil"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	masterconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
)

var (
	reconcileInterval         = 8 * time.Second
	podTerminationGracePeriod = int64(5)
)

const (
	eventDeleteCluster clusterEventType = "Delete"
	eventModifyCluster clusterEventType = "Modify"
)

type clusterEventType string

type clusterEvent struct {
	typ     clusterEventType
	cluster *kinkyv1alpha1.Kinky
}

type Cluster struct {
	cluster *kinkyv1alpha1.Kinky

	config Config

	status kinkyv1alpha1.ClusterStatus

	eventCh chan *clusterEvent
	stopCh  chan struct{}
}

type Config struct {
	BaseHost string

	KinkyClient  kinkyclientset.Interface
	K8sClient    kubernetes.Interface
	EtcdClient   etcdclientset.Interface
	APIExtClient apiextensionsclientset.Interface
}

func New(config Config, cl *kinkyv1alpha1.Kinky) (*Cluster, error) {
	c := &Cluster{
		cluster: cl,
		config:  config,
		status:  *(cl.Status.DeepCopy()),

		eventCh: make(chan *clusterEvent, 100),
		stopCh:  make(chan struct{}),
	}

	go func() {
		if err := c.setup(); err != nil {
			glog.Errorf("cluster failed to setup: %v", err)
			if c.status.Phase != kinkyv1alpha1.ClusterPhaseFailed {
				c.status.SetReason(err.Error())
				c.status.SetPhase(kinkyv1alpha1.ClusterPhaseFailed)
				if err := c.updateCRStatus(); err != nil {
					glog.Errorf("failed to update cluster phase (%v): %v", kinkyv1alpha1.ClusterPhaseFailed, err)
				}
			}
			return
		}
		c.run()
	}()

	return c, nil
}

func (c *Cluster) setup() error {
	var shouldCreateCluster bool
	switch c.status.Phase {
	case kinkyv1alpha1.ClusterPhaseNone:
		shouldCreateCluster = true
	case kinkyv1alpha1.ClusterPhaseCreating:
		return errCreatedCluster
	case kinkyv1alpha1.ClusterPhaseRunning:
		shouldCreateCluster = false

	default:
		return fmt.Errorf("unexpected cluster phase: %s", c.status.Phase)
	}

	if shouldCreateCluster {
		return c.create()
	}
	return nil
}

func (c *Cluster) create() error {
	etcdCluster, err := c.CreateEtcdCluster()
	if err != nil {
		glog.Errorf("Error spawning ETCD cluster: %v", err)
		return err
	}

	apiServiceName := c.cluster.Name + "-kube-apiserver"

	labels := k8sutil.LabelsForCluster(c.cluster.Name)
	labels["component"] = "kube-apiserver"

	apiService := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServiceName,
			Namespace: c.cluster.Namespace,
			Labels:    labels,
		},
		Spec: apiv1.ServiceSpec{
			Selector: labels,
			Ports: []apiv1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.Parse("443"),
					Protocol:   "TCP",
				},
			},
		},
	}

	c.config.K8sClient.CoreV1().Services(apiService.ObjectMeta.Namespace).Create(apiService)
	// Wait for API service to have an IP
	if err := wait.Poll(5*time.Second, 30*time.Minute, func() (bool, error) {
		svc, err := c.config.K8sClient.CoreV1().Services(c.cluster.Namespace).Get(apiServiceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if svc.Spec.ClusterIP == "" {
			return false, nil
		}
		return true, nil

	}); err != nil {
		glog.Errorf("error while checking pod status: %v", err)
		return err
	}
	svc, err := c.config.K8sClient.CoreV1().Services(c.cluster.Namespace).Get(apiServiceName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("Fail service: %v", err)
		return err
	}

	kubeadmCfg := &kubeadm.MasterConfiguration{
		Etcd: kubeadm.Etcd{
			Endpoints: []string{fmt.Sprintf("http://%s:%d", etcdCluster.Status.ServiceName, etcdCluster.Status.ClientPort)},
		},
		CertificatesDir: "/pki",
		API: kubeadm.API{
			BindPort:         443,
			AdvertiseAddress: "0.0.0.0",
		},
		ControllerManagerExtraArgs: map[string]string{"address": "0.0.0.0"},
		SchedulerExtraArgs:         map[string]string{"address": "0.0.0.0"},
	}
	if c.cluster.Spec.Version != "" {
		kubeadmCfg.KubernetesVersion = c.cluster.Spec.Version
	}

	SetDefaults_MasterConfiguration(kubeadmCfg)
	if err := masterconfig.SetInitDynamicDefaults(kubeadmCfg); err != nil {
		glog.Errorf("Set Init Dynamic defaults configs fail: %v", err)
	}

	internalAPIIP := svc.Spec.ClusterIP
	internalKubeadmCfg := kubeadmCfg.DeepCopy()
	internalKubeadmCfg.API.AdvertiseAddress = internalAPIIP

	clusterHostname := fmt.Sprintf("%s.%s.%s", c.cluster.Name, c.cluster.Namespace, c.config.BaseHost)

	if err := certs.CreateCerts(c.config.K8sClient, internalKubeadmCfg, c.cluster.Namespace, []net.IP{net.ParseIP(internalAPIIP)}, clusterHostname); err != nil {
		glog.Errorf("Create certificates and configs fail: %v", err)
		return err
	}

	deployments, err := c.GetControleplaneDeployments(kubeadmCfg)
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		if err := apiclient.CreateOrUpdateDeployment(c.config.K8sClient, deployment); err != nil {
			glog.Errorf("Pod deployment fail: %v", err)
			return err
		}
	}

	if err := ingress.CreateIngress(c.config.K8sClient, "ingress-"+c.cluster.Name, c.cluster.Namespace, clusterHostname, apiServiceName); err != nil {
		glog.Errorf("could not create ingress: %v", err)
		return err
	}
	// tokenDescription := "The default bootstrap token generated."
	// if err := nodebootstraptokenphase.UpdateOrCreateToken(c.config.K8sClient, kubeadmCfg.Token, false, kubeadmCfg.TokenTTL.Duration, kubeadmconstants.DefaultTokenUsages, []string{kubeadmconstants.V18NodeBootstrapTokenAuthGroup}, tokenDescription); err != nil {
	// 	glog.Errorf("Creation default bootstrap: %v", err)
	// 	return err
	// }

	c.status.SetPhase(kinkyv1alpha1.ClusterPhaseRunning)
	if err := c.updateCRStatus(); err != nil {
		glog.Errorf("failed to update cluster phase: %v", err)
	}
	return nil
}

func (c *Cluster) updateCRStatus() error {
	if reflect.DeepEqual(c.cluster.Status, c.status) {
		return nil
	}

	newCluster := c.cluster
	newCluster.Status = c.status
	newCluster, err := c.config.KinkyClient.KinkyV1alpha1().Kinkies(c.cluster.Namespace).Update(c.cluster)
	if err != nil {
		return fmt.Errorf("failed to update CR status: %v", err)
	}

	c.cluster = newCluster

	return nil
}

func (c *Cluster) Delete() {
	glog.Infof("cluster %s is deleted by user", c.cluster.Name)
	if deployments, err := c.runningDeployments(); err == nil {
		glog.Infof("delete deployments: %v", deployments)
		for _, deploy := range deployments {
			c.config.K8sClient.ExtensionsV1beta1().Deployments(c.cluster.Namespace).Delete(deploy.Name, &metav1.DeleteOptions{})
		}
	}
	if etcd, err := c.GetEtcdCluster(); err == nil {
		glog.Infof("delete etcd cluster: %v", etcd)
		c.config.EtcdClient.EtcdV1beta2().EtcdClusters(etcd.Namespace).Delete(etcd.Name, &metav1.DeleteOptions{})
	}
	close(c.stopCh)
}

func (c *Cluster) Update(cl *kinkyv1alpha1.Kinky) {
	c.send(&clusterEvent{
		typ:     eventModifyCluster,
		cluster: cl,
	})
}

func (c *Cluster) send(ev *clusterEvent) {
	select {
	case c.eventCh <- ev:
		l, ecap := len(c.eventCh), cap(c.eventCh)
		if l > int(float64(ecap)*0.8) {
			glog.Warningf("eventCh buffer is almost full [%d/%d]", l, ecap)
		}
	case <-c.stopCh:
	}
}

func (c *Cluster) run() {
	// if err := c.setupServices(); err != nil {
	// 	c.logger.Errorf("fail to setup etcd services: %v", err)
	// }
	// c.status.ServiceName = k8sutil.ClientServiceName(c.cluster.Name)
	// c.status.ClientPort = k8sutil.EtcdClientPort

	// defer func() {
	// 	glog.Infof("deleting the failed cluster")
	// 	c.reportFailedStatus()
	// 	c.Delete()
	// }()

	c.status.SetPhase(kinkyv1alpha1.ClusterPhaseRunning)
	if err := c.updateCRStatus(); err != nil {
		glog.Warningf("update initial CR status failed: %v", err)
	}
	glog.Infof("start running...")

	var rerr error
	for {
		select {
		case <-c.stopCh:
			return
		case event := <-c.eventCh:
			switch event.typ {
			case eventModifyCluster:
				err := c.handleUpdateEvent(event)
				if err != nil {
					glog.Errorf("handle update event failed: %v", err)
					c.status.SetReason(err.Error())
					c.reportFailedStatus()
					return
				}
			default:
				panic("unknown event type" + event.typ)
			}

		case <-time.After(reconcileInterval):
			start := time.Now()
			if c.cluster.Spec.Paused {
				c.status.PauseControl()
				glog.Infof("control is paused, skipping reconciliation")
				continue
			} else {
				c.status.Control()
			}

			// running, pending, err := c.pollPods()
			// if err != nil {
			// 	glog.Errorf("fail to poll pods: %v", err)
			// 	reconcileFailed.WithLabelValues("failed to poll pods").Inc()
			// 	continue
			// }
			//
			// if len(pending) > 0 {
			// 	// Pod startup might take long, e.g. pulling image. It would deterministically become running or succeeded/failed later.
			// 	glog.Infof("skip reconciliation: running (%v), pending (%v)", k8sutil.GetPodNames(running), k8sutil.GetPodNames(pending))
			// 	reconcileFailed.WithLabelValues("not all pods are running").Inc()
			// 	continue
			// }
			// if len(running) == 0 {
			// 	glog.Warningf("all etcd pods are dead. Trying to recover from a previous backup")
			// 	rerr = c.disasterRecovery(nil)
			// 	if rerr != nil {
			// 		glog.Errorf("fail to do disaster recovery: %v", rerr)
			// 	}
			// 	// On normal recovery case, we need backoff. On error case, this could be either backoff or leading to cluster delete.
			// 	break
			// }

			// rerr = c.reconcile(running)
			// if rerr != nil {
			// 	glog.Errorf("failed to reconcile: %v", rerr)
			// 	break
			// }

			if err := c.updateCRStatus(); err != nil {
				glog.Warningf("periodic update CR status failed: %v", err)
			}

			reconcileHistogram.WithLabelValues(c.name()).Observe(time.Since(start).Seconds())
		}

		if rerr != nil {
			reconcileFailed.WithLabelValues(rerr.Error()).Inc()
		}

		if isFatalError(rerr) {
			c.status.SetReason(rerr.Error())
			glog.Errorf("cluster failed: %v", rerr)
			return
		}
	}
}

func (c *Cluster) reportFailedStatus() {
	retryInterval := 5 * time.Second

	f := func() (bool, error) {
		c.status.SetPhase(kinkyv1alpha1.ClusterPhaseFailed)
		err := c.updateCRStatus()
		if err == nil || k8sutil.IsKubernetesResourceNotFoundError(err) {
			return true, nil
		}

		if !apierrors.IsConflict(err) {
			glog.Warningf("retry report status in %v: fail to update: %v", retryInterval, err)
			return false, nil
		}

		cl, err := c.config.KinkyClient.KinkyV1alpha1().Kinkies(c.cluster.Namespace).Get(c.cluster.Name, metav1.GetOptions{})
		if err != nil {
			// Update (PUT) will return conflict even if object is deleted since we have UID set in object.
			// Because it will check UID first and return something like:
			// "Precondition failed: UID in precondition: 0xc42712c0f0, UID in object meta: ".
			if k8sutil.IsKubernetesResourceNotFoundError(err) {
				return true, nil
			}
			glog.Warningf("retry report status in %v: fail to get latest version: %v", retryInterval, err)
			return false, nil
		}
		c.cluster = cl
		return false, nil

	}

	retryutil.Retry(retryInterval, math.MaxInt64, f)
}

func (c *Cluster) handleUpdateEvent(event *clusterEvent) error {
	oldSpec := c.cluster.Spec.DeepCopy()
	c.cluster = event.cluster

	if isSpecEqual(event.cluster.Spec, *oldSpec) {
		// We have some fields that once created could not be mutated.
		if !reflect.DeepEqual(event.cluster.Spec, *oldSpec) {
			glog.Infof("ignoring update event: %#v", event.cluster.Spec)
		}
		return nil
	}
	// TODO: we can't handle another upgrade while an upgrade is in progress

	c.logSpecUpdate(*oldSpec, event.cluster.Spec)

	return nil
}

func isSpecEqual(s1, s2 kinkyv1alpha1.KinkySpec) bool {
	if s1.Paused != s2.Paused || s1.Version != s2.Version {
		return false
	}
	return true
}

func (c *Cluster) logSpecUpdate(oldSpec, newSpec kinkyv1alpha1.KinkySpec) {
	oldSpecBytes, err := json.MarshalIndent(oldSpec, "", "    ")
	if err != nil {
		glog.Errorf("failed to marshal cluster spec: %v", err)
	}
	newSpecBytes, err := json.MarshalIndent(newSpec, "", "    ")
	if err != nil {
		glog.Errorf("failed to marshal cluster spec: %v", err)
	}

	glog.Infof("spec update: Old Spec:")
	for _, m := range strings.Split(string(oldSpecBytes), "\n") {
		glog.Info(m)
	}

	glog.Infof("New Spec:")
	for _, m := range strings.Split(string(newSpecBytes), "\n") {
		glog.Info(m)
	}
}

func (c *Cluster) name() string {
	return c.cluster.GetName()
}
