package main

import (
	"log"
	"time"

	"github.com/gorhill/cronexpr"
	_ "github.com/mattn/go-sqlite3"
)

var MroUgradeTriggered bool
var Done chan bool

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
	done := make(chan bool)
	log.Println("Upgrade triggered ? ", MroUgradeTriggered)

	go func() {
		for {
			select {
			case <-done:
				log.Println("Ticker Ends")
				return
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
				nodeUpToDate := 0
				nodeCount := len(members)
				for _, member := range members {
					if m.LatestOsVersion != member.InstalledVersion {
						log.Printf("Latest version %s differs from Installed one %s on node %s", m.LatestOsVersion, member.InstalledVersion, member.MachineID)
						if member.SysUpdate {
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
								nodeUpToDate++
								log.Printf("%v/%v node up to date", nodeUpToDate, len(members))
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
						} else {
							nodeUpToDate++
							log.Printf("Automatic Node Upgrade disabled for node: %s", member.Hostname)
							log.Printf("%v/%v node treated", nodeUpToDate, len(members))
						}
					} else {
						nodeUpToDate++
						log.Printf("Node %s allready up to date", member.Hostname)
						log.Printf("%v/%v node treated", nodeUpToDate, len(members))
					}
				}
				ctl, _ := m.getNodeIP(endpoint)
				if m.K8sUpdate {
					_, err := m.upgradeKubernetes(ctl, "", "")
					if err != nil {
						log.Printf("Échec de la mise à jour de Kubernetes : %v", err)
					}
				} else {
					log.Printf("Auto Update Kubernetes disabled for cluster: %s", clusterID)
				}
				// manage Ticker stop
				if nodeUpToDate == nodeCount {
					log.Println("scheduleClusterUpgrade All nodes ard Up to date")
					log.Printf("%v/%v node treated", nodeUpToDate, len(members))
					done <- true
				}

			}
		}
	}()
}

// scheduleClusterSync manage cluster upgrade schedules
func (m *TalosCockpit) scheduleSafeUpgrade(cfg Config) {
	ticker := time.NewTicker(time.Duration(1) * time.Minute)

	log.Println("scheduleSafeUpgrades - Upgrade triggered ? ", MroUgradeTriggered)
	go func() {
		for {
			select {

			case <-ticker.C:

				if Mro >= time.Second && cfg.Schedule.MaintenanceWindow.Cron != "" {
					// get maintenance window size

					// To avoid maintenance overlap, no uprgade will be triggered in the last ten minutes
					safeEnd := time.Minute * 1
					//log.Println(Mro)
					MROCron := cfg.Schedule.MaintenanceWindow.Cron
					nextCron := cronexpr.MustParse(MROCron).Next(time.Now().UTC())
					nextCronEnd := nextCron.Add(Mro)
					db_start, db_end := m.getLastSched()
					beforeStart := db_start.Sub(time.Now().UTC())
					beforeEnd := db_end.Add(-safeEnd).Sub(time.Now().UTC())
					// Use case at database startup
					if db_start.IsZero() {
						m.upsertSchedules(nextCron, nextCronEnd)
						log.Printf("MRO - Maintenance has been planed to start %v until %v\n", nextCron, nextCronEnd)
					} else if beforeStart < time.Second && beforeEnd > time.Minute {
						// Proceed upgrade
						log.Printf("MRO - Maintenance started since %v until %v\n", beforeStart, beforeEnd)
						// Disabled to test MRO + grace period
						//if !MroUgradeTriggered {
						//	MroUgradeTriggered = true
						//	safeUpgrade(m)
						//	log.Println("MRO - Update has been triggered during the maintenance period")
						//} else {
						//	log.Println("MRO - Update already running during the maintenance period")
						//}
						safeUpgrade(m)
						log.Println("MRO - Update has been triggered during the maintenance period")
					} else if beforeEnd < time.Second {
						// End of maintenance has been reached
						// Disabled to test MRO + grace period
						//MroUgradeTriggered = false
						// Upsert database with the next schedule
						m.upsertSchedules(nextCron, nextCronEnd)
						log.Printf("MRO - New maintenance has been planed to start %v until %v\n", nextCron, nextCronEnd)
					} else if beforeStart > time.Second || (beforeStart < time.Second && beforeEnd < time.Minute) {
						// Disabled to test MRO + grace period
						//MroUgradeTriggered = false
						log.Printf("MRO - Waiting for next maintenance planed at %v until %v\n", db_start, db_end)
						log.Println("MRO_Cron: ", MROCron)
						log.Println("MRO_NextStart: ", nextCron)
						log.Println("MRO_NextEnd: ", nextCronEnd)
						log.Println("MRO_DB_NextStart: ", db_start)
						log.Println("MRO_DB_NextEnd: ", db_end)
						log.Println("MRO_DB_timeleft_Start: ", beforeStart)
						log.Println("MRO_DB_timeleft_End: ", beforeEnd)
					} else {
						// Disabled to test MRO + grace period
						//MroUgradeTriggered = false
						log.Println("MRO - Unindentified usecase - Debugging vars")
						log.Println("MRO_Cron: ", MROCron)
						log.Println("MRO_NextStart: ", nextCron)
						log.Println("MRO_NextEnd: ", nextCronEnd)
						log.Println("MRO_DB_NextStart: ", db_start)
						log.Println("MRO_DB_NextEnd: ", db_end)
						log.Println("MRO_DB_timeleft_Start: ", beforeStart)
						log.Println("MRO_DB_timeleft_End: ", beforeEnd)
					}
				} else {
					safeUpgrade(m)
				}
			}
		}
	}()
}
