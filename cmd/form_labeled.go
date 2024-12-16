package main

import (
	"log"
	"net/http"
	"talos-cockpit/internal/services"
	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
)

type ReturnLabeled struct {
	Label         string
	NodesMatching []string
	TargetVersion string
}

func labeledUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	versions, _ := fetchLastTalosReleases("")
	//fmt.Println(manual.Versions)

	// Template form
	templmanager.RenderTemplate(w, "form_labeled.tmpl", versions)

}

func performLabeledUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	var m *TalosCockpit
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
	Label := r.Form.Get("target_label")
	log.Printf("target_label : %s", Label)
	TargetVersion := r.Form.Get("target_version")
	//log.Printf("target_version : %v", TargetVersion)
	nodeService := services.NewNodeService()

	// List nodes with a specific label
	nodes, _, err := nodeService.ListNodesByLabel(Label)
	if err != nil {
		log.Printf("Failed to list nodes: %v", err)
	}
	report := ReturnLabeled{
		Label:         Label,
		NodesMatching: nodes,
		TargetVersion: TargetVersion,
	}

	templmanager.RenderTemplate(w, "form_labeled_return.tmpl", report)
	// Upgrade group

	m.updateGroupByLabel(Label, TargetVersion)

	log.Printf("Upgrade nodes grouped by label %s to version %s", Label, TargetVersion)
	//}
	// Rediriger vers la liste des utilisateurs
	//http.Redirect(w, r, "/", http.StatusSeeOther)
}
