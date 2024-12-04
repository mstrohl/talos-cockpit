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

// getClusterID récupère dynamiquement l'identifiant du cluster Talos
func (m *TalosCockpit) getClusterID(endpoint string) (string, error) {
	// Exécution de la commande talosctl pour obtenir les informations du cluster
	output, err := m.runCommand("talosctl", "-n", endpoint, "get", "info", "-o", "json")
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

// getNodeIP récupère dynamiquement l'IP
func (m *TalosCockpit) getNodeIP(endpoint string) (string, error) {
	// Exécution de la commande talosctl pour obtenir les informations du cluster
	output, err := m.runCommand("talosctl", "-n", endpoint, "get", "nodeip", "-o", "yaml")
	if err != nil {
		return "", err
	}

	// Structure pour parser les informations du cluster
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

// getTalosVersion récupère la version actuellement installée
func (m *TalosCockpit) getTalosVersion(endpoint string) error {
	output, err := m.runCommand("talosctl", "-n", endpoint, "version")
	if err != nil {
		return err
	}
	m.InstalledVersion = strings.TrimSpace(output)
	return nil
}

// getTalosVersion récupère la version actuellement installée
func (m *TalosCockpit) getMemberVersion(endpoint string) (string, error) {
	// talosctl version --short | grep Tag | awk '{print $2}'
	//talosctl version | sed $'s/\t/  /g' | yq
	output, err := m.runCommand("talosctl", "-n", endpoint, "version")
	if err != nil {
		return "", err
	}
	fmt.Printf(output)
	// Structure pour parser les informations du cluster
	type VersionData struct {
		Client struct {
			Tag string `yaml:"Tag"`
		} `yaml:"Client"`
		Server struct {
			Tag string `yaml:"Tag"`
		} `yaml:"Server"`
	}

	var memberVersion VersionData
	err = yaml.Unmarshal([]byte(output), &memberVersion)
	if err != nil {
		log.Printf("erreur de parsing YAML : %v", err)
		return "", fmt.Errorf("erreur de parsing YAML : %v", err)
	}
	fmt.Printf("Tag: %s", memberVersion.Client.Tag)
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

// getKubeConfig
func (m *TalosCockpit) getKubeConfig(endpoint string) error {
	if home := homedir.HomeDir(); home != "" {
		_, err := m.runCommand("talosctl", "-n", endpoint, "kubeconfig", filepath.Join(home, ".kube", "talos-kubeconfig"))
		if err != nil {
			return err
		}
	}

	return nil
}
