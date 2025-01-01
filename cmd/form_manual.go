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

// Render on demand upgrade template
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

// Apply on demand node upgrade
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

	report := ReturnMan{
		MachineID:     MachineID,
		TargetVersion: TargetVersion,
	}
	templmanager.RenderTemplate(w, "form_manual_return.tmpl", report)

	// Update Existing node
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
	log.Println("Output: ", string(output))

	log.Printf("Upgrade node %s to version %s", MachineID, TargetVersion)
	//}
	// Redirect to index
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
