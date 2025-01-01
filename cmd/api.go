package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

// ApiNodeEdit godoc
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
// ApiNodeEdit Provide capability to manage nodes auto upgrade system through API calls
func ApiNodeEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var idStr string
	var idCluster string
	var action string

	// Check method is a POST
	// TODO add a GET to get current config
	if r.Method != "POST" {
		http.Error(w, "HTTP-405 Method Not Allowed - Only Method POST is available", 405)
		log.Printf("ApiNodeEdit - Method Not Allowed")
		return
	}

	if r.URL.Query().Get("enable") != "" {
		action = strings.ToLower(r.URL.Query().Get("enable"))
		log.Println("ApiNodeEdit - Action param: ", action)

		switch action {
		case "true":
			log.Printf("ApiNodeEdit - AutoUpgrade enable ")
		case "false":
			log.Printf("ApiNodeEdit - AutoUpgrade disable ")
		default:
			http.Error(w, "status param error", http.StatusBadRequest)
			return
		}

	}

	if r.URL.Query().Get("member_id") != "" {

		// Get memberID
		idStr = r.URL.Query().Get("member_id")
		log.Printf("ApiNodeEdit - member_id %s set to %v", idStr, action)
		NodeUpdate(idStr, "", action, db)

	} else if r.URL.Query().Get("cluster_id") != "" {
		// Get Cluster ID
		idCluster = r.URL.Query().Get("cluster_id")
		log.Printf("ApiNodeEdit - cluster_id %s set to %v \n", idCluster, action)

		NodeUpdate("", idCluster, action, db)

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
		log.Printf("ApiNodeEdit - member_id %s set to %v", idStr, version)
		m.customUpgradeSystem(idStr, TalosImageInstaller, version)

	} else if r.URL.Query().Get("cluster_id") != "" {
		// Get Cluster ID
		idCluster = r.URL.Query().Get("cluster_id")
		log.Printf("ApiNodeEdit - cluster_id %s set to %v \n", idCluster, version)
		members, err := m.getClusterMembers(idCluster)
		if err != nil {
			log.Println("Fail to get member list of the cluster ID ", idCluster)
		}

		for _, member := range members {
			m.customUpgradeSystem(member.Hostname, TalosImageInstaller, version)
		}

	}

}
