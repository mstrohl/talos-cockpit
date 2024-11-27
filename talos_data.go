package main

import (
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
)

// getClusterID récupère dynamiquement l'identifiant du cluster Talos
func (m *TalosVersionManager) getClusterID() (string, error) {
	// Exécution de la commande talosctl pour obtenir les informations du cluster
	output, err := m.runCommand("talosctl", "get", "info", "-o", "yaml")
	if err != nil {
		return "", err
	}

	// Structure pour parser les informations du cluster
	type ClusterInfoData struct {
		Spec struct {
			ClusterID string `yaml:"clusterId"`
		} `yaml:"spec"`
	}

	var clusterInfo ClusterInfoData
	err = yaml.Unmarshal([]byte(output), &clusterInfo)
	if err != nil {
		return "", fmt.Errorf("erreur de parsing YAML : %v", err)
	}

	return clusterInfo.Spec.ClusterID, nil
}

// getNodeIP récupère dynamiquement l'IP
func (m *TalosVersionManager) getNodeIP() (string, error) {
	// Exécution de la commande talosctl pour obtenir les informations du cluster
	output, err := m.runCommand("talosctl", "get", "nodeip", "-o", "yaml")
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

// getConfigVersion récupère la version actuellement installée
func (m *TalosVersionManager) getConfigVersion() error {
	output, err := m.runCommand("talosctl", "version")
	if err != nil {
		return err
	}
	m.ConfigVersion = strings.TrimSpace(output)
	return nil
}

// getConfigVersion récupère la version actuellement installée
func (m *TalosVersionManager) getKubeConfig() error {
	output, err := m.runCommand("talosctl", "kubeconfig", "~/talos-kubeconfig")
	if err != nil {
		return err
	}
	m.ConfigVersion = strings.TrimSpace(output)
	return nil
}
