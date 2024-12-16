package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
)

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

func handleNodeEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "GET" {
		// Get memberID
		idStr := r.URL.Query().Get("member_id")

		member := NodeEdit(idStr, db)

		// Template form
		templmanager.RenderTemplate(w, "form_auto.tmpl", member)
	}
}

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
		// Mise à jour d'un noeud existant
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

	// Vérification du résultat
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
	//}
	// Rediriger vers la liste des utilisateurs
	//http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

func handleNodeUpdate(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check method is a POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Récupérer les données du formulaire
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
	// Mise à jour d'un noeud existant
	result, err = db.Exec("UPDATE cluster_members SET auto_sys_update = ? WHERE member_id = ?", SysUpdate, MachineID)
	if err != nil {
		log.Printf("Erreur de mise à jour : %v", err)
		http.Error(w, "Erreur de mise à jour", http.StatusInternalServerError)
		return
	}

	// Vérification du résultat
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

func ApiNodeEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var idStr string
	var idCluster string
	var action string

	// Check method is a POST
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
		log.Println("ApiNodeEdit - cluster_id %s set to %v ", idCluster, action)

		NodeUpdate("", idCluster, action, db)

	}

}
