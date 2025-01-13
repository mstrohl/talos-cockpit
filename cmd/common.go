package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"talos-cockpit/internal/services"
	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
)

type Nodes struct {
	Hostname string
}

type K8SManage struct {
	ClusterID string
	NodeList  []Nodes
}

// filterIPv4Addresses filter IPv4 from list of IP addresses
func filterIPv4Addresses(addresses []string) []string {
	var ipv4Addresses []string
	for _, addr := range addresses {
		ip := net.ParseIP(addr)
		if ip != nil && ip.To4() != nil {
			ipv4Addresses = append(ipv4Addresses, addr)
		}
	}
	return ipv4Addresses
}

// TODO replace cmd.CombinedOutput() by managing both output
// runCommand exec command wrapper
func (m *TalosCockpit) runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", string(output), err)
	}
	return string(output), nil
}

// Render k8s nodes template
func availableK8SNodes(w http.ResponseWriter, m *TalosCockpit, l string, t string) {
	//log.Printf("INVENTORY - TalosApiEndpoint: %s", TalosApiEndpoint)
	/////////////////////
	// TODO:  Use clusterID to get the right K8S endpoint
	//
	clusterID, err := m.getClusterID(TalosApiEndpoint)
	if err != nil {
		http.Error(w, "Cannot get cluster ID", http.StatusInternalServerError)
		return
	}
	//log.Printf("INVENTORY - clusterID: %s", clusterID)

	// Create node service
	nodeService := services.NewNodeService()

	nodes, _, err := nodeService.ListNodesByLabel(l)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var nodeList []Nodes
	for _, node := range nodes {
		structNode := Nodes{
			Hostname: node,
		}
		nodeList = append(nodeList, structNode)

	}

	data := K8SManage{
		ClusterID: clusterID,
		NodeList:  nodeList,
	}

	// Template form
	templmanager.RenderTemplate(w, t, data)

}

func DownloadFile(filepath string, url string) error {

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func checkVersion(TargetVersion string, Ref string) error {
	var err error
	if strings.HasPrefix(TargetVersion, "v") {
		log.Printf("target_version : %s", TargetVersion)
	} else {
		//http.Error(w, "Error on form - Version has bad format", http.StatusBadRequest)
		err = fmt.Errorf("Error on form - Version has bad format. Should be like vX.Y.Z")
	}
	if compareVersions(TargetVersion, Ref) > 0 && TargetVersion != Ref {
		//http.Error(w, "Error on form - Version is higher than the last version available", http.StatusBadRequest)
		err = fmt.Errorf("Error on form - Version %s is higher than the last version available %s", TargetVersion, Ref)
	}
	return err
}
