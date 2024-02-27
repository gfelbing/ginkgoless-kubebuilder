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
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	guestbookv1 "my.domain/guestbook/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_Reconcile(t *testing.T) {
	tests := []struct {
		Name            string
		Obj             *guestbookv1.Guestbook
		State           []client.Object
		Want            ctrl.Result
		WantErr         error
		WantSideEffects func(ctx context.Context, r *GuestbookReconciler) error
	}{
		{
			Name: "valid",
			Obj:  fixtureGuestbook(),
			Want: ctrl.Result{},
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
			Name: "spec'd to fail",
			Obj: fixtureGuestbook(func(g *guestbookv1.Guestbook) {
				g.Spec.Foo = "fail"
			}),
			WantErr: &FailSpecError{},
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// prepare context, scheme and fakeclient
			ctx := context.Background()
			sc := scheme.Scheme
			if err := guestbookv1.AddToScheme(sc); err != nil {
				t.Fatalf("init scheme: %s", err)
			}
			c := fake.NewClientBuilder().
				WithScheme(sc).
				WithObjects(tt.Obj).
				WithObjects(tt.State...).
				Build()

			// create the reconciler and run it
			reconciler := GuestbookReconciler{
				Client: c,
				Scheme: sc,
			}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.Obj.Namespace,
					Name:      tt.Obj.Name,
				},
			})

			// assert error, reconcile result and state
			if !errors.Is(err, tt.WantErr) {
				t.Errorf("gotErr: %s, want: %s", err, tt.WantErr)
				return
			}
			if diff := cmp.Diff(result, tt.Want); diff != "" {
				t.Errorf("got: %v\nwant: %v\ndiff: %s", result, tt.Want, diff)
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
