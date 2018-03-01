package operator

import (
	"github.com/spotahome/kooper/client/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	kinkyv1alpha1 "github.com/barpilot/kinky/apis/kinky/v1alpha1"
	podtermk8scli "github.com/barpilot/kinky/client/k8s/clientset/versioned"
)

// podTerminatorCRD is the crd pod terminator.
type podTerminatorCRD struct {
	crdCli     crd.Interface
	kubecCli   kubernetes.Interface
	podTermCli podtermk8scli.Interface
}

func newPodTermiantorCRD(podTermCli podtermk8scli.Interface, crdCli crd.Interface, kubeCli kubernetes.Interface) *podTerminatorCRD {
	return &podTerminatorCRD{
		crdCli:     crdCli,
		podTermCli: podTermCli,
		kubecCli:   kubeCli,
	}
}

// podTerminatorCRD satisfies resource.crd interface.
func (p *podTerminatorCRD) Initialize() error {
	crd := crd.Conf{
		Kind:       kinkyv1alpha1.PodTerminatorKind,
		NamePlural: kinkyv1alpha1.PodTerminatorNamePlural,
		Group:      kinkyv1alpha1.SchemeGroupVersion.Group,
		Version:    kinkyv1alpha1.SchemeGroupVersion.Version,
		Scope:      kinkyv1alpha1.PodTerminatorScope,
	}

	return p.crdCli.EnsurePresent(crd)
}

// GetListerWatcher satisfies resource.crd interface (and retrieve.Retriever).
func (p *podTerminatorCRD) GetListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return p.podTermCli.kinkyv1alpha1().PodTerminators().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return p.podTermCli.kinkyv1alpha1().PodTerminators().Watch(options)
		},
	}
}

// GetObject satisfies resource.crd interface (and retrieve.Retriever).
func (p *podTerminatorCRD) GetObject() runtime.Object {
	return &kinkyv1alpha1.PodTerminator{}
}
