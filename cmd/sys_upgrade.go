package main

import (
	"log"

	"talos-cockpit/internal/services"
)

// upgradeSystem effectue la mise à jour du système Talos
func (m *TalosCockpit) upgradeSystem(node string, installerImage string) error {
	log.Printf("Launch System upgrade to %s for node %s", m.LatestOsVersion, node)
	log.Printf("talosctl upgrade -n %s --image %s:%s --preserve=true", node, installerImage, m.LatestOsVersion)
	_, err := m.runCommand(
		"talosctl",
		"upgrade",
		"-n", node,
		"--image", installerImage+":"+m.LatestOsVersion,
		"--preserve=true",
	)
	return err
}

// customUpgradeSystem effectue la mise à jour du système Talos
func (m *TalosCockpit) customUpgradeSystem(node string, installerImage string, version string) error {
	log.Printf("Launch Sytem upgrade to %s for node %s", version, node)
	log.Printf("talosctl upgrade -n %s --image %s:%s --preserve=true", node, installerImage, version)

	_, err := m.runCommand(
		"talosctl",
		"upgrade",
		"-n", node,
		"--image", installerImage+":"+m.LatestOsVersion,
		"--preserve=true",
	)
	return err
}

func (m *TalosCockpit) updateGroupByLabel(label string, version string) {
	// Create node service
	nodeService := services.NewNodeService()

	// List nodes with a specific label
	nodes, _, err := nodeService.ListNodesByLabel(label)
	if err != nil {
		log.Printf("Failed to list nodes: %v", err)
	}
	//nodes := m.getNodeByLabel(label)
	log.Printf("Node list %s matching label %s", nodes, label)
	log.Printf("There are %d nodes in the cluster matching the label %s \n", len(nodes), label)

	for _, node := range nodes {
		installedVersion, _ := m.getMemberVersion(node)
		log.Printf("Wanted to upgrade node %s matching label %s from version %s to version %s", nodes, label, installedVersion, version)
		//if err := m.customUpgradeSystem(node, TalosImageInstaller, version); err != nil {
		//	log.Printf("Error during automatic Node Upgrade : %v", err)
		//	report := NodeUpdateReport{
		//		NodeName:          node,
		//		PreviousVersion:   installedVersion,
		//		ImageSource:       TalosImageInstaller,
		//		NewVersion:        version,
		//		UpdateStatus:      "Failed",
		//		AdditionalDetails: "Error during Upgrade Node by label",
		//		Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		//	}
		//
		//	subject := "Automatic Node Upgrade"
		//	// Generate email body
		//	emailBody, err := generateUpdateEmailBody(report)
		//	if err != nil {
		//		return
		//	}
		//
		//	sendMail(subject, emailBody)
		//} else {
		//	log.Printf("Operating system of %s Updated to : %s", node, TalosImageInstaller)
		//	report := NodeUpdateReport{
		//		NodeName:          node,
		//		PreviousVersion:   installedVersion,
		//		ImageSource:       TalosImageInstaller,
		//		NewVersion:        version,
		//		UpdateStatus:      "Success",
		//		AdditionalDetails: "Node updated successfully without any issues",
		//		Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		//	}
		//
		//	subject := "Automatic Node Upgrade"
		//	// Generate email body
		//	emailBody, err := generateUpdateEmailBody(report)
		//	if err != nil {
		//		return
		//	}
		//
		//	sendMail(subject, emailBody)
		//}
	}
	return
}
