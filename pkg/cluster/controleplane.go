package cluster

import (
	kinky "github.com/barpilot/kinky/pkg/apis/kinky/v1alpha1"
	"github.com/barpilot/kinky/pkg/util"
	"github.com/barpilot/kinky/pkg/util/constants"
	apiv1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane"
	"k8s.io/kubernetes/pkg/util/version"
)

func GetControleplaneDeployments(cluster kinky.Kinky, cfg *kubeadm.MasterConfiguration, k8sVersion *version.Version) map[string]*extv1beta1.Deployment {
	deployments := make(map[string]*extv1beta1.Deployment)

	pods := controlplane.GetStaticPodSpecs(cfg, k8sVersion)
	pods["kube-apiserver"].Spec.Containers[0].Ports = []apiv1.ContainerPort{
		{
			ContainerPort: 443,
			Name:          "secure",
		},
	}

	for _, pod := range pods {
		pod.Spec.HostNetwork = false

		pod.ObjectMeta.Name = cluster.Name + "-" + pod.ObjectMeta.Name
		pod.ObjectMeta.Labels["component"] = cluster.Name + "-" + pod.ObjectMeta.Labels["component"]

		for i, volume := range pod.Spec.Volumes {
			if volume.Name == kubeadmconstants.KubeCertificatesVolumeName {
				pod.Spec.Volumes[i].VolumeSource = apiv1.VolumeSource{
					Secret: &apiv1.SecretVolumeSource{
						SecretName: kubeadmconstants.KubeCertificatesVolumeName,
					},
				}
			}
			if volume.Name == kubeadmconstants.KubeConfigVolumeName {
				pod.Spec.Volumes[i].VolumeSource = apiv1.VolumeSource{
					Secret: &apiv1.SecretVolumeSource{
						SecretName: constants.KubeconfigSecret,
					},
				}
				for iC, container := range pod.Spec.Containers {
					for iVM, volumeMount := range container.VolumeMounts {
						if volumeMount.Name == kubeadmconstants.KubeConfigVolumeName {
							pod.Spec.Containers[iC].VolumeMounts[iVM].MountPath = kubeadmconstants.KubernetesDir
							pod.Spec.Containers[iC].VolumeMounts[iVM].ReadOnly = false
						}
					}
				}
			}

			pod.Spec.Containers[0].LivenessProbe.HTTPGet.Host = ""
		}

		deployments[pod.Name] = &extv1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: cluster.Namespace,
			},
			Spec: extv1beta1.DeploymentSpec{
				Replicas: util.Int32Ptr(1),
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: pod.ObjectMeta,
					Spec:       pod.Spec,
				},
			},
		}
	}

	return deployments
}
