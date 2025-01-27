package main

import (
	"log"
	"math"
	"net/http"
	"time"

	"talos-cockpit/internal/services"
	templmanager "talos-cockpit/internal/tmplmanager"
)

type SysUpgrade struct {
	ClusterID       string
	LatestOsVersion string
	MembersHTML     []MemberHTML
	Versions        []string
}

// upgradeSystem upgrade talos system of a node
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

// Keep it outside legacy upgradeSystem to split automatic upgrade management and human triggered one
// customUpgradeSystem upgrade with a custom version
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

// updateGroupByLabel upgrade all node matching the parameter label
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

// Render multi patch template
func sysUpgradeHandler(w http.ResponseWriter, m *TalosCockpit) {
	//log.Printf("INVENTORY - TalosApiEndpoint: %s", TalosApiEndpoint)
	versions, _ := fetchLastTalosReleases("")

	clusterID, err := m.getClusterID(TalosApiEndpoint)
	if err != nil {
		http.Error(w, "Cannot get cluster ID", http.StatusInternalServerError)
		return
	}
	//log.Printf("INVENTORY - clusterID: %s", clusterID)
	members, err := m.getClusterMembers(clusterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var membershtml []MemberHTML
	for _, member := range members {
		if member.SysUpdate {
			Syscheckbox = "\u2705"
		} else {
			Syscheckbox = "\u274C"
		}

		// DEBUG
		//log.Printf("Member List:")
		//fmt.Printf("%+v\n", memberList)

		// Transform members data
		memberhtml := MemberHTML{
			Namespace:        member.Namespace,
			MachineID:        member.MachineID,
			Hostname:         member.Hostname,
			Role:             member.Role,
			ConfigVersion:    member.ConfigVersion,
			InstalledVersion: member.InstalledVersion,
			IP:               member.IP,
			Syscheckbox:      Syscheckbox,
		}
		membershtml = append(membershtml, memberhtml)

	}

	data := SysUpgrade{
		ClusterID:       clusterID,
		LatestOsVersion: m.LatestOsVersion,
		MembersHTML:     membershtml,
		Versions:        versions,
	}

	// Template form
	templmanager.RenderTemplate(w, "sys_upgrades.tmpl", data)

}

func safeUpgrade(m *TalosCockpit) {
	safeUpgradeDate := m.LatestReleaseDate.AddDate(0, 0, UpgradeSafePeriod)
	log.Printf("No upgrade before %v for the latest release %s", safeUpgradeDate, m.LatestOsVersion)

	timeLeft := safeUpgradeDate.Sub(time.Now().UTC())

	if timeLeft > time.Hour*24 {
		days := math.Round(timeLeft.Hours() / 24)
		log.Printf("%v days remaining for a safe Upgrade", days)
	} else if timeLeft >= time.Second {
		log.Printf("%v remainings for a safe Upgrade", timeLeft)
	} else {
		timeLeft := time.Now().UTC().Sub(safeUpgradeDate)
		log.Printf("Safe upgrades available since %v", timeLeft)

		log.Printf("Launching Upgrade schedule")
		m.scheduleClusterUpgrade(UpgradeSched, TalosApiEndpoint)
	}
}
