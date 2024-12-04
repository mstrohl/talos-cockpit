package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"encoding/json"

	"github.com/google/go-github/v39/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

var (
	SyncSched           time.Duration
	UpgradeSched        time.Duration
	Syscheckbox         string
	K8scheckbox         string
	TalosApiEndpoint    string
	TalosImageInstaller string
	kubeconfig          *string
)

// Cluster représente les informations de base sur un cluster Kubernetes
type Cluster struct {
	ID       int
	Name     string
	Endpoint string
}

// ClusterMember contient les détails d'un membre du cluster Talos
type ClusterMember struct {
	ClusterID       string
	Namespace       string
	Type            string
	MachineID       string
	Hostname        string
	Role            string
	ConfigVersion   json.Number
	LatestOsVersion string
	IP              string
	CreatedAt       time.Time
	LastUpdated     time.Time
	SysUpdate       bool
	K8sUpdate       bool
}

// TalosCockpit gère les opérations sur le cluster Talos
type TalosCockpit struct {
	githubClient    *github.Client
	webServer       *http.Server
	db              *sql.DB
	ConfigVersion   string
	LatestOsVersion string
	clientInfo      string
	SysUpdate       bool
	K8sUpdate       bool
}

// filterIPv4Addresses filtre et ne conserve que les adresses IPv4 valides
func filterIPv4Addresses(addresses []string) []string {
	var ipv4Addresses []string
	for _, addr := range addresses {
		ip := net.ParseIP(addr)
		if ip != nil && ip.To4() != nil {
			ipv4Addresses = append(ipv4Addresses, addr)
		}
	}
	return ipv4Addresses
}

// runCommand exécute une commande système et retourne sa sortie
func (m *TalosCockpit) runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", string(output), err)
	}
	return string(output), nil
}

// fetchLatestRelease récupère la dernière version de Talos depuis GitHub
func (m *TalosCockpit) fetchLatestRelease() error {
	ctx := context.Background()
	release, _, err := m.githubClient.Repositories.GetLatestRelease(ctx, "siderolabs", "talos")
	if err != nil {
		return err
	}
	m.LatestOsVersion = release.GetTagName()
	return nil
}

// upgradeSystem effectue la mise à jour du système Talos
func (m *TalosCockpit) upgradeSystem(node string, installerImage string) error {
	log.Printf("Launch Sytem upgrade to %s for node %s", m.LatestOsVersion, node)
	log.Printf("talosctl upgrade -n %s --image %s:%s --preserve=true", node, installerImage, m.LatestOsVersion)
	_, err := m.runCommand(
		"talosctl",
		"upgrade",
		"-n", node,
		"--image", installerImage+":"+m.LatestOsVersion,
		"--preserve=true",
	)
	return err
}

// upgradeKubernetes effectue la mise à jour de Kubernetes
func (m *TalosCockpit) upgradeKubernetes(controller string) error {
	_, err := m.runCommand(
		"talosctl",
		"upgrade-k8s",
		"-n", controller,
		"--to", m.LatestOsVersion,
	)
	log.Printf("talosctl upgrade-k8s -n %s --to %s", controller, m.LatestOsVersion)
	return err
}

// NewTalosCockpit crée une nouvelle instance du gestionnaire de versions
func NewTalosCockpit(githubToken string) (*TalosCockpit, error) {
	if githubToken != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		manager := &TalosCockpit{
			githubClient: github.NewClient(tc),
		}
		if err := manager.initDatabase(); err != nil {
			return nil, err
		}
		return manager, nil
	} else {
		manager := &TalosCockpit{
			githubClient: github.NewClient(nil),
		}
		if err := manager.initDatabase(); err != nil {
			return nil, err
		}
		return manager, nil
	}

}

