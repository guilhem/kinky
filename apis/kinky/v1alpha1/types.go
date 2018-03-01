package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	k8sutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Database describes a database.
type Kinky struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KinkySpec     `json:"spec"`
	Status ClusterStatus `json:"status"`
}

// DatabaseSpec is the spec for a Foo resource
type KinkySpec struct {
	Version string `json:"version,omitempty"`
	// Paused is to pause the control of the operator for the etcd cluster.
	Paused bool `json:"paused,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatabaseList is a list of Database resources
type KinkyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Kinky `json:"items"`
}

// TODO: should be an admission controller
func (e *Kinky) SetDefaults() {
	c := &e.Spec
	if len(c.Version) == 0 {
		c.Version = kubeadmapi.DefaultKubernetesVersion
	}
}

func (c *KinkySpec) Validate() error {
	v, err := k8sutil.KubernetesReleaseVersion(c.Version)
	if err != nil {
		return fmt.Errorf("fail to set kubernetes version: %v", err)
	}
	c.Version = v
	return nil
}
