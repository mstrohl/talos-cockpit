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

func handleNodeEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get memberID
	idStr := r.URL.Query().Get("member_id")

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

	// Template form
	templmanager.RenderTemplate(w, "form_auto.tmpl", member)
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
	log.Printf("Updating node %s set system automatic update to %s", MachineID, SysUpdate)
	log.Printf("Rows updated : %d", rowsAffected)
	//}
	// Rediriger vers la liste des utilisateurs
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
