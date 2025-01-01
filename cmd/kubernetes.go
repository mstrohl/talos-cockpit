package main

import (
	"fmt"
	"log"
	"net/http"
	templmanager "talos-cockpit/internal/tmplmanager"
	"time"
)

type ReturnUpgrade struct {
	MachineID     string
	ClusterID     string
	TargetVersion string
}

// TODO Add OIDC capability
// Uncomment to load all auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth"
//
// Or uncomment to load specific auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

// NewKubernetesClient creates a new Kubernetes client

// upgradeKubernetes apply kubernetes upgrade

func (m *TalosCockpit) upgradeKubernetes(controller string) error {
	var cfg Config

	readFile(&cfg)
	readEnv(&cfg)

	if cfg.Images.CustomRegistryPath != "" {
		r := cfg.Images.CustomRegistryPath
		UpgradeK8SOptions = "--apiserver-image " + r + "/kube-apiserver --controller-manager-image " + r + "/kube-controller-manager --kubelet-image " + r + "/kubelet --scheduler-image " + r + "/kube-scheduler"
	}
	if cfg.Images.KubeProxyEnabled {
		r := cfg.Images.CustomRegistryPath
		UpgradeK8SOptions = UpgradeK8SOptions + " --proxy-image " + r + "/kube-proxy"
	}
	if !cfg.Images.PrePull {
		UpgradeK8SOptions = UpgradeK8SOptions + " --pre-pull-images=false"
	}
	log.Println("K8S upgrade options: ", UpgradeK8SOptions)

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

func performK8SUpgrade(w http.ResponseWriter, r *http.Request, m *TalosCockpit) {
	// Check method is a POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)
		return
	}

	// Récupérer les données du formulaire
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Parsing error on form", http.StatusBadRequest)
		return
	}

	// Get form values
	MachineID := r.Form.Get("ctl")
	//log.Printf("member_id : %s", MachineID)
	ClusterID := r.Form.Get("cluster_id")
	//log.Printf("cluster_id : %s", ClusterID)

	report := ReturnUpgrade{
		MachineID: MachineID,
	}
	templmanager.RenderTemplate(w, "k8s_upgrade_return.tmpl", report)

	// Update Cluster
	m.upgradeKubernetes(MachineID)
	if err != nil {
		log.Printf("Upgrade error for cluster %s: %v", ClusterID, err)
		report := K8SUpdateReport{
			ClusterID:         ClusterID,
			UpdateStatus:      "Failed",
			AdditionalDetails: "Error during K8S Upgrade",
			Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		}

		subject := "Kubernetes Upgrade"
		// Generate email body
		emailBody, err := generateK8SUpdateEmailBody(report)
		if err != nil {
			return
		}

		sendMail(subject, emailBody)
		fmt.Fprintf(w, "<p>Error for cluster %s: %v</p>", ClusterID, err)
	} else {
		log.Printf("Upgrade successful cluster %s", ClusterID)
		report := K8SUpdateReport{
			ClusterID:         ClusterID,
			UpdateStatus:      "Success",
			AdditionalDetails: "K8S updated successfully without any issues",
			Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		}

		subject := "Kubernetes Upgrade"
		// Generate email body
		emailBody, err := generateK8SUpdateEmailBody(report)
		if err != nil {
			return
		}

		sendMail(subject, emailBody)
		fmt.Fprintf(w, "<p>Kubernetes Upgrade successful for cluster %s</p>", ClusterID)
	}

	log.Printf("Kubernetes Upgrade successful for cluster %s", ClusterID)
	//}
	// Redirect to index
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}
