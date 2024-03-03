package utils

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

type Reconciler interface {
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}

type TestCase[R Reconciler] struct {
	// Name of the testcase
	Name string
	// Obj to reconcile
	Obj client.Object
	// Previous state in cluster
	State []client.Object
	// Amount of reconciliation loops, defaults to 1
	Loops int
	// Desired result after all loops
	Want ctrl.Result
	// Desired error after all loops
	WantErr error
	// Sideeffects to assert after reconciliation. Objects created by controller should be cleaned up here.
	WantSideEffects func(ctx context.Context, r R) error
}

// RunEnvTest bootstraps a testenv and executes all given testcases.
// addToScheme be used to add the controller scheme, e.g. by passing yourapiv1.AddToScheme
func RunEnvTest[R Reconciler](
	t *testing.T,
	addToScheme func(*runtime.Scheme) error,
	env *envtest.Environment,
	newReconciler func(client.Client) R,
	tests []TestCase[R],
) {
	t.Helper()
	// prepare context, scheme, fakeclient and reconciler
	ctx := context.Background()

	if err := addToScheme(scheme.Scheme); err != nil {
		t.Fatalf("init scheme: %s", err)
	}

	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("init envtest: %s", err)
	}
	defer func() {
		if err := env.Stop(); err != nil {
			t.Fatal("stop testenv:", err)
		}
	}()
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatalf("init client: %s", err)
	}
	reconciler := newReconciler(c)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// create state & obj
			for _, obj := range tt.State {
				obj := obj
				if err := c.Create(ctx, obj); err != nil {
					t.Fatalf("create obj: %s", err)
				}
				defer func() {
					if err := c.Delete(ctx, obj); err != nil {
						t.Fatalf("create obj: %s", err)
					}
				}()
			}
			if err := c.Create(ctx, tt.Obj); err != nil {
				t.Fatalf("create obj: %s", err)
			}
			defer func() {
				if err := c.Delete(ctx, tt.Obj); err != nil {
					t.Fatalf("create obj: %s", err)
				}
			}()

			// run the reconciliation
			var got ctrl.Result
			var gotErr error
			for i := 0; i < max(1, tt.Loops); i++ {
				got, gotErr = reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: client.ObjectKeyFromObject(tt.Obj),
				})
			}

			// assert error, reconcile result and state
			if !errors.Is(gotErr, tt.WantErr) {
				t.Errorf("gotErr: %s\nwant: %s", gotErr, tt.WantErr)
				return
			}
			if diff := cmp.Diff(got, tt.Want); diff != "" {
				t.Errorf("got: %v\nwant: %v\ndiff: %s", got, tt.Want, diff)
			}
			if tt.WantSideEffects != nil {
				if err := tt.WantSideEffects(ctx, reconciler); err != nil {
					t.Error("failed sideeffect:", err)
				}
			}
		})
	}
}
