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

// getClusterMembers récupère les membres d'un cluster spécifique depuis la base de données
func (m *TalosCockpit) getClusterMembers(clusterID string) ([]ClusterMember, error) {
	fmt.Printf("SELECT cluster_id, namespace,  member_id, hostname, machine_type, config_version, os_version, addresses, created_at,last_updated,auto_sys_update,auto_k8s_update FROM cluster_members WHERE cluster_id = %s", clusterID)
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
			auto_sys_update,
			auto_k8s_update
		FROM cluster_members 
		WHERE cluster_id = ?
	`, clusterID)
	log.Printf("SELECT cluster_id")
	if err != nil {
		return nil, err
	}
	log.Println(rows)
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
			&member.K8sUpdate,
		)
		member.ClusterID = clusterID

		if err != nil {
			return nil, err
		}
		log.Println(member)
		members = append(members, member)
	}

	return members, nil
}

func handleClusterInventory(w http.ResponseWriter, r *http.Request, db *sql.DB, m *TalosCockpit) {

	log.Printf("INVENTORY - TalosApiEndpoint: %s", TalosApiEndpoint)
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
		log.Printf("Cannot Get NodeIP : %v", err)
	}

	var membershtml []MemberHTML
	for _, member := range members {
		if member.SysUpdate {
			Syscheckbox = "\u2705"
		} else {
			Syscheckbox = "\u274C"
		}
		if m.K8sUpdate {
			K8scheckbox = "checked"
		} else {
			K8scheckbox = ""
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
