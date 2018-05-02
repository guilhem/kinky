package stub

import (
	api "github.com/barpilot/kinky/pkg/apis/kinky/v1alpha1"
	"github.com/barpilot/kinky/pkg/kinky"

	"github.com/operator-framework/operator-sdk/pkg/sdk/handler"
	"github.com/operator-framework/operator-sdk/pkg/sdk/types"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) error {
	switch o := event.Object.(type) {
	case *api.Cluster:
		return kinky.Reconcile(o)
		// cluster := o
		//
		// // Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// // All secondary resources must have the CR set as their OwnerReference for this to be the case
		// if event.Deleted {
		// 	return nil
		// }
		//
		// err := action.Create(newbusyBoxPod(cluster))
		// if err != nil && !errors.IsAlreadyExists(err) {
		// 	logrus.Errorf("Failed to create busybox pod : %v", err)
		// 	return err
		// }
	}
	return nil
}

// newbusyBoxPod demonstrates how to create a busybox pod
func newbusyBoxPod(cr *api.Cluster) *v1.Pod {
	labels := map[string]string{
		"app": "busy-box",
	}
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "busy-box",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, schema.GroupVersionKind{
					Group:   api.SchemeGroupVersion.Group,
					Version: api.SchemeGroupVersion.Version,
					Kind:    "Cluster",
				}),
			},
			Labels: labels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
