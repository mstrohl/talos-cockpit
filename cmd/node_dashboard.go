package main

import (
	"net/http"
	"strings"
	templmanager "talos-cockpit/internal/tmplmanager"
)

type NodeReport struct {
	Dmesg         string
	MachineID     string
	MachineConfig string
}

// Get dmesg of a node
func getNodeDmesg(endpoint string, m *TalosCockpit) (dmesg string, err error) {
	output, err := m.runCommand("talosctl", "-n", endpoint, "dmesg")
	if err != nil {
		return "", err
	}
	return output, nil
}

// Get machine config of a node
func getNodeMC(endpoint string, m *TalosCockpit) (mc string, err error) {
	cmd := "talosctl -n " + endpoint + " get mc -ojson | jq -r .spec"
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return "", err
	}
	//log.Println("MC Output:", output)
	return output, nil
}

// Render Node dashboard template
func handleNodeDashboard(w http.ResponseWriter, r *http.Request, m *TalosCockpit) {
	endpoint := r.URL.Query().Get("member_id")
	//log.Printf("Dashboard")
	sourcedmesg, _ := getNodeDmesg(endpoint, m)
	dmesg := strings.Replace(sourcedmesg, ` \n `, "\n ", -1)
	//log.Println("MC Output:", sourcedmesg)
	srcmc, _ := getNodeMC(endpoint, m)
	//log.Println("MC Output:", srcmc)
	mc := strings.Replace(srcmc, `\n `, "\n ", -1)
	data := NodeReport{
		MachineID:     endpoint,
		Dmesg:         dmesg,
		MachineConfig: mc,
	}

	templmanager.RenderTemplate(w, "node_dashboard.tmpl", data)

}
