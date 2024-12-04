package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// getClusterMembers récupère les membres d'un cluster spécifique depuis la base de données
func (m *TalosCockpit) getClusterMembers(clusterID string) ([]ClusterMember, error) {
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
			&member.LatestOsVersion,
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
		members = append(members, member)
	}

	return members, nil
}

// startWebServer démarre un serveur web pour visualiser les informations du cluster
func (m *TalosCockpit) startWebServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Récupérer dynamiquement l'ID du cluster
		clusterID, err := m.getClusterID(TalosApiEndpoint)
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
		clientIP, err := m.getNodeIP(TalosApiEndpoint)
		if err != nil {
			log.Printf("Échec de la récupération du NodeIP : %v", err)
		}
		//DEBUG
		//log.Printf("Liste des member ds startwebserver")
		//log.Println(members)

		membersHTML := ""
		for _, member := range members {
			if member.SysUpdate {
				Syscheckbox = "Enable"
			} else {
				Syscheckbox = "Disable"
			}
			if m.K8sUpdate {
				K8scheckbox = "checked"
			} else {
				K8scheckbox = ""
			}
			membersHTML += fmt.Sprintf(`
				<tr>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td><b>%s</b></td>
					<td>
					<a href="/edit?member_id=%s"><button>Modifier</button></a>
					</td>
				</tr>
			`,
				member.Namespace,
				//member.Type,
				member.MachineID,
				member.Hostname,
				member.Role,
				member.ConfigVersion,
				member.LatestOsVersion,
				member.IP,
				member.LastUpdated,
				Syscheckbox,
				member.MachineID,
			)
		}

		html := fmt.Sprintf(`
			<html>
			<script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.6.9/angular.min.js"></script>
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
					<div class="toggle">
					<h3><b> Auto Update K8S : </b><input type="checkbox" id="auto_k8s_update" name="auto_k8s_update" %s></h3>
					</div>
					<table>
					<tr>
						<th>Namespace</th>
						<th>ID</th>
						<th>Hostname</th>
						<th>Machine Type</th>
						<th>Config Version</th>
						<th>OS Version</th>
						<th>Adresses</th>
						<th>Updated at</th>
						<th>Auto Sys Update</th>
						<th>Action</th>
					</tr>
					 %s
					</table>
					</main>
					</div>
				</body>
			</html>
		`, clientIP, clusterID, m.LatestOsVersion, m.ConfigVersion, K8scheckbox, membersHTML)

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
