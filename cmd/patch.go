package main

import (
	"log"
	"net/http"
	"strings"
	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
)

type simplePatch struct {
	Hostname string
}

type Patch struct {
	Hostname     []string
	TargetNodes  []string
	TargetFormat string
	Operation    string
	Path         string
	Value        string
	Output       string
	Opt          string
}

func patchNode(endpoint string, patch string) {

}

func patchHandler(w http.ResponseWriter, r *http.Request, m *TalosCockpit) {
	//log.Printf("INVENTORY - TalosApiEndpoint: %s", TalosApiEndpoint)
	clusterID, err := m.getClusterID(TalosApiEndpoint)
	if err != nil {
		http.Error(w, "Cannot get cluster ID", http.StatusInternalServerError)
		return
	}
	//log.Printf("INVENTORY - clusterID: %s", clusterID)
	members, err := m.getClusterMembers(clusterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var membershtml []simplePatch
	for _, member := range members {
		memberhtml := simplePatch{
			Hostname: member.Hostname,
		}
		membershtml = append(membershtml, memberhtml)

	}

	// Template form
	templmanager.RenderTemplate(w, "simple_patch.tmpl", membershtml)

}

func performPatchHandler(w http.ResponseWriter, r *http.Request, option string) {
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
	TargetNodes := r.Form["target_nodes"]
	log.Println("target_nodes : ", TargetNodes)
	Operation := r.Form.Get("operation")
	log.Println("operation : ", Operation)
	Path := r.Form.Get("path")
	log.Println("path : ", Path)
	Value := r.Form.Get("value")
	log.Println("value : ", Value)

	cmd := "talosctl -n " + strings.Join(TargetNodes, ",") + " patch machineconfig -p '[{\"op\": \"" + Operation + "\", \"path\": \"" + Path + "\", \"value\": \"" + Value + "\"}]' " + option
	log.Println("Patch command : ", cmd)
	output, err := m.runCommand("bash", "-c", cmd)
	if err != nil {
		log.Println("performPatch - ERROR -", err)
		templmanager.RenderTemplate(w, "patch_err.tmpl", err)
		return
	}

	data := Patch{
		TargetFormat: strings.Join(TargetNodes, ","),
		Operation:    Operation,
		Path:         Path,
		Value:        Value,
		Output:       output,
		Opt:          option,
	}

	templmanager.RenderTemplate(w, "patch.tmpl", data)
	// Upgrade group

	//}
	// Rediriger vers la liste des utilisateurs
	//http.Redirect(w, r, "/", http.StatusSeeOther)
}
