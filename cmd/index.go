package main

import (
	"log"
	"net/http"
	"time"

	"talos-cockpit/internal/services"
	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
	v1 "k8s.io/api/core/v1"
)

type DashboardData struct {
	ClientIP        string
	ClusterID       string
	LatestOsVersion string
	LastPreRelease  string
	SyncSched       time.Duration
	UpgradeSched    time.Duration
	NodeCount       int
	NodeData        []v1.Node
}

// Render index/dashboard template
func handleIndex(w http.ResponseWriter, m *TalosCockpit) {
	// Get ClusterID
	clusterID, err := m.getClusterID(TalosApiEndpoint)
	if err != nil {
		http.Error(w, "Impossible de récupérer l'ID du cluster", http.StatusInternalServerError)
		return
	}

	nodeService := services.NewNodeService()

	// List nodes with a specific label
	nodes, data, err := nodeService.ListNodesByLabel("")
	if err != nil {
		log.Printf("Failed to list nodes: %v", err)

	}

	//for _, node := range nodes {
	//	log.Printf("Node %s status %s", node)
	//}

	clientIP, err := m.getNodeIP(TalosApiEndpoint)
	if err != nil {
		log.Printf("Échec de la récupération du NodeIP : %v", err)
	}

	DashboardData := DashboardData{
		ClientIP:        clientIP,
		ClusterID:       clusterID,
		LatestOsVersion: m.LatestOsVersion,
		SyncSched:       SyncSched,
		UpgradeSched:    UpgradeSched,
		LastPreRelease:  LastPreRelease,
		NodeCount:       len(nodes),
		NodeData:        data,
	}

	templmanager.RenderTemplate(w, "index.tmpl", DashboardData)
}
