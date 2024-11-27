package main

import (
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// scheduleClusterSync gère la synchronisation périodique du cluster
func (m *TalosVersionManager) scheduleClusterSync() {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := m.fetchLatestRelease(); err != nil {
					log.Printf("Échec de la récupération de la dernière version : %v", err)
				}

				if err := m.getConfigVersion(); err != nil {
					log.Printf("Échec de la récupération de la version installée : %v", err)
				}

				_, err := m.listAndStoreClusterMembers()
				if err != nil {
					log.Printf("Échec de la synchronisation des membres du cluster : %v", err)
				}
				// Récupérer dynamiquement l'ID du cluster
				clusterID, err := m.getClusterID()
				if err != nil {
					log.Printf("Impossible de récupérer l'ID du cluster")
					return
				}
				members, err := m.getClusterMembers(clusterID)
				if err != nil {
					log.Printf("Erreur de récupération de la liste des membres")
					return
				}
				for _, member := range members {
					if m.LatestOsVersion != m.ConfigVersion {

						if member.SysUpdate {
							if err := m.upgradeSystem(member.Hostname); err != nil {
								log.Printf("Échec de la mise à jour du système : %v", err)
							}
						} else {
							log.Printf("Auto Update Sytem désactivé pour le node: %s", member.Hostname)
						}
					}
				}
				ctl, _ := m.getNodeIP()
				if m.K8sUpdate {
					if err := m.upgradeKubernetes(ctl); err != nil {
						log.Printf("Échec de la mise à jour de Kubernetes : %v", err)
					}
				} else {
					log.Printf("Auto Update Kubernetes désactivé pour le cluster: %s", clusterID)
				}

			}
		}
	}()
}
