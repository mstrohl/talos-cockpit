package main

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"talos-cockpit/internal/services"
	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
	v1 "k8s.io/api/core/v1"
)

type DashboardData struct {
	ClientIP            string
	ClusterID           string
	LatestOsVersion     string
	LastPreRelease      string
	SyncSched           time.Duration
	UpgradeSched        time.Duration
	NodeCount           int
	NodeData            []v1.Node
	LatestK8sVersion    string
	TalosctlVersion     string
	MaintenanceDuration time.Duration
	SafetyPeriod        int
	Timeremaining       string
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
		log.Printf("Fail to get NodeIP : %v", err)
	}

	//latestK8S := m.getLatestK8sVersion()
	//if err != nil {
	//	log.Printf("Fail to get last k8s available version : %v", err)
	//}

	cliVersion, err := m.getTalosctlVersion(TalosApiEndpoint)
	if err != nil {
		log.Printf("Fail to get talosctl cli version : %v", err)
	}

	safeDate := m.LatestReleaseDate.AddDate(0, 0, UpgradeSafePeriod)
	safetimeLeft := safeDate.Sub(time.Now().UTC())
	var timesremain string
	if safetimeLeft > time.Hour*24 {
		timesremain = strconv.FormatFloat(math.Round(safetimeLeft.Hours()/24), 'f', -1, 64) + "d"
	} else if safetimeLeft >= time.Second {
		timesremain = strings.Split(safetimeLeft.String(), "m")[0] + "m"
	} else {
		timesremain = ""
	}

	DashboardData := DashboardData{
		ClientIP:            clientIP,
		ClusterID:           clusterID,
		LatestOsVersion:     m.LatestOsVersion,
		LatestK8sVersion:    m.K8sVersionAvailable,
		TalosctlVersion:     cliVersion,
		SyncSched:           SyncSched,
		UpgradeSched:        UpgradeSched,
		LastPreRelease:      LastPreRelease,
		NodeCount:           len(nodes),
		NodeData:            data,
		SafetyPeriod:        UpgradeSafePeriod,
		MaintenanceDuration: time.Duration(Mro),
		Timeremaining:       timesremain,
	}

	templmanager.RenderTemplate(w, "index.tmpl", DashboardData)
}
