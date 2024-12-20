package main

import (
	"log"
)

// TODO Add OIDC capability
// Uncomment to load all auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth"
//
// Or uncomment to load specific auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

// NewKubernetesClient creates a new Kubernetes client

// upgradeKubernetes apply kubernetes upgrade
func (m *TalosCockpit) upgradeKubernetes(controller string) error {
	// TODO MANAGE VERSION INSTEAD OF LATEST
	//  + " --to " + k8sversion
	cmd := "talosctl upgrade-k8s -n " + controller
	if UpgradeK8SOptions != "" {
		cmd = cmd + " " + UpgradeK8SOptions
	}
	_, err := m.runCommand("bash", "-c", cmd)

	log.Println("Upgrade K8S command :", cmd)
	return err
}
