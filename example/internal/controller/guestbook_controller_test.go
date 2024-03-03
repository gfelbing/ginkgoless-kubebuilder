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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gfelbing/ginkgoless-kubebuilder/envtesthelper"
	guestbookv1 "github.com/gfelbing/ginkgoless-kubebuilder/example/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Test_Reconcile(t *testing.T) {
	tests := []envtesthelper.TestCase[*GuestbookReconciler]{
		{
			Name:            "valid",
			Obj:             fixtureGuestbook(),
			WantSideEffects: assertStatusDone(types.NamespacedName{Namespace: "default", Name: "my-guestbook"}),
		},
		{
			Name: "custom namespace",
			State: []client.Object{
				fixtureNamespace("custom"),
			},
			Obj: fixtureGuestbook(func(g *guestbookv1.Guestbook) {
				g.Namespace = "custom"
			}),
			WantSideEffects: assertStatusDone(types.NamespacedName{Namespace: "custom", Name: "my-guestbook"}),
		},
		{
			Name: "spec'd to fail",
			Obj: fixtureGuestbook(func(g *guestbookv1.Guestbook) {
				g.Spec.Foo = "fail"
			}),
			WantErr: &FailSpecError{},
		},
	}
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
	envtesthelper.RunEnvTest(
		t, guestbookv1.AddToScheme, env,
		func(c client.Client) *GuestbookReconciler {
			return &GuestbookReconciler{Client: c}
		},
		tests,
	)
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

func assertStatusDone(namespacedName types.NamespacedName) func(context.Context, *GuestbookReconciler) error {
	return func(ctx context.Context, r *GuestbookReconciler) error {
		got := &guestbookv1.Guestbook{}
		if err := r.Client.Get(ctx, namespacedName, got); err != nil {
			return err
		}
		if !got.Status.Done {
			return fmt.Errorf("status should be done, was %t", got.Status.Done)
		}
		return nil
	}
}
