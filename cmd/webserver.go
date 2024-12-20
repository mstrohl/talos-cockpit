package main

import (
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// startWebServer Start webserver
func (m *TalosCockpit) startWebServer() {

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
