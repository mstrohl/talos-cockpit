package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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
	MultiPatches string
}

type PatchFault struct {
	Referer string
	Error   error
}

// Render multi patch template
func multiPatchHandler(w http.ResponseWriter, m *TalosCockpit) {
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
	templmanager.RenderTemplate(w, "multi_patch.tmpl", membershtml)

}

// Render single patch template
func patchHandler(w http.ResponseWriter, m *TalosCockpit) {
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

// Apply multiple or single patch Get form data, dry-run patches and apply if confirmed
func performPatchHandler(w http.ResponseWriter, r *http.Request, option string) {
	var m *TalosCockpit
	// Check method is a POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	sourceUri := r.Header.Get("Referer")

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
	MultiPatches := r.Form.Get("multi_patches")
	log.Println("multi : ", MultiPatches)

	var data Patch

	if Operation != "" && Path != "" && Value != "" && MultiPatches == "" {
		cmd := "talosctl -n " + strings.Join(TargetNodes, ",") + " patch machineconfig -p '[{\"op\": \"" + Operation + "\", \"path\": \"" + Path + "\", \"value\": \"" + Value + "\"}]' " + option
		log.Println("Patch command : ", cmd)
		output, err := m.runCommand("bash", "-c", cmd)
		if err != nil {
			log.Println("performPatch - ERROR -", err)
			fault := PatchFault{
				Error:   err,
				Referer: sourceUri,
			}
			templmanager.RenderTemplate(w, "patch_err.tmpl", fault)
			return
		}

		data = Patch{
			TargetFormat: strings.Join(TargetNodes, ","),
			Operation:    Operation,
			Path:         Path,
			Value:        Value,
			Output:       output,
			Opt:          option,
		}
	} else if Operation == "" && Path == "" && Value == "" && MultiPatches != "" {
		log.Println("performPatch -INFO - MultiPatches:" + MultiPatches)
		f, erro := os.Create("/app/multi_patches.yaml")
		if erro != nil {
			log.Println("performPatch - ERROR - CreatingFile - ", erro)
			fault := PatchFault{
				Error:   erro,
				Referer: sourceUri,
			}
			templmanager.RenderTemplate(w, "patch_err.tmpl", fault)
			return
		}
		defer f.Close()
		test, erro := os.Stat("/app/multi_patches.yaml")
		if erro != nil {
			log.Println("performPatch - ERROR - StatFile - ", erro)
			fault := PatchFault{
				Error:   erro,
				Referer: sourceUri,
			}
			templmanager.RenderTemplate(w, "patch_err.tmpl", fault)
			return
		}
		fmt.Println("MultipatchPrint: ", test)

		ln, _ := f.WriteString(MultiPatches)
		fmt.Printf("MultipatchPrint: %v", ln)

		cmd := "talosctl -n " + strings.Join(TargetNodes, ",") + " patch machineconfig -p @/app/multi_patches.yaml " + option
		log.Println("Patch command : ", cmd)
		output, err := m.runCommand("bash", "-c", cmd)
		if err != nil {
			log.Println("performPatch - ERROR -", err)
			fault := PatchFault{
				Error:   err,
				Referer: sourceUri,
			}
			templmanager.RenderTemplate(w, "patch_err.tmpl", fault)
			return
		}

		data = Patch{
			TargetFormat: strings.Join(TargetNodes, ","),
			Output:       output,
			MultiPatches: MultiPatches,
			Opt:          option,
		}

	} else {
		msg := errors.New("performPatch - ERROR - Not Simple Patch neither Multi patches - Operation:" + Operation + "|Path:" + Path + "|Value:" + Value + "|MultiPatches:" + MultiPatches)
		log.Println(msg)
		fault := PatchFault{
			Error:   msg,
			Referer: sourceUri,
		}
		templmanager.RenderTemplate(w, "patch_err.tmpl", fault)
		return
	}

	templmanager.RenderTemplate(w, "patch.tmpl", data)
}
