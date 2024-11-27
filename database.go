package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
)

// initDatabase initialise la base de données SQLite pour stocker les informations du cluster
func (m *TalosVersionManager) initDatabase() error {
	// Créer le répertoire pour la base de données
	dbDir := filepath.Join(os.Getenv("HOME"), ".talos-manager")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	// Ouvrir ou créer la base de données
	dbPath := filepath.Join(dbDir, "talos_clusters.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Créer les tables nécessaires
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

// upsertCluster insère ou met à jour les informations d'un cluster
func (m *TalosVersionManager) upsertCluster(clusterID, endpoint string) (int, error) {
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

// listAndStoreClusterMembers récupère et stocke les informations des membres du cluster
func (m *TalosVersionManager) listAndStoreClusterMembers() ([]ClusterMember, error) {
	// Récupérer l'ID du cluster
	clusterID, err := m.getClusterID()
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer l'ID du cluster : %v", err)
	}

	// Récupérer les membres du cluster
	//output, err := m.runCommand("talosctl", "get", "members", "-o", "json", "\|", "jq", "-s", ".")
	cmd := "talosctl get members -o json | jq -s ."
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		return nil, err
	}

	// Structures pour parser les données YAML
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
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}
	// Debug
	// sortie yaml brute
	//log.Printf("Sortie Brute:")
	//println(output)

	var members []ClusterMember

	// Stocker le premier client comme client global
	//if len(memberList) > 0 {
	//	m.clientInfo = memberList.Spec
	//}

	// DEBUG
	//log.Printf("Member List:")
	//fmt.Printf("%+v\n", memberList)

	// Transformer les données membres
	for _, memberData := range memberList {
		member := ClusterMember{
			Namespace: memberData.Metadata.Namespace,
			//Type:            memberData.Metadata.Type,
			MachineID:       memberData.Metadata.ID,
			Hostname:        memberData.Spec.Hostname,
			Role:            memberData.Spec.MachineType,
			ConfigVersion:   memberData.Metadata.ConfigVersion,
			LatestOsVersion: strings.TrimLeft(strings.TrimRight(memberData.Spec.OsVersion, ")"), "Talos ("),
			IP:              strings.Join(memberData.Spec.Addresses, ", "),
			LastUpdated:     memberData.Metadata.Updated,
			SysUpdate:       false,
			K8sUpdate:       false,
		}
		members = append(members, member)
	}
	//log.Printf("Liste:")
	//for _, memberData := range memberList {
	//	println(memberData.Metadata.ID)
	//}

	// Insérer ou mettre à jour le cluster
	_, err = m.upsertCluster(clusterID, "https://kubernetes.default.svc.cluster.local")
	if err != nil {
		return nil, err
	}

	// Stocker les membres du cluster
	err = m.upsertClusterMembers(clusterID, members)
	if err != nil {
		return nil, err
	}

	return members, nil
}

// upsertClusterMembers insère ou met à jour les informations des membres du cluster
func (m *TalosVersionManager) upsertClusterMembers(clusterID string, members []ClusterMember) error {
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
		_, err = stmt.Exec(
			clusterID,
			member.Namespace,
			//member.Type,
			member.MachineID,
			member.Hostname,
			member.Role,
			member.ConfigVersion,
			strings.TrimLeft(strings.TrimRight(member.LatestOsVersion, ")"), "Talos ("),
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

	return tx.Commit()
}

func (m *TalosVersionManager) updateMemberInfo(version string, clusterID string, members []ClusterMember) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		UPDATE cluster_members 
		set os_version = ? , last_updated = ? 
		where cluster_id = ? AND member_id = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	//now := time.Now()
	for _, member := range members {
		_, err = stmt.Exec(
			version,
			member.LastUpdated,
			member.MachineID,
			clusterID,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
