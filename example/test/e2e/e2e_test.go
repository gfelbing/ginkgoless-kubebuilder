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

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"my.domain/guestbook/test/utils"
)

const namespace = "ginkgoless-kubebuilder-system"

func Test_E2E(t *testing.T) {
	t.Run("prepare test env", func(t *testing.T) {
		if err := utils.InstallPrometheusOperator(); err != nil {
			t.Fatal("install prometheus:", err)
		}
		if err := utils.InstallCertManager(); err != nil {
			t.Fatal("install certmanager:", err)
		}
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		if _, err := utils.Run(cmd); err != nil {
			t.Fatal("create namespace:", err)
		}
	})
	t.Cleanup(func() {
		utils.UninstallPrometheusOperator()
		utils.UninstallCertManager()
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})
	t.Run("operator", func(t *testing.T) {
		var controllerPodName string

		// projectimage stores the name of the image used in the example
		var projectimage = "example.com/ginkgoless-kubebuilder:v0.0.1"

		t.Log("building the manager(Operator) image")
		cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
		if _, err := utils.Run(cmd); err != nil {
			t.Error(err)
			return
		}

		t.Log("loading the the manager(Operator) image on Kind")
		if err := utils.LoadImageToKindClusterWithName(projectimage); err != nil {
			t.Error(err)
			return
		}

		t.Log("installing CRDs")
		cmd = exec.Command("make", "install")
		if _, err := utils.Run(cmd); err != nil {
			t.Error(err)
			return
		}

		t.Log("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
		if _, err := utils.Run(cmd); err != nil {
			t.Error(err)
			return
		}

		t.Log("validating that the controller-manager pod is running as expected")
		verifyControllerUp := func() error {
			// Get pod name

			cmd = exec.Command("kubectl", "get",
				"pods", "-l", "control-plane=controller-manager",
				"-o", "go-template={{ range .items }}"+
					"{{ if not .metadata.deletionTimestamp }}"+
					"{{ .metadata.name }}"+
					"{{ \"\\n\" }}{{ end }}{{ end }}",
				"-n", namespace,
			)

			podOutput, err := utils.Run(cmd)
			if err != nil {
				return fmt.Errorf("pod output: %w", err)
			}
			podNames := utils.GetNonEmptyLines(string(podOutput))
			if len(podNames) != 1 {
				return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
			}
			controllerPodName = podNames[0]
			substr := "controller-manager"
			if strings.Contains(controllerPodName, substr) {
				return fmt.Errorf("expected %q in %q", substr, controllerPodName)
			}

			// Validate pod status
			cmd = exec.Command("kubectl", "get",
				"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
				"-n", namespace,
			)
			status, err := utils.Run(cmd)
			if err != nil {
				return fmt.Errorf("get pods: %w", err)
			}
			if string(status) != "Running" {
				return fmt.Errorf("controller pod in %s status", status)
			}
			return nil
		}
		if err := eventually(verifyControllerUp, time.Minute, time.Second); err != nil {
			t.Error("verify controller up:", err)
		}
	})
}

func eventually(f func() error, timeout time.Duration, intervall time.Duration) error {
	ticker := time.NewTicker(intervall)
	defer ticker.Stop()
	to := time.NewTimer(timeout)
	defer to.Stop()
	var err error
	for {
		select {
		case <-to.C:
			return fmt.Errorf("timeout %s reached: %w", timeout, err)
		case <-ticker.C:
			err = f()
			if err == nil {
				return nil
			}
		}
	}
}
