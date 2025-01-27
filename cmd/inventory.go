package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	templmanager "talos-cockpit/internal/tmplmanager"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type MemberManage struct {
	ClientIP        string
	ClusterID       string
	LatestOsVersion string
	SyncSched       time.Duration
	UpgradeSched    time.Duration
	K8scheckbox     string
	MembersHTML     []MemberHTML
}

type MemberHTML struct {
	Namespace        string
	MachineID        string
	Hostname         string
	Role             string
	ConfigVersion    json.Number
	InstalledVersion string
	IP               string
	SysUpdate        bool
	Syscheckbox      string
}

// getClusterMembers get members of a cluster from database
func (m *TalosCockpit) getClusterMembers(clusterID string) ([]ClusterMember, error) {
	//fmt.Printf("SELECT cluster_id, namespace,  member_id, hostname, machine_type, config_version, os_version, addresses, created_at,last_updated,auto_sys_update,auto_k8s_update FROM cluster_members WHERE cluster_id = %s", clusterID)
	rows, err := m.db.Query(`
		SELECT 
			cluster_id, 
			namespace,  
			member_id, 
			hostname, 
			machine_type, 
			config_version, 
			os_version, 
			addresses, 
			created_at,
			last_updated,
			auto_sys_update
		FROM cluster_members 
		WHERE cluster_id = ?
	`, clusterID)
	//log.Printf("SELECT cluster_id")
	if err != nil {
		return nil, err
	}
	//log.Println(rows)
	defer rows.Close()

	var members []ClusterMember

	for rows.Next() {
		var member ClusterMember
		err := rows.Scan(
			&member.ClusterID,
			&member.Namespace,
			&member.MachineID,
			&member.Hostname,
			&member.Role,
			&member.ConfigVersion,
			&member.InstalledVersion,
			&member.IP,
			&member.CreatedAt,
			&member.LastUpdated,
			&member.SysUpdate,
		)
		member.ClusterID = clusterID

		if err != nil {
			return nil, err
		}
		//log.Println(member)
		members = append(members, member)
	}

	return members, nil
}

// getClusterState get status of automatic K8S upgrade of a cluster from database
func (m *TalosCockpit) getClusterState(clusterID string) (bool, error) {
	//fmt.Printf("SELECT name, endpoint, auto_k8s_update FROM clusters WHERE cluster_id = %s", clusterID)
	var state bool
	// Query for a value based on a single row.
	if err := m.db.QueryRow("SELECT auto_k8s_update FROM clusters WHERE name = ?",
		clusterID).Scan(&state); err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("getClusterState %s: unknown cluster", clusterID)
		}
		return false, fmt.Errorf("getClusterState %s: %v", clusterID, err)
	}
	return state, nil
}

// Render inventory tempalte
func handleClusterInventory(w http.ResponseWriter, m *TalosCockpit) {

	//log.Printf("INVENTORY - TalosApiEndpoint: %s", TalosApiEndpoint)
	clusterID, err := m.getClusterID(TalosApiEndpoint)
	if err != nil {
		http.Error(w, "Cannot get cluster ID", http.StatusInternalServerError)
		return
	}
	//log.Printf("INVENTORY - clusterID: %s", clusterID)
	members, err := m.getClusterMembers(clusterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	clientIP, err := m.getNodeIP(TalosApiEndpoint)
	if err != nil {
		log.Printf("Cannot Get NodeIP : %v \n", err)
	}
	k8sUpdate, err := m.getClusterState(clusterID)
	if err != nil {
		log.Printf("Cannot Get Cluster Update State : %v \n", err)
	}
	if k8sUpdate {
		K8scheckbox = "\u2705"
	} else {
		K8scheckbox = "\u274C"
	}
	var membershtml []MemberHTML
	for _, member := range members {
		if member.SysUpdate {
			Syscheckbox = "\u2705"
		} else {
			Syscheckbox = "\u274C"
		}

		// DEBUG
		//log.Printf("Member List:")
		//fmt.Printf("%+v\n", memberList)

		// Transform members data
		memberhtml := MemberHTML{
			Namespace:        member.Namespace,
			MachineID:        member.MachineID,
			Hostname:         member.Hostname,
			Role:             member.Role,
			ConfigVersion:    member.ConfigVersion,
			InstalledVersion: member.InstalledVersion,
			IP:               member.IP,
			Syscheckbox:      Syscheckbox,
		}
		membershtml = append(membershtml, memberhtml)

	}

	MemberManage := MemberManage{
		ClientIP:        clientIP,
		ClusterID:       clusterID,
		LatestOsVersion: m.LatestOsVersion,
		SyncSched:       SyncSched,
		UpgradeSched:    UpgradeSched,
		K8scheckbox:     K8scheckbox,
		MembersHTML:     membershtml,
	}

	templmanager.RenderTemplate(w, "inventory.tmpl", MemberManage)
}
