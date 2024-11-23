package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"encoding/json"

	"github.com/google/go-github/v39/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

// Cluster représente les informations de base sur un cluster Kubernetes
type Cluster struct {
	ID       int
	Name     string
	Endpoint string
}

// ClusterMember contient les détails d'un membre du cluster Talos
type ClusterMember struct {
	ClusterID       string
	Namespace       string
	Type            string
	MachineID       string
	Hostname        string
	Role            string
	ConfigVersion   json.Number
	latestOsVersion string
	IP              string
	LastUpdated     time.Time
}

// TalosVersionManager gère les opérations sur le cluster Talos
type TalosVersionManager struct {
	githubClient    *github.Client
	webServer       *http.Server
	db              *sql.DB
	ConfigVersion   string
	latestOsVersion string
	clientInfo      string
}

// filterIPv4Addresses filtre et ne conserve que les adresses IPv4 valides
func filterIPv4Addresses(addresses []string) []string {
	var ipv4Addresses []string
	for _, addr := range addresses {
		ip := net.ParseIP(addr)
		if ip != nil && ip.To4() != nil {
			ipv4Addresses = append(ipv4Addresses, addr)
		}
	}
	return ipv4Addresses
}

// runCommand exécute une commande système et retourne sa sortie
func (m *TalosVersionManager) runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", string(output), err)
	}
	return string(output), nil
}

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
			last_updated DATETIME,
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
	// sortie yaml brute
	log.Printf("Sortie Brute:")
	println(output)

	var members []ClusterMember

	// Stocker le premier client comme client global
	//if len(memberList) > 0 {
	//	m.clientInfo = memberList.Spec
	//}
	log.Printf("Member List:")
	fmt.Printf("%+v\n", memberList)

	// Transformer les données membres
	for _, memberData := range memberList {
		member := ClusterMember{
			Namespace: memberData.Metadata.Namespace,
			//Type:            memberData.Metadata.Type,
			MachineID:       memberData.Metadata.ID,
			Hostname:        memberData.Spec.Hostname,
			Role:            memberData.Spec.MachineType,
			ConfigVersion:   memberData.Metadata.ConfigVersion,
			latestOsVersion: strings.TrimLeft(strings.TrimRight(memberData.Spec.OsVersion, ")"), "Talos ("),
			IP:              strings.Join(memberData.Spec.Addresses, ", "),
		}
		members = append(members, member)
	}
	log.Printf("Liste:")
	for _, memberData := range memberList {
		println(memberData.Metadata.ID)
	}

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
		INSERT OR REPLACE INTO cluster_members 
		(cluster_id, namespace, member_id, hostname, machine_type, config_version, os_version, addresses, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			strings.TrimLeft(strings.TrimRight(member.latestOsVersion, ")"), "Talos ("),
			member.IP,
			now,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// getClusterMembers récupère les membres d'un cluster spécifique depuis la base de données
func (m *TalosVersionManager) getClusterMembers(clusterID string) ([]ClusterMember, error) {
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
			last_updated 
		FROM cluster_members 
		WHERE cluster_id = ?
	`, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []ClusterMember
	for rows.Next() {
		var member ClusterMember
		err := rows.Scan(
			&member.ClusterID,
			&member.Namespace,
			//&member.Type,
			&member.MachineID,
			&member.Hostname,
			&member.Role,
			&member.ConfigVersion,
			&member.latestOsVersion,
			&member.IP,
			&member.LastUpdated,
		)
		member.ClusterID = clusterID

		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

// fetchLatestRelease récupère la dernière version de Talos depuis GitHub
func (m *TalosVersionManager) fetchLatestRelease() error {
	ctx := context.Background()
	release, _, err := m.githubClient.Repositories.GetLatestRelease(ctx, "siderolabs", "talos")
	if err != nil {
		return err
	}
	m.latestOsVersion = release.GetTagName()
	return nil
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

// upgradeSystem effectue la mise à jour du système Talos
func (m *TalosVersionManager) upgradeSystem() error {
	_, err := m.runCommand(
		"talosctl",
		"upgrade",
		"-n", "nodeIP",
		"--image", m.latestOsVersion,
		"--preserve=true",
	)
	return err
}

// upgradeKubernetes effectue la mise à jour de Kubernetes
func (m *TalosVersionManager) upgradeKubernetes() error {
	_, err := m.runCommand(
		"talosctl",
		"upgrade-k8s",
		"-n", "nodeIP",
		"--to", m.latestOsVersion,
	)
	return err
}

// scheduleClusterSync gère la synchronisation périodique du cluster
func (m *TalosVersionManager) scheduleClusterSync() {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := m.fetchLatestRelease(); err != nil {
					log.Printf("Échec de la récupération de la dernière version : %v", err)
				}

				if err := m.getConfigVersion(); err != nil {
					log.Printf("Échec de la récupération de la version installée : %v", err)
				}

				_, err := m.listAndStoreClusterMembers()
				if err != nil {
					log.Printf("Échec de la synchronisation des membres du cluster : %v", err)
				}

				if m.latestOsVersion != m.ConfigVersion {
					if err := m.upgradeSystem(); err != nil {
						log.Printf("Échec de la mise à jour du système : %v", err)
					}

					if err := m.upgradeKubernetes(); err != nil {
						log.Printf("Échec de la mise à jour de Kubernetes : %v", err)
					}
				}
			}
		}
	}()
}

// startWebServer démarre un serveur web pour visualiser les informations du cluster
func (m *TalosVersionManager) startWebServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Récupérer dynamiquement l'ID du cluster
		clusterID, err := m.getClusterID()
		if err != nil {
			http.Error(w, "Impossible de récupérer l'ID du cluster", http.StatusInternalServerError)
			return
		}

		//// Convertir l'ID en entier pour la base de données
		//_, err = m.upsertCluster(clusterID, "https://kubernetes.default.svc.cluster.local")
		//if err != nil {
		//	http.Error(w, "Erreur lors de l'insertion du cluster", http.StatusInternalServerError)
		//	return
		//}

		members, err := m.getClusterMembers(clusterID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clientIP, err := m.getNodeIP()
		if err != nil {
			log.Printf("Échec de la récupération du NodeIP : %v", err)
		}
		//DEBUG
		log.Printf("Liste des member ds startwebserver")
		log.Println(members)

		membersHTML := ""
		for _, member := range members {
			membersHTML += fmt.Sprintf(`
				<tr>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
				</tr>
			`,
				member.Namespace,
				//member.Type,
				member.MachineID,
				member.Hostname,
				member.Role,
				member.ConfigVersion,
				member.latestOsVersion,
				member.IP,
			)
		}

		html := fmt.Sprintf(`
			<html>
				<head>
					<title>Talos Cockpit</title>
					<style>
						.client-info { 
							background-color: #f0f0f0; 
							padding: 10px; 
							margin-bottom: 20px; 
							border-radius: 5px; 
							font-weight: bold;
						}
						table { 
							border-collapse: collapse; 
							width: 100%%; 
						}
						th, td { 
							border: 1px solid #ddd; 
							padding: 8px; 
							text-align: left; 
						}
						th { 
							background-color: #f2f2f2; 
							font-weight: bold;
						}
						/* (PART A) SHARED */
						/* (PART A1) STANDARD FONT & BOX SIZING */
						* {
						font-family: Arial, Helvetica, sans-serif;
						box-sizing: border-box;
						}

						/* (PART A2) COLOR & PADDING */
						#top, #side { color: #f54714; background: #37304b; }
						#top, #side, #main, #slinks { padding: 10px; }

						/* (PART A3) FLEX LAYOUT */
						html, body, #top, #bottom { display: flex; }
						#bottom, #main { flex-grow: 1; }

						/* (PART B) BODY - SPLIT TOP-BOTTOM */
						html, body {
						padding: 0; margin: 0; min-height: 100vh;
						flex-direction: column;
						}

						/* (PART C) TOP NAV BAR */
						#top {
						position: sticky; height: 50px;
						align-items: center;
						}

						/* (PART D1) SIDEBAR */
						#side { width: 220px; transition: all 0.2s; }


						/* (PART D3) SIDEBAR LINKS */
						#slinks a {
						display: block;
						padding: 10px 8px; margin-bottom: 5px;
						color: #fff; text-decoration: none;
						}
						#slinks a:hover, #slinks a.now {
						background: #111; border-radius: 10px;
						}
						#slinks i { font-style: normal; }

						/* (PART E) RESPONSIVE */
						/* (PART E1) SIDEBAR TOGGLE BUTTON */
						#stog {
						display: none; cursor: pointer;
						font-size: 28px; margin-right: 10px;
						}

						/* (PART E2) ON SMALL SCREENS */
						@media screen and (max-width: 600px) {
						/* (PART E2-1) SHOW TOGGLE BUTTON */
						#stog { display: block; }

						/* (PART E2-2) SHRINK SIDEBAR */
						#side.mini { width: 100px; }
						#side.mini #upic { width: 60px; height: 60px; }
						#side.mini #uname, #side.mini #uacct, #side.mini #slinks span { display: none; }
						#side.mini #slinks a { text-align: center; }
						#side.mini #slinks i { font-size: 32px; }
						}
					</style>
				</head>
				<body>
					<!-- (PART A) TOP NAV BAR -->
					<nav id="top">
					<!-- (PART A1) SIDEBAR TOGGLE -->
					<div id="stog" onclick="document.getElementById('side').classList.toggle('mini')">
						&#9776;
					</div>

					<!-- (PART A2) LOGO & WHATEVER ELSE -->
					<h1>Talos Cockpit</h1>
					</nav>

					<!-- (PART B) BOTTOM CONTENT -->
					<div id="bottom">
					<!-- (PART B1) SIDEBAR -->
					<nav id="side" class="mini">

						<!-- (PART B1-2) LINKS -->
						<div id="slinks">
						<a href="#">
							<i>&#9733;</i> <span>Section</span>
						</a>
						</div>
					</nav>

					<!-- (PART B2) MAIN CONTENT -->
					<main id="main">
					<div class="client-info">
					Machine ayant traité votre requête : %s
					</div>
					<h1>Talos Cluster Manager</h1>
					<p>ID du Cluster : %s</p>
					<p>Dernière version disponible : %s</p>
					<p>Version installée : %s</p>
					<h2>Membres du Cluster</h2>
					<table>
					<tr>
						<th>Namespace</th>
						<th>ID</th>
						<th>Hostname</th>
						<th>Machine Type</th>
						<th>Config Version</th>
						<th>OS Version</th>
						<th>Adresses</th>
					</tr>
					 %s
					</table>
					</main>
					</div>
				</body>
			</html>
		`, clientIP, clusterID, m.latestOsVersion, m.ConfigVersion, membersHTML)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	m.webServer = &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Démarrage du serveur web sur http://localhost:8080")
		if err := m.webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erreur du serveur HTTP : %v", err)
		}
	}()
}

// NewTalosVersionManager crée une nouvelle instance du gestionnaire de versions
func NewTalosVersionManager(githubToken string) (*TalosVersionManager, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	manager := &TalosVersionManager{
		githubClient: github.NewClient(tc),
	}

	if err := manager.initDatabase(); err != nil {
		return nil, err
	}

	return manager, nil
}

// Fonction principale qui initialise et démarre le gestionnaire de cluster
func main() {
	// Récupérer le token GitHub depuis l'environnement
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("Un token GitHub est requis. Définissez GITHUB_TOKEN.")
	}

	// Créer une nouvelle instance du gestionnaire
	manager, err := NewTalosVersionManager(githubToken)
	if err != nil {
		log.Fatalf("Échec de l'initialisation du gestionnaire : %v", err)
	}

	// Récupérer l'ID du cluster
	clusterID, err := manager.getClusterID()
	if err != nil {
		log.Fatalf("Impossible de récupérer l'ID du cluster : %v", err)
	}
	log.Printf("ID du cluster : %s", clusterID)

	// Synchroniser les membres du cluster
	_, err = manager.listAndStoreClusterMembers()
	if err != nil {
		log.Fatalf("Échec de la synchronisation initiale des membres du cluster : %v", err)
	}

	// Récupérer la dernière version disponible
	if err := manager.fetchLatestRelease(); err != nil {
		log.Fatalf("Échec de la récupération de la dernière version : %v", err)
	}

	// Récupérer la version actuellement installée
	if err := manager.getConfigVersion(); err != nil {
		log.Fatalf("Échec de la récupération de la version installée : %v", err)
	}

	// Démarrer le serveur web
	manager.startWebServer()

	// Planifier la synchronisation périodique
	manager.scheduleClusterSync()

	// Attendre un signal d'interruption
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Arrêter proprement le serveur web
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.webServer.Shutdown(ctx); err != nil {
		log.Printf("Erreur lors de l'arrêt du serveur web : %v", err)
	}
}
