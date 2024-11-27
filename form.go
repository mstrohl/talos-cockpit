package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"

	_ "github.com/mattn/go-sqlite3"
)

func handleNodeEdit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Récupérer l'ID de l'utilisateur (s'il existe)
	idStr := r.URL.Query().Get("member_id")

	var member ClusterMember
	if idStr != "" {
		// Récupérer l'utilisateur existant
		err := db.QueryRow("SELECT member_id, os_version, auto_sys_update FROM cluster_members WHERE member_id = ?", idStr).Scan(&member.MachineID, &member.LatestOsVersion, &member.SysUpdate)
		// Débogage explicite
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("Aucun utilisateur trouvé pour l'ID : %s\n", idStr)
			} else {
				fmt.Printf("Erreur de scan : %v\n", err)

				// Vérification des valeurs avant le scan
				row := db.QueryRow("SELECT member_id, name, email FROM users WHERE member_id = \"?\"", idStr)
				var member_id, os_version, auto_sys_update string
				scanErr := row.Scan(&member_id, &os_version, &auto_sys_update)

				fmt.Printf("Valeurs récupérées - member_id: %s, os_version: %s, auto_sys_update: %s\n", member_id, os_version, auto_sys_update)
				fmt.Printf("Erreur de scan détaillée : %v\n", scanErr)
			}
		}
	}

	// Template pour le formulaire d'édition
	tmpl := `
	<!DOCTYPE html>
	<html>
	<script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.6.9/angular.min.js"></script>
	<head>
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
	<title>Editor</title>
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
		<h1>Modifier la configuration du noeud {{.MachineID}}</h1>
		<form action="/update" method="post">
			<label>MachineID :</label><br>
			<input type="text" name="member_id" value="{{.MachineID}}" readonly><br><br>
			
			<label>LatestOsVersion :</label><br>
			<input type="text" name="LatestOsVersion" value="{{.LatestOsVersion}}" readonly><br><br>

			<label>Auto Update Système :</label><br>
			<select list="auto_sys_update" name="auto_sys_update">
				<option value=true {{if .SysUpdate}} selected="selected" {{end}}>True</option>
				<option value=false {{if not .SysUpdate}} selected="selected" {{end}}>False</option>
			</select>
			<input type="submit" value="Mettre à jour">
		</form>
		<br>
		<a href="/">Retour à la liste</a>
	</body>
	</html>
	`

	t, err := template.New("editNode").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, member)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleNodeUpdate(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Vérifier que c'est bien un POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Récupérer les données du formulaire
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Erreur de parsing du formulaire", http.StatusBadRequest)
		return
	}

	//cluster_id,
	//namespace,
	//member_id,
	//hostname,
	//machine_type,
	//config_version,
	//os_version,
	//addresses,
	//created_at,
	//last_updated,
	//auto_sys_update,
	//auto_k8s_update

	// Récupérer les valeurs
	MachineID := r.Form.Get("member_id")
	log.Printf("member_id : %s", MachineID)
	SysUpdate, _ := strconv.ParseBool(r.Form.Get("auto_sys_update"))
	log.Printf("auto_sys_updates : %v", SysUpdate)
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
	log.Printf("Lignes affectées : %d", rowsAffected)
	//}
	// Rediriger vers la liste des utilisateurs
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
