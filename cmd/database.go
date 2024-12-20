package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
)

// initDatabase SQLite database init
func (m *TalosCockpit) initDatabase() error {
	// Créer le répertoire pour la base de données
	dbDir := filepath.Join(os.Getenv("HOME"), ".talos-cockpit")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	// Open or create databse
	dbPath := filepath.Join(dbDir, "talos_clusters.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS clusters (
			name TEXT UNIQUE PRIMARY KEY,
			endpoint TEXT
		);

		CREATE TABLE IF NOT EXISTS cluster_members (
			cluster_id TEXT,
			namespace TEXT,
			member_id TEXT UNIQUE PRIMARY KEY,
			hostname TEXT,
			machine_type TEXT,
			config_version TEXT,
			os_version TEXT,
			addresses TEXT,
			created_at DATETIME,
			last_updated DATETIME,
			auto_sys_update TEXT,
			auto_k8s_update TEXT,
			FOREIGN KEY(cluster_id) REFERENCES clusters(name)
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_member_id ON cluster_members(member_id);
	`)
	if err != nil {
		return err
	}

	m.db = db
	return nil
}

// upsertCluster insert or replace cluster information
func (m *TalosCockpit) upsertCluster(clusterID, endpoint string) (int, error) {
	result, err := m.db.Exec(`
		INSERT OR REPLACE INTO clusters (name, endpoint) 
		VALUES (?, ?)
	`, clusterID, endpoint)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

// listAndStoreClusterMembers
func (m *TalosCockpit) listAndStoreClusterMembers(endpoint string) ([]ClusterMember, error) {
	// Get Cluster ID
	clusterID, err := m.getClusterID(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Cannot get Cluster ID : %v", err)
	}

	// Récupérer les membres du cluster
	//output, err := m.runCommand("talosctl", "get", "members", "-o", "json", "\|", "jq", "-s", ".")
	cmd := "talosctl -n " + endpoint + " get members -o json | jq -s ."
	//log.Printf("Command: %s", cmd)
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return nil, err
	}

	// Structures pour parser les données JSON
	type MemberData struct {
		Metadata struct {
			Namespace     string      `json:"namespace"`
			Type          string      `json:"type"`
			ID            string      `json:"id"`
			ConfigVersion json.Number `json:"version"`
			Updated       time.Time   `json:"updated"`
		} `json:"metadata"`
		Spec struct {
			Hostname    string   `json:"hostname"`
			MachineType string   `json:"machineType"`
			Addresses   []string `json:"addresses"`
			OsVersion   string   `json:"operatingSystem"`
		} `json:"spec"`
		Node string `json:"node"`
	}

	var memberList []MemberData
	err = json.Unmarshal([]byte(output), &memberList)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	// Debug
	// yaml as is
	//log.Printf("OUTPUT as is:")
	//println(output)

	var members []ClusterMember

	// DEBUG
	//log.Printf("Member List:")
	//fmt.Printf("%+v\n", memberList)

	// Transform members data
	for _, memberData := range memberList {
		member := ClusterMember{
			Namespace: memberData.Metadata.Namespace,
			//Type:            memberData.Metadata.Type,
			MachineID:        memberData.Metadata.ID,
			Hostname:         memberData.Spec.Hostname,
			Role:             memberData.Spec.MachineType,
			ConfigVersion:    memberData.Metadata.ConfigVersion,
			InstalledVersion: strings.TrimLeft(strings.TrimRight(memberData.Spec.OsVersion, ")"), "Talos ("),
			IP:               strings.Join(memberData.Spec.Addresses, ", "),
			LastUpdated:      memberData.Metadata.Updated,
			SysUpdate:        false,
			K8sUpdate:        false,
		}
		members = append(members, member)
	}
	//log.Printf("Member List:")
	//for _, memberData := range memberList {
	//	println(memberData.Metadata.ID)
	//}

	// Insert Cluster into database
	_, err = m.upsertCluster(clusterID, "https://kubernetes.default.svc.cluster.local")
	if err != nil {
		return nil, err
	}

	// Insert members into database
	err = m.upsertClusterMembers(clusterID, members)
	if err != nil {
		return nil, err
	}

	// Update members
	err = m.updateMemberInfo(clusterID, members)
	if err != nil {
		return nil, err
	}

	return members, nil
}

// upsertClusterMembers insert cluster members
func (m *TalosCockpit) upsertClusterMembers(clusterID string, members []ClusterMember) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO cluster_members 
		(cluster_id, namespace, member_id, hostname, machine_type, config_version, os_version, addresses, created_at, last_updated, auto_sys_update, auto_k8s_update)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, member := range members {
		var count int32
		check := m.db.QueryRow("SELECT count(*) as count FROM cluster_members WHERE member_id = ?", member.MachineID)
		check.Scan(&count)
		if count == 0 {
			log.Printf("Adding new member %s with role %s , OS version %s in cluster %s", member.MachineID, member.Role, strings.TrimLeft(strings.TrimRight(member.InstalledVersion, ")"), "Talos ("), clusterID)

			_, err = stmt.Exec(
				clusterID,
				member.Namespace,
				//member.Type,
				member.MachineID,
				member.Hostname,
				member.Role,
				member.ConfigVersion,
				strings.TrimLeft(strings.TrimRight(member.InstalledVersion, ")"), "Talos ("),
				member.IP,
				now,
				member.LastUpdated,
				false,
				false,
			)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

// Update members information
func (m *TalosCockpit) updateMemberInfo(clusterID string, members []ClusterMember) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		UPDATE cluster_members 
		SET os_version = ? , last_updated = ? 
		where member_id = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	//now := time.Now()
	for _, member := range members {
		var result sql.Result
		result, err = stmt.Exec(
			member.InstalledVersion,
			member.LastUpdated,
			member.MachineID,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Cannot find updated rows : %v", err)
		}
		log.Printf("Syncing node %s OS version %s", member.MachineID, member.InstalledVersion)
		log.Printf("Rows updated : %d", rowsAffected)
	}

	return tx.Commit()
}
