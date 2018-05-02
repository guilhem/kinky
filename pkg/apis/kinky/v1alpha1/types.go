package v1alpha1

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sutil "github.com/barpilot/kinky/pkg/utils/k8s"
)

type ClusterPhase string

const (
	ClusterPhaseInitial ClusterPhase = ""
	ClusterPhaseRunning              = "Running"
)

const (
	defaultBaseImage = "quay.io/coreos/vault"
	// version format is "<upstream-version>-<our-version>"
	defaultVersion = "stable-1.9"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Cluster `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status,omitempty"`
}

type ClusterSpec struct {
	Version string `json:"version,omitempty"`

	// Pod defines the policy for pods owned by vault operator.
	// This field cannot be updated once the CR is created.
	Pod *PodPolicy `json:"pod,omitempty"`
	// Fill me
}
type ClusterStatus struct {
	Ready bool `json:"ready,omitempty"`
	// Fill me
	// Phase indicates the state this Vault cluster jumps in.
	// Phase goes as one way as below:
	//   Initial -> Running
	Phase ClusterPhase `json:"phase"`
}

// PodPolicy defines the policy for pods owned by vault operator.
type PodPolicy struct {
	// Resources is the resource requirements for the containers.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

// SetDefaults sets the default vaules for the kinky spec and returns true if the spec was changed
func (cl *Cluster) SetDefaults() bool {
	changed := false
	cls := &cl.Spec
	if len(cls.Version) == 0 {
		cls.Version = defaultVersion
		changed = true
	}
	// if cls.TLS == nil {
	// 	cls.TLS = &TLSPolicy{Static: &StaticTLS{
	// 		ServerSecret: DefaultVaultServerTLSSecretName(cl.Name),
	// 		ClientSecret: DefaultVaultClientTLSSecretName(cl.Name),
	// 	}}
	// 	changed = true
	// }
	return changed
}

func (cl *Cluster) Validate() error {
	cls := &cl.Spec
	if _, err := k8sutil.KubernetesReleaseVersion(cls.Version); err != nil {
		return fmt.Errorf("fail to set kubernetes version: %v", err)
	}
	return nil
}

// DefaultVaultClientTLSSecretName returns the name of the default vault client TLS secret
func DefaultVaultClientTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-client-tls"
}

// DefaultVaultServerTLSSecretName returns the name of the default vault server TLS secret
func DefaultVaultServerTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-server-tls"
}
