package envtesthelper

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Test_RunEnvTest(t *testing.T) {
	tests := []TestCase[*mockReconciler]{
		{
			Name: "happy",
			Obj: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "test-namespace",
				},
			},
			State: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			WantSideEffects: func(ctx context.Context, r *mockReconciler) error {
				cm := &corev1.ConfigMap{}
				err := r.Client.Get(ctx, types.NamespacedName{Name: "test-cm", Namespace: "test-namespace"}, cm)
				if err != nil {
					return fmt.Errorf("get obj: %w", err)
				}
				if foo, ok := cm.Data["foo"]; !ok || foo != "bar" {
					return fmt.Errorf("want %q, got %q", "bar", foo)
				}
				return nil
			},
		},
	}
	RunEnvTest(
		t,
		corev1.AddToScheme,
		&envtest.Environment{},
		NewMockReconciler,
		tests,
	)
}

type mockReconciler struct {
	Client client.Client
}

func NewMockReconciler(client client.Client) *mockReconciler {
	return &mockReconciler{
		Client: client,
	}
}

func (r *mockReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, req.NamespacedName, cm); err != nil {
		return ctrl.Result{}, fmt.Errorf("get obj: %w", err)
	}
  _, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{
			"foo": "bar",
		}
		return nil
	})
  if err != nil {
		return ctrl.Result{}, fmt.Errorf("update obj: %w", err)
  }
	return ctrl.Result{}, nil
}
