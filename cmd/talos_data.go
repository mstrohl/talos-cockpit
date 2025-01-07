package main

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/homedir"
)

// getClusterID Get id of the talos cluster
func (m *TalosCockpit) getClusterID(endpoint string) (string, error) {
	// Exécution de la commande talosctl pour obtenir les informations du cluster
	cmd := "talosctl -n " + endpoint + " get info -o json 2>/dev/null"
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return "", err
	}

	// Structure pour parser les informations du cluster
	type ClusterInfoData struct {
		Spec struct {
			ClusterID string `json:"clusterId"`
		} `json:"spec"`
	}

	var clusterInfo ClusterInfoData
	err = json.Unmarshal([]byte(output), &clusterInfo)
	if err != nil {
		return "", fmt.Errorf("erreur de parsing YAML : %v", err)
	}

	return clusterInfo.Spec.ClusterID, nil
}

// getNodeIP get talos node IP
func (m *TalosCockpit) getNodeIP(endpoint string) (string, error) {
	cmd := "talosctl -n " + endpoint + " get nodeip -o yaml 2>/dev/null"
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return "", err
	}

	type NodeInfoData struct {
		Spec struct {
			Addresses []string `yaml:"addresses"`
		} `yaml:"spec"`
	}

	var nodeInfo NodeInfoData
	err = yaml.Unmarshal([]byte(output), &nodeInfo)
	if err != nil {
		return "", fmt.Errorf("erreur de parsing YAML : %v", err)
	}

	return nodeInfo.Spec.Addresses[0], nil
}

// getNodeIP get talos node IP
func (m *TalosCockpit) getLatestK8sVersion() error {
	cmd := "talosctl images default | grep kubelet | cut -d: -f2 2>/dev/null"
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return err
	}

	log.Println("getLatestK8sVersion - output:", output)
	m.K8sVersionAvailable = strings.TrimSpace(output)
	return nil
}

// TODO: Check if needed
// getTalosVersion get installed version of talos
func (m *TalosCockpit) getTalosVersion(endpoint string) error {
	output, err := m.runCommand("talosctl", "-n", endpoint, "version")
	if err != nil {
		return err
	}
	m.InstalledVersion = strings.TrimSpace(output)
	return nil
}

// getTalosctlVersion get installed version of talos
func (m *TalosCockpit) getTalosctlVersion(endpoint string) (string, error) {
	cmd := "talosctl version -n " + endpoint + " --client | sed $'s/\t/  /g' | yq -o json"
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return "", err
	}
	//fmt.Printf(output)
	// Struct to parse json
	type TalosctlVersion struct {
		Client struct {
			Tag string `json:"Tag"`
		} `json:"Client"`
	}

	var talosctlVersion TalosctlVersion
	err = json.Unmarshal([]byte(output), &talosctlVersion)
	if err != nil {
		log.Printf("erreur de parsing YAML : %v", err)
		return "", fmt.Errorf("erreur de parsing YAML : %v", err)
	}
	log.Printf("Client Tag: %s\n", talosctlVersion.Client.Tag)
	return talosctlVersion.Client.Tag, nil
}

// getTalosVersion get installed version of talos
func (m *TalosCockpit) getMemberVersion(endpoint string) (string, error) {
	cmd := "talosctl version -n " + endpoint + " | sed $'s/\t/  /g' | yq -o json"
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return "", err
	}
	//fmt.Printf(output)
	// Struct to parse json
	type VersionData struct {
		Client struct {
			Tag string `json:"Tag"`
		} `json:"Client"`
		Server struct {
			Tag string `json:"Tag"`
		} `json:"Server"`
	}

	var memberVersion VersionData
	err = json.Unmarshal([]byte(output), &memberVersion)
	if err != nil {
		log.Printf("erreur de parsing YAML : %v", err)
		return "", fmt.Errorf("erreur de parsing YAML : %v", err)
	}
	log.Printf("Server Tag: %s\n", memberVersion.Server.Tag)
	return memberVersion.Server.Tag, nil
}

//func GetEndpoints() []string {
//	//cl := client.New(context.TODO(), WithConfigFromFile{})
//	cl, _ := client.New(context.TODO(), client.WithDefaultConfig())
//	output := cl.GetEndpoints()
//
//	fmt.Println(output)
//	return output
//}

// TODO: Check if needed
// getKubeConfig get kubeconfig from TalosAPI
func (m *TalosCockpit) getKubeConfig(endpoint string) error {
	if home := homedir.HomeDir(); home != "" {
		_, err := m.runCommand("talosctl", "-n", endpoint, "kubeconfig", filepath.Join(home, ".kube", "talos-kubeconfig"))
		if err != nil {
			return err
		}
	}

	return nil
}
