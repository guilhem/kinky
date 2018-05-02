package kinky

import (
	api "github.com/barpilot/kinky/pkg/apis/kinky/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// addOwnerRefToObject appends the desired OwnerReference to the object
func addOwnerRefToObject(o metav1.Object, r metav1.OwnerReference) {
	o.SetOwnerReferences(append(o.GetOwnerReferences(), r))
}

// labelsForKinky returns the labels for selecting the resources
// belonging to the given kinky name.
func labelsForKinky(name string) map[string]string {
	return map[string]string{"app": "kinky", "kinky_cluster": name}
}

// asOwner returns an owner reference set as the kinky cluster CR
func asOwner(v *api.Cluster) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: api.SchemeGroupVersion.String(),
		Kind:       api.KinkyKind,
		Name:       v.Name,
		UID:        v.UID,
		Controller: &trueVar,
	}
}
