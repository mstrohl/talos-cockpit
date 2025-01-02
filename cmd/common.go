package main

import (
	"net/http"
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

// Render single patch template
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
