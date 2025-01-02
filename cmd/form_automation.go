package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
)

// Get member information needed to edit system upgrade configuration of a node
func NodeEdit(member_id string, db *sql.DB) ClusterMember {
	// Get memberID
	idStr := member_id

	var member ClusterMember
	if idStr != "" {
		// Check data
		err := db.QueryRow("SELECT member_id, os_version, auto_sys_update FROM cluster_members WHERE member_id = ?", idStr).Scan(&member.MachineID, &member.InstalledVersion, &member.SysUpdate)

		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("Aucun utilisateur trouvé pour l'ID : %s\n", idStr)
			} else {
				fmt.Printf("Erreur de scan : %v\n", err)

				// Check value before Scan
				row := db.QueryRow("SELECT member_id, name, email FROM users WHERE member_id = \"?\"", idStr)
				var member_id, os_version, auto_sys_update string
				scanErr := row.Scan(&member_id, &os_version, &auto_sys_update)

				fmt.Printf("Valeurs récupérées - member_id: %s, os_version: %s, auto_sys_update: %s\n", member_id, os_version, auto_sys_update)
				fmt.Printf("Erreur de scan détaillée : %v\n", scanErr)
			}
		}
	}
	return member

}

// Render node edition template
func handleNodeEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "GET" {
		// Get memberID
		idStr := r.URL.Query().Get("member_id")

		member := NodeEdit(idStr, db)

		// Template form
		templmanager.RenderTemplate(w, "form_auto.tmpl", member)
	}
}

// Render node edition template
func handleClusterEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "GET" {
		// Get memberID
		Clusterid := r.URL.Query().Get("cluster_id")

		var cluster Cluster
		if Clusterid != "" {
			// Check data
			err := db.QueryRow("SELECT name, endpoint, auto_k8s_update FROM clusters WHERE name = ?", Clusterid).Scan(&cluster.Name, &cluster.Endpoint, &cluster.K8sUpdate)

			if err != nil {
				if err == sql.ErrNoRows {
					fmt.Printf("Aucun utilisateur trouvé pour l'ID : %s\n", Clusterid)
				} else {
					fmt.Printf("Erreur de scan : %v\n", err)

					// Check value before Scan
					row := db.QueryRow("SELECT name, endpoint, auto_k8s_update FROM clusters WHERE name = \"?\"", Clusterid)
					var cluster_name, cluster_endpoint, auto_k8s_update string
					scanErr := row.Scan(&cluster_name, &cluster_endpoint, &auto_k8s_update)

					fmt.Printf("Valeurs récupérées - cluster_name: %s, cluster_endpoint: %s, auto_k8s_update: %s\n", cluster_name, cluster_endpoint, auto_k8s_update)
					fmt.Printf("Erreur de scan détaillée : %v\n", scanErr)
				}
			}
		}

		// Template form
		templmanager.RenderTemplate(w, "k8s_auto.tmpl", cluster)
	}
}

// NodeUpdate Update Node information
func NodeUpdate(member_id string, cluster_id string, action string, db *sql.DB) {
	// Get form values
	memberId := member_id
	log.Printf("member_id : %s", memberId)
	clusterId := cluster_id
	log.Printf("cluster_id : %s", clusterId)
	status := action
	log.Printf("auto_sys_updates : %v", status)

	var result sql.Result
	var err error

	if memberId != "" {
		// Update existing node
		log.Printf("UPDATE cluster_members SET auto_sys_update = %v WHERE member_id = %s", status, memberId)
		result, err = db.Exec("UPDATE cluster_members SET auto_sys_update = ? WHERE member_id = ?", status, memberId)
		if err != nil {
			log.Printf("Erreur de mise à jour : %v", err)
			return
		}
	} else if clusterId != "" {
		log.Printf("UPDATE cluster_members SET auto_sys_update = %v WHERE cluster_id = %s", status, clusterId)
		result, err = db.Exec("UPDATE cluster_members SET auto_sys_update = ? WHERE cluster_id = ?", status, clusterId)
		if err != nil {
			log.Printf("Erreur de mise à jour : %v", err)
			return
		}
	}

	// Check results
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Impossible de vérifier les lignes affectées : %v", err)
	}
	if memberId != "" {
		log.Printf("Updating node %s set system automatic update to %s", memberId, status)
	} else if clusterId != "" {
		log.Printf("Updating nodes from cluster %s set system automatic update to %s", clusterId, status)
	}
	log.Printf("Rows updated : %d", rowsAffected)
}

// NodeUpdate Update Node information
func ClusterUpdate(cluster_id string, action string, db *sql.DB) {
	// Get form values
	clusterId := cluster_id
	log.Printf("cluster_id : %s", clusterId)
	status := action
	log.Printf("auto_k8s_updates : %v", status)

	var result sql.Result
	var err error

	if clusterId != "" {
		log.Printf("UPDATE clusters SET auto_k8s_update = %v WHERE name = %s", status, clusterId)
		result, err = db.Exec("UPDATE clusters SET auto_k8s_update = ? WHERE name = ?", status, clusterId)
		if err != nil {
			log.Printf("Erreur de mise à jour : %v", err)
			return
		}
	}

	// Check results
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Impossible de vérifier les lignes affectées : %v", err)
	}
	if clusterId != "" {
		log.Printf("Updating cluster %s set k8s automatic update to %s", clusterId, status)
	}
	log.Printf("Rows updated : %d", rowsAffected)
}

// Apply update on database and redirect to inventory
func handleNodeUpdate(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check method is a POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Parsing error on form", http.StatusBadRequest)
		return
	}

	// Get form values
	MachineID := r.Form.Get("member_id")
	//log.Printf("member_id : %s", MachineID)
	SysUpdate, _ := strconv.ParseBool(r.Form.Get("auto_sys_update"))
	//log.Printf("auto_sys_updates : %v", SysUpdate)
	var result sql.Result
	// Update existing node
	result, err = db.Exec("UPDATE cluster_members SET auto_sys_update = ? WHERE member_id = ?", SysUpdate, MachineID)
	if err != nil {
		log.Printf("Erreur de mise à jour : %v", err)
		http.Error(w, "Erreur de mise à jour", http.StatusInternalServerError)
		return
	}

	// Check results
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Impossible de vérifier les lignes affectées : %v", err)
	}
	log.Printf("Updating node %s set system automatic update to %v", MachineID, SysUpdate)
	log.Printf("Rows updated : %d", rowsAffected)
	//}
	// Rediriger vers la liste des utilisateurs
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

// Apply update on database and redirect to inventory
func handleClusterUpdate(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check method is a POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Parsing error on form", http.StatusBadRequest)
		return
	}

	// Get form values
	ClusterID := r.Form.Get("cluster_id")
	//log.Printf("member_id : %s", MachineID)
	K8sUpdate := r.Form.Get("auto_k8s_update")
	//log.Printf("auto_sys_updates : %v", SysUpdate)
	ClusterUpdate(ClusterID, K8sUpdate, db)
	// Rediriger vers la liste des utilisateurs
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}
