package main

import (
	"log"
	"net/http"
	"time"

	templmanager "talos-cockpit/internal/tmplmanager"

	_ "github.com/mattn/go-sqlite3"
)

type DashboardData struct {
	ClientIP        string
	ClusterID       string
	LatestOsVersion string
	LastPreRelease  string
	SyncSched       time.Duration
	UpgradeSched    time.Duration
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

		clientIP, err := m.getNodeIP(TalosApiEndpoint)
		if err != nil {
			log.Printf("Échec de la récupération du NodeIP : %v", err)
		}

		DashboardData := DashboardData{
			ClientIP:        clientIP,
			ClusterID:       clusterID,
			LatestOsVersion: m.LatestOsVersion,
			SyncSched:       SyncSched,
			UpgradeSched:    UpgradeSched,
			LastPreRelease:  LastPreRelease,
		}

		templmanager.RenderTemplate(w, "index.tmpl", DashboardData)
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
