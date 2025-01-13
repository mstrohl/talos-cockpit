package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"talos-cockpit/internal/services"
	templmanager "talos-cockpit/internal/tmplmanager"
)

type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// ApiMemberEdit godoc
//
//	@Summary		Manage nodes auto upgrade system
//	@Description	Manage nodes auto upgrade system
//	@Tags			SysUpdate
//	@ID				nodeEdit
//	@Accept			x-www-form-urlencoded
//	@Produce		plain
//	@Param			member_id	query		string	false	"used to enable/disable automated system upgrade on a node"
//	@Param			cluster_id	query		string	false	"used to enable/disable automated system upgrade on all nodes in the cluster"
//	@Param			enable		query		string	true	"used to define action enabling or disabling"
//	@Success		200			{integer}	string	"answer"
//	@Router			/api/sysupdate [post]
//
// ApiMemberEdit Provide capability to manage nodes auto upgrade system through API calls
func ApiMemberEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var idStr string
	var idCluster string
	var action string

	// Check method is a POST
	// TODO add a GET to get current config
	if r.Method != "POST" {
		http.Error(w, "HTTP-405 Method Not Allowed - Only Method POST is available", 405)
		log.Printf("ApiMemberEdit - Method Not Allowed")
		return
	}

	if r.URL.Query().Get("enable") != "" {
		action = strings.ToLower(r.URL.Query().Get("enable"))
		log.Println("ApiMemberEdit - Action param: ", action)

		switch action {
		case "true":
			log.Printf("ApiMemberEdit - AutoUpgrade enable ")
		case "false":
			log.Printf("ApiMemberEdit - AutoUpgrade disable ")
		default:
			http.Error(w, "status param error", http.StatusBadRequest)
			return
		}

	}

	if r.URL.Query().Get("member_id") != "" {

		// Get memberID
		idStr = r.URL.Query().Get("member_id")
		log.Printf("ApiMemberEdit - member_id %s set to %v", idStr, action)
		NodeUpdate(idStr, "", action, db)

	} else if r.URL.Query().Get("cluster_id") != "" {
		// Get Cluster ID
		idCluster = r.URL.Query().Get("cluster_id")
		log.Printf("ApiMemberEdit - cluster_id %s set to %v \n", idCluster, action)

		NodeUpdate("", idCluster, action, db)

	}

}

// ApiClusterEdit godoc
//
//	@Summary		Manage cluster auto upgrade K8S
//	@Description	Manage nodes auto upgrade K8S
//	@Tags			K8SUpdate
//	@ID				clusterEdit
//	@Accept			x-www-form-urlencoded
//	@Produce		plain
//	@Param			cluster_id	query		string	false	"used to enable/disable automated k8s upgrade"
//	@Param			enable		query		string	true	"used to define action enabling or disabling"
//	@Success		200			{integer}	string	"answer"
//	@Router			/api/k8supdate [post]
//
// ApiClusterEdit Provide capability to manage cluster auto upgrade k8s through API calls
func ApiClusterEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var idCluster string
	var action string

	// Check method is a POST
	// TODO add a GET to get current config
	if r.Method != "POST" {
		http.Error(w, "HTTP-405 Method Not Allowed - Only Method POST is available", 405)
		log.Printf("ApiClusterEdit - Method Not Allowed")
		return
	}

	if r.URL.Query().Get("enable") != "" {
		action = strings.ToLower(r.URL.Query().Get("enable"))
		log.Println("ApiClusterEdit - Action param: ", action)

		switch action {
		case "true":
			log.Printf("ApiClusterEdit - K8SAutoUpgrade enable ")
		case "false":
			log.Printf("ApiClusterEdit - K8SAutoUpgrade disable ")
		default:
			http.Error(w, "status param error", http.StatusBadRequest)
			return
		}

	}

	if r.URL.Query().Get("cluster_id") != "" {
		// Get Cluster ID
		idCluster = r.URL.Query().Get("cluster_id")
		log.Printf("ApiClusterEdit - K8S upgrade for cluster_id %s set to %v \n", idCluster, action)

		ClusterUpdate(idCluster, action, db)

	}

}