// Fonction principale qui initialise et démarre le gestionnaire de cluster
func main() {

	//////////////////////////////////
	// Github

	// Récupérer le token GitHub depuis l'environnement
	githubToken := os.Getenv("GITHUB_TOKEN")
	//if githubToken == "" {
	//	log.Fatal("Un token GitHub est requis. Définissez GITHUB_TOKEN.")
	//}

	// Créer le répertoire pour la base de données
	dbDir := filepath.Join(os.Getenv("HOME"), ".talos-cockpit")

	// Ouvrir ou créer la base de données
	dbPath := filepath.Join(dbDir, "talos_clusters.db")
	db, _ := sql.Open("sqlite3", dbPath)
	http.HandleFunc("/edit", func(w http.ResponseWriter, r *http.Request) {
		handleNodeEdit(w, r, db)
	})
	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		handleNodeUpdate(w, r, db)
	})

	// Créer une nouvelle instance du gestionnaire
	manager, err := NewTalosCockpit(githubToken)
	if err != nil {
		log.Fatalf("Échec de l'initialisation du gestionnaire : %v", err)
	}

	//////////////////////////////////
	// Configs

	var cfg Config
	readFile(&cfg)
	readEnv(&cfg)
	fmt.Printf("%+v\n", cfg)

	TalosApiEndpoint = cfg.Talosctl.Endpoint

	if cfg.Schedule.SyncMembers != "" {
		ss, err := strconv.Atoi(cfg.Schedule.SyncMembers)
		if err != nil {
			log.Printf("Schedule sync Conversion Error")
		}
		SyncSched = time.Duration(ss) * time.Minute
	} else {
		SyncSched = (5 * time.Minute)
	}
	if cfg.Schedule.SysUpgrade != "" {
		su, err := strconv.Atoi(cfg.Schedule.SysUpgrade)
		if err != nil {
			log.Printf("Schedule Update Conversion Error")
		}
		UpgradeSched = time.Duration(su) * time.Minute
	} else {
		UpgradeSched = (10 * time.Minute)
	}

	if cfg.Images.Installer != "" {
		TalosImageInstaller = cfg.Images.Installer
	} else {
		TalosImageInstaller = "ghcr.io/siderolabs/installer"
	}
	//fmt.Printf("Talos Endpoint: %s \n", TalosApiEndpoint)

	//////////////////////////////////
	// talos/talosctl Calls

	//Endpoints du cluster
	//endpoint := GetEndpoints()
	//if err != nil {
	//	log.Fatalf("Impossible de les endpoint du cluster : %v", err)
	//}
	//log.Printf("Get Endpoints : %s", endpoint)

	//kubeconfig du cluster
	k8scfg := manager.getKubeConfig(TalosApiEndpoint)
	if err != nil {
		log.Fatalf("Impossible de récupérer le kubeconfig du cluster : %v", err)
	}
	log.Printf("Get Kubeconfig : %s", k8scfg)

	// Récupérer l'ID du cluster
	clusterID, err := manager.getClusterID(TalosApiEndpoint)
	if err != nil {
		log.Fatalf("Impossible de récupérer l'ID du cluster : %v", err)
	}
	log.Printf("ID du cluster : %s", clusterID)

	// Synchroniser les membres du cluster
	_, err = manager.listAndStoreClusterMembers(TalosApiEndpoint)
	if err != nil {
		log.Fatalf("Échec de la synchronisation initiale des membres du cluster : %v", err)
	}

	// Récupérer la dernière version disponible
	if err := manager.fetchLatestRelease(); err != nil {
		log.Fatalf("Échec de la récupération de la dernière version : %v", err)
	}

	// Récupérer la version actuellement installée
	if err := manager.getConfigVersion(TalosApiEndpoint); err != nil {
		log.Fatalf("Échec de la récupération de la version installée : %v", err)
	}

	kconfig := manager.getKubeConfig(TalosApiEndpoint)
	if err != nil {
		log.Fatalf("Impossible de récupérer le kubeconfig du cluster : %v", err)
	}
	log.Printf("Etat du cluster : %s", kconfig)

	// Démarrer le serveur web
	manager.startWebServer()

	// Planifier la synchronisation périodique

	manager.scheduleClusterSync(SyncSched, TalosApiEndpoint)
	manager.scheduleClusterUpgrade(UpgradeSched, TalosApiEndpoint)

	//////////////////////////////////
	// K8S API Calls

	//Get Nodes du cluster
	nodes := manager.getNodeByLabel("node-role.kubernetes.io/control-plane=")
	if err != nil {
		log.Fatalf("Impossible de récupérer le kubeconfig du cluster : %v", err)
	}
	log.Printf("liste des noeuds avec le label node-role.kubernetes.io/control-plane= : %s", nodes)

	// Attendre un signal d'interruption
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Arrêter proprement le serveur web
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.webServer.Shutdown(ctx); err != nil {
		log.Printf("Erreur lors de l'arrêt du serveur web : %v", err)
	}
}
