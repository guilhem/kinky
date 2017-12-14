package main

import (
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func createIngress(k8sClient *kubernetes.Clientset, name, namespace, host, service string) error {
	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{"kubernetes.io/ingress.class": "k8sniff"},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				v1beta1.IngressRule{
					Host: host,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								v1beta1.HTTPIngressPath{
									Backend: v1beta1.IngressBackend{
										ServiceName: service,
										ServicePort: intstr.Parse("443"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := k8sClient.Extensions().Ingresses(namespace).Create(ingress)
	if !apierrors.IsAlreadyExists(err) {
		return err
	}
	_, err = k8sClient.Extensions().Ingresses(namespace).Update(ingress)
	return err
}
