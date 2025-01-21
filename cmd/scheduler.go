package main

import (
	"log"
	"math"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// scheduleClusterSync manage member list schedules
func (m *TalosCockpit) scheduleClusterSync(sched time.Duration, endpoint string) {
	ticker := time.NewTicker(sched)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := m.getTalosVersion(endpoint); err != nil {
					log.Printf("Échec de la récupération de la version installée : %v", err)
				}
				_, err := m.listAndStoreClusterMembers(endpoint)
				if err != nil {
					log.Printf("Échec de la synchronisation des membres du cluster : %v", err)
				}

			}
		}
	}()
}

// scheduleClusterSync manage cluster upgrade schedules
func (m *TalosCockpit) scheduleClusterUpgrade(sched time.Duration, endpoint string) {
	ticker := time.NewTicker(sched)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := m.fetchLatestRelease(); err != nil {
					log.Printf("Échec de la récupération de la dernière version : %v", err)
				}

				if err := m.getTalosVersion(endpoint); err != nil {
					log.Printf("Échec de la récupération de la version installée : %v", err)
				}
				// Récupérer dynamiquement l'ID du cluster
				clusterID, err := m.getClusterID(endpoint)
				if err != nil {
					log.Printf("Impossible de récupérer l'ID du cluster")
					return
				}
				members, err := m.getClusterMembers(clusterID)
				if err != nil {
					log.Printf("Erreur de récupération de la liste des membres")
					return
				}
				safeUpgradeDate := m.LatestReleaseDate.AddDate(0, 0, UpgradeSafePeriod)
				log.Printf("No auto upgrade before %v for the latest release %s", safeUpgradeDate, m.LatestOsVersion)

				timeLeft := safeUpgradeDate.Sub(time.Now().UTC())

				for _, member := range members {
					if m.LatestOsVersion != member.InstalledVersion {
						log.Printf("Latest version %s differs from Installed one %s on node %s", m.LatestOsVersion, member.InstalledVersion, member.MachineID)
						if member.SysUpdate {

							if timeLeft > time.Hour*24 {
								days := math.Round(timeLeft.Hours() / 24)
								log.Printf("%v days remaining for a safe Upgrade", days)
							} else if timeLeft >= time.Second {
								log.Printf("%v remainings for a safe Upgrade", timeLeft)
							} else {
								timeLeft := time.Now().UTC().Sub(safeUpgradeDate)
								log.Printf("Safe upgrades available since %v", timeLeft)
								log.Printf("Launching Upgrade schedule")

								if err := m.upgradeSystem(member.Hostname, TalosImageInstaller); err != nil {
									log.Printf("Error during automatic Node Upgrade : %v", err)
									report := NodeUpdateReport{
										NodeName:          member.Hostname,
										PreviousVersion:   member.InstalledVersion,
										ImageSource:       TalosImageInstaller,
										NewVersion:        m.LatestOsVersion,
										UpdateStatus:      "Failed",
										AdditionalDetails: "Error during automatic Node Upgrade",
										Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
									}

									subject := "Automatic Node Upgrade"
									// Generate email body
									emailBody, err := generateUpdateEmailBody(report)
									if err != nil {
										return
									}

									sendMail(subject, emailBody)
								} else {
									log.Printf("Operating system of %s Updated to : %s", member.Hostname, TalosImageInstaller)
									report := NodeUpdateReport{
										NodeName:          member.Hostname,
										PreviousVersion:   member.InstalledVersion,
										ImageSource:       TalosImageInstaller,
										NewVersion:        m.LatestOsVersion,
										UpdateStatus:      "Success",
										AdditionalDetails: "Node updated successfully without any issues",
										Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
									}

									subject := "Automatic Node Upgrade"
									// Generate email body
									emailBody, err := generateUpdateEmailBody(report)
									if err != nil {
										return
									}

									sendMail(subject, emailBody)
								}
							}
						} else {
							log.Printf("Automatic Node Upgrade disabled for node: %s", member.Hostname)
						}
					} else {
						log.Printf("Node %s allready up to date", member.Hostname)
					}
				}
				ctl, _ := m.getNodeIP(endpoint)
				if m.K8sUpdate {
					_, err := m.upgradeKubernetes(ctl, "", "")
					if err != nil {
						log.Printf("Échec de la mise à jour de Kubernetes : %v", err)
					}
				} else {
					log.Printf("Auto Update Kubernetes désactivé pour le cluster: %s", clusterID)
				}
			}
		}
	}()
}
