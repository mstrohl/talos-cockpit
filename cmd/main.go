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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"encoding/json"

	"github.com/google/go-github/v39/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes"
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
	ClusterID        string
	Namespace        string
	Type             string
	MachineID        string
	Hostname         string
	Role             string
	ConfigVersion    json.Number
	LatestOsVersion  string
	InstalledVersion string
	IP               string
	CreatedAt        time.Time
	LastUpdated      time.Time
	SysUpdate        bool
	K8sUpdate        bool
}

// TalosCockpit gère les opérations sur le cluster Talos
type TalosCockpit struct {
	githubClient     *github.Client
	webServer        *http.Server
	db               *sql.DB
	clientset        *kubernetes.Clientset
	ConfigVersion    string
	LatestOsVersion  string
	InstalledVersion string
	clientInfo       string
	SysUpdate        bool
	K8sUpdate        bool
	MailRecipient    string
	MailCc           string
	MailHost         string
	MailUsername     string
	MailPassword     string
}

// LatestGithubVersions
type LatestGithubVersions struct {
	Versions []string
}

// ManualUpdateForm
type ManualUpdateForm struct {
	LatestGithubVersions
	ClusterMember
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

// fetchLatestRelease Get LastTalos version from GitHub
func (m *TalosCockpit) fetchLatestRelease() error {
	ctx := context.Background()
	release, _, err := m.githubClient.Repositories.GetLatestRelease(ctx, "siderolabs", "talos")
	if err != nil {
		return err
	}
	m.LatestOsVersion = release.GetTagName()
	return nil
}

func fetchLastTalosReleases(githubToken string) ([]string, error) {
	// Create a context
	ctx := context.Background()

	// Create an authenticated GitHub client (optional, but recommended to avoid rate limits)
	var client *github.Client
	if githubToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	// Fetch releases for Talos
	releases, _, err := client.Repositories.ListReleases(
		ctx,
		"siderolabs",
		"talos",
		&github.ListOptions{
			Page:    1,
			PerPage: 100, // Fetch more than 5 to ensure we have enough
		},
	)
	if err != nil {
		return nil, err
	}

	// Find the last prerelease
	var prereleaseVersion string
	var stableVersions []string

	for _, release := range releases {
		if release.GetPrerelease() && prereleaseVersion == "" {
			prereleaseVersion = release.GetTagName()
		} else if !release.GetPrerelease() {
			stableVersions = append(stableVersions, release.GetTagName())
		}
	}

	// Sort stable versions in descending order
	sort.Slice(stableVersions, func(i, j int) bool {
		return compareVersions(stableVersions[i], stableVersions[j]) > 0
	})

	// Combine last prerelease (if found) with first 4 stable releases
	var result []string
	if prereleaseVersion != "" {
		result = append(result, prereleaseVersion)
	}

	// Add up to 4 stable releases
	limit := 4
	if len(stableVersions) < limit {
		limit = len(stableVersions)
	}
	result = append(result, stableVersions[:limit]...)

	return result, nil
}

// compareVersions compares two semantic version strings
func compareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split versions into components
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	// Compare each part
	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		n1, _ := strconv.Atoi(v1Parts[i])
		n2, _ := strconv.Atoi(v2Parts[i])

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	// If all parts are equal, longer version is considered greater
	return len(v1Parts) - len(v2Parts)
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
	http.HandleFunc("/req", func(w http.ResponseWriter, r *http.Request) {
		upgradeHandler(w, r, db)
	})
	http.HandleFunc("/manual", func(w http.ResponseWriter, r *http.Request) {
		performUpgradeHandler(w, r)
	})
	http.HandleFunc("/labelform", func(w http.ResponseWriter, r *http.Request) {
		labeledUpgradeHandler(w, r)
	})
	http.HandleFunc("/labelupgrade", func(w http.ResponseWriter, r *http.Request) {
		performLabeledUpgradeHandler(w, r)
	})

	// Créer une nouvelle instance du gestionnaire
	manager, err := NewTalosCockpit(githubToken)
	if err != nil {
		log.Fatalf("Échec de l'initialisation du gestionnaire : %v", err)
	}

	versions, err := fetchLastTalosReleases(githubToken)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Last 5 Talos Releases:")
	for _, version := range versions {
		fmt.Println(version)
	}

	//////////////////////////////////
	// Configs

	var cfg Config
	readFile(&cfg)
	readEnv(&cfg)
	//fmt.Printf("%+v\n", cfg)

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
	//k8scfg := manager.getKubeConfig(TalosApiEndpoint)
	//if err != nil {
	//	log.Fatalf("Impossible de récupérer le kubeconfig du cluster : %v", err)
	//}
	//log.Printf("Get Kubeconfig : %s", k8scfg)

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
	if err := manager.getTalosVersion(TalosApiEndpoint); err != nil {
		log.Fatalf("Échec de la récupération de la version installée : %v", err)
	}

	//kconfig := manager.getKubeConfig(TalosApiEndpoint)
	//if err != nil {
	//	log.Fatalf("Impossible de récupérer le kubeconfig du cluster : %v", err)
	//}
	//log.Printf("Etat du cluster : %s", kconfig)

	// Démarrer le serveur web
	manager.startWebServer()

	// Planifier la synchronisation périodique

	manager.scheduleClusterSync(SyncSched, TalosApiEndpoint)
	manager.scheduleClusterUpgrade(UpgradeSched, TalosApiEndpoint)

	//////////////////////////////////
	// K8S API Calls
	// Récupérer la version actuellement installée

	//Get Nodes du cluster
	//nodes := manager.getNodeByLabel("node-role.kubernetes.io/control-plane=")
	//if err != nil {
	//	log.Fatalf("Impossible de récupérer le kubeconfig du cluster : %v", err)
	//}
	//log.Printf("liste des noeuds avec le label node-role.kubernetes.io/control-plane= : %s", nodes)

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