// ApiNodeUpgrade API to provide node upgrade
func ApiNodeUpgrade(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var m *TalosCockpit
	var idStr string
	var idCluster string
	var version string

	// Check method is a POST
	// TODO add a GET to get current config
	if r.Method != "POST" {
		http.Error(w, "HTTP-405 Method Not Allowed - Only Method POST is available", 405)
		log.Printf("ApiNodeUpgrade - Method Not Allowed")
		return
	}

	if r.URL.Query().Get("version") != "" {
		version = strings.ToLower(r.URL.Query().Get("version"))
		log.Println("ApiNodeUpgrade - Version param: ", version)

	} else {
		http.Error(w, "status param version missing", http.StatusBadRequest)
		log.Println("ApiNodeUpgrade - Version param missing ")
		return

	}

	if r.URL.Query().Get("member_id") != "" {

		// Get memberID
		idStr = r.URL.Query().Get("member_id")
		log.Printf("ApiMemberEdit - member_id %s set to %v", idStr, version)
		m.customUpgradeSystem(idStr, TalosImageInstaller, version)

	} else if r.URL.Query().Get("cluster_id") != "" {
		// Get Cluster ID
		idCluster = r.URL.Query().Get("cluster_id")
		log.Printf("ApiMemberEdit - cluster_id %s set to %v \n", idCluster, version)
		members, err := m.getClusterMembers(idCluster)
		if err != nil {
			log.Println("Fail to get member list of the cluster ID ", idCluster)
		}

		for _, member := range members {
			m.customUpgradeSystem(member.Hostname, TalosImageInstaller, version)
		}

	}

}

func apiSysUpgrades(w http.ResponseWriter, r *http.Request, m *TalosCockpit, db *sql.DB) {
	// Check method is a POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Parse form data
	//err := r.ParseForm()
	//if err != nil {
	//	http.Error(w, "Parsing error on form", http.StatusBadRequest)
	//	return
	//}

	// Get form values
	Scope := r.URL.Query().Get("scope")
	log.Println("scope : ", Scope)
	SelectedItems := r.URL.Query().Get("selectedItems")
	log.Println("selectedItems : ", SelectedItems)
	Type := r.URL.Query().Get("updateType")
	log.Println("updateType : ", Type)
	TargetVersion := r.URL.Query().Get("specificVersion")
	log.Println("specificVersion : ", TargetVersion)

	// Manage Usage of Latest Version in form
	if TargetVersion == "" {
		TargetVersion = m.LatestOsVersion
	} else {
		if err := checkVersion(TargetVersion, m.LatestOsVersion); err != nil {
			fault := UpgradeFault{
				Error: err,
			}
			templmanager.RenderTemplate(w, "form_err.tmpl", fault)
			return
		}
		log.Printf("Version targeted %v | Last version available %v", TargetVersion, m.LatestOsVersion)
	}

	switch Scope {
	case "label":
		log.Printf("apiSysUpgrades - Scope Label ")
		// Create node service
		nodeService := services.NewNodeService()
		// List nodes with a specific label
		nodes, _, err := nodeService.ListNodesByLabel(SelectedItems)
		if err != nil {
			log.Printf("Failed to list nodes: %v", err)
		}
		log.Println(nodes)
		if Type == "auto" {
			for _, node := range nodes {
				NodeUpdate(node, "", action, db)
			}
		}
		m.updateGroupByLabel(SelectedItems, TargetVersion)
		log.Printf("Upgrade nodes grouped by label %s to version %s", SelectedItems, TargetVersion)

		response := Response{
			Message: "apiSysUpgrades - Upgrade nodes grouped by label " + SelectedItems + " to version " + TargetVersion,
			Status:  http.StatusOK,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case "machine":
		log.Printf("apiSysUpgrades - Scope Machine ")

		members := strings.Split(SelectedItems, ",")

		log.Printf("Upgrade nodes %s to version %s", SelectedItems, TargetVersion)

		for _, member := range members {
			log.Println("apiSysUpgrades - INFO - Starting Upgrade on node ", member)
			//m.customUpgradeSystem(member.Hostname, TalosImageInstaller, TargetVersion)

		}

		log.Printf("Upgrade nodes %s to version %s", SelectedItems, TargetVersion)

		response := Response{
			Message: "apiSysUpgrades - Upgrade nodes grouped by Machine " + SelectedItems + " to version " + TargetVersion,
			Status:  http.StatusOK,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case "cluster":
		log.Printf("apiSysUpgrades - Scope Cluster ")

		members, err := m.getClusterMembers(SelectedItems)
		if err != nil {
			response := Response{
				Message: "apiSysUpgrades - ERROR - " + Scope + " error getting ClusterMembers ",
				Status:  http.StatusInternalServerError,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		log.Printf("Upgrade nodes grouped by cluster %s to version %s", SelectedItems, TargetVersion)

		for _, member := range members {
			msg := "apiSysUpgrades - INFO - " + Scope + " Starting Upgrade on node "
			log.Println(msg, member)
			m.customUpgradeSystem(member.Hostname, TalosImageInstaller, TargetVersion)

		}

		response := Response{
			Message: "apiSysUpgrades - INFO - Upgrade nodes grouped by cluster " + SelectedItems + " to version " + TargetVersion,
			Status:  http.StatusOK,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		//http.Error(w, "apiSysUpgrades param error", http.StatusBadRequest)
		log.Printf("Upgrade nodes grouped by label %s to version %s", SelectedItems, TargetVersion)

		response := Response{
			Message: "apiSysUpgrades - scope param error :" + Scope,
			Status:  http.StatusBadRequest,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

}
