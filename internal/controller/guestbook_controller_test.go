/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	guestbookv1 "my.domain/guestbook/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Test_Reconcile(t *testing.T) {
	tests := []struct {
		// Name of the testcase
		Name string
		// Obj to reconcile
		Obj *guestbookv1.Guestbook
		// Previous state in cluster
		State []client.Object
		// Amount of reconciliation loops, defaults to 1
		Loops int
		// Desired result after all loops
		Want ctrl.Result
		// Desired error after all loops
		WantErr error
		// Sideeffects to assert after reconciliation
		WantSideEffects func(ctx context.Context, r *GuestbookReconciler) error
	}{
		{
			Name: "valid",
			Obj:  fixtureGuestbook(),
			WantSideEffects: func(ctx context.Context, r *GuestbookReconciler) error {
				got := &guestbookv1.Guestbook{}
				if err := r.Client.Get(ctx, client.ObjectKeyFromObject(fixtureGuestbook()), got); err != nil {
					return err
				}
				if !got.Status.Done {
					return fmt.Errorf("status should be done, was %t", got.Status.Done)
				}
				return nil
			},
		},
		{
			Name: "custom namespace",
			State: []client.Object{
				fixtureNamespace("custom"),
			},
			Obj: fixtureGuestbook(func(g *guestbookv1.Guestbook) {
				g.Namespace = "custom"
			}),
			WantSideEffects: func(ctx context.Context, r *GuestbookReconciler) error {
				got := &guestbookv1.Guestbook{}
				if err := r.Client.Get(ctx, types.NamespacedName{Namespace: "custom", Name: "my-guestbook"}, got); err != nil {
					return err
				}
				if !got.Status.Done {
					return fmt.Errorf("status should be done, was %t", got.Status.Done)
				}
				return nil
			},
		},
		{
			Name: "spec'd to fail",
			Obj: fixtureGuestbook(func(g *guestbookv1.Guestbook) {
				g.Spec.Foo = "fail"
			}),
			WantErr: &FailSpecError{},
		},
	}

	// prepare context, scheme, fakeclient and reconciler
	ctx := context.Background()
	if err := guestbookv1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("init scheme: %s", err)
	}
	env := envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
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
	reconciler := GuestbookReconciler{Client: c}

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
				if err := tt.WantSideEffects(ctx, &reconciler); err != nil {
					t.Error("failed sideeffect:", err)
				}
			}
		})
	}
}

func fixtureGuestbook(mods ...func(*guestbookv1.Guestbook)) *guestbookv1.Guestbook {
	f := &guestbookv1.Guestbook{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-guestbook",
			Namespace: "default",
		},
	}
	for _, mod := range mods {
		mod(f)
	}
	return f
}

func fixtureNamespace(name string, mods ...func(*corev1.Namespace)) *corev1.Namespace {
	f := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, mod := range mods {
		mod(f)
	}
	return f
}
