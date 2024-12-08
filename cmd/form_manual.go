package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	templmanager "talos-cockpit/internal/tmplmanager"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ReturnMan struct {
	MachineID     string
	TargetVersion string
}

func upgradeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get memberID
	idStr := r.URL.Query().Get("member_id")
	var manual ManualUpdateForm
	if idStr != "" {
		// Check data
		err := db.QueryRow("SELECT member_id, os_version, auto_sys_update FROM cluster_members WHERE member_id = ?", idStr).Scan(&manual.MachineID, &manual.InstalledVersion, &manual.SysUpdate)
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

	manual.Versions, _ = fetchLastTalosReleases("")
	//fmt.Println(manual.Versions)

	// Template form
	templmanager.RenderTemplate(w, "form_manual.tmpl", manual)

}

func performUpgradeHandler(w http.ResponseWriter, r *http.Request) {
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
	InstalledVersion := r.Form.Get("installed_version")
	TargetVersion := r.Form.Get("target_version")
	//log.Printf("auto_sys_updates : %v", SysUpdate)

	// Préparer un script HTML avec redirection automatique
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
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
		<title>Upgrade in progress</title>
		<meta http-equiv="refresh" content="5;url=/">
	</head>
	<body>
		<!-- (PART A) TOP NAV BAR -->
	<nav id="top">
	<!-- (PART A1) SIDEBAR TOGGLE -->
	<div id="stog" onclick="document.getElementById('side').classList.toggle('mini')">
		&#9776;
	</div>

	<!-- (PART A2) LOGO & WHATEVER ELSE -->

		<h2>Upgrade in progress</h2>
	`)
	report := ReturnMan{
		MachineID:     MachineID,
		TargetVersion: TargetVersion,
	}
	templmanager.RenderTemplate(w, "form_manual_return.tmpl", report)

	// Mise à jour d'un noeud existant
	cmd := exec.Command("talosctl", "upgrade", "-n", MachineID, "--image", TalosImageInstaller+":"+TargetVersion, "--preserve=true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Upgrade error for node %s with version %s: %v", MachineID, TargetVersion, err)
		report := NodeUpdateReport{
			NodeName:          MachineID,
			PreviousVersion:   InstalledVersion,
			ImageSource:       TalosImageInstaller,
			NewVersion:        TargetVersion,
			UpdateStatus:      "Failed",
			AdditionalDetails: "Error during automatic Node Upgrade",
			Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		}

		subject := "Manual Node Upgrade"
		// Generate email body
		emailBody, err := generateUpdateEmailBody(report)
		if err != nil {
			return
		}

		sendMail(subject, emailBody)
		fmt.Fprintf(w, "<p>Error for node %s with version %s: %v</p>", MachineID, TargetVersion, err)
	} else {
		log.Printf("Upgrade successful for node %s with version %s", MachineID, TargetVersion)
		report := NodeUpdateReport{
			NodeName:          MachineID,
			PreviousVersion:   InstalledVersion,
			ImageSource:       TalosImageInstaller,
			NewVersion:        TargetVersion,
			UpdateStatus:      "Success",
			AdditionalDetails: "Node updated successfully without any issues",
			Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		}

		subject := "Manual Node Upgrade"
		// Generate email body
		emailBody, err := generateUpdateEmailBody(report)
		if err != nil {
			return
		}

		sendMail(subject, emailBody)
		fmt.Fprintf(w, "<p>Upgrade successful for node %s with version %s</p>", MachineID, TargetVersion)
	}
	log.Printf("Output: ", string(output))

	log.Printf("Upgrade node %s to version %s", MachineID, TargetVersion)
	//}
	// Redirect to index
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
