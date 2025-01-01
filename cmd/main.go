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

	templmanager "talos-cockpit/internal/tmplmanager"

	"encoding/json"

	//httpSwagger "github.com/swaggo/http-swagger"

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
	UpgradeK8SOptions   string
	LastPreRelease      string
	StaticDir           string
	kubeconfig          *string
)

// Cluster contain kubernetes cluster information
type Cluster struct {
	ID        int
	Name      string
	Endpoint  string
	K8sUpdate bool
}

// ClusterMember containing details of a Talos Member
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
}

// TalosCockpit managing cluster operations
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

// filterIPv4Addresses filter IPv4 from list of IP addresses
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

// runCommand exec command wrapper
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
		LastPreRelease = prereleaseVersion
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

// NewTalosCockpit create New Version Manager Instance
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

type Configuration struct {
	LayoutPath  string
	IncludePath string
}

//	@title			Talos Cockpit
//	@version		0.4
//	@description	Golang App To manage Talos cluster With somme API calls

//	@contact.name	STROHL Matthieu
//	@contact.url	https://blog.dive-in-it.com
//	@contact.email	postmaster@dive-in-it.com

// @host		localhost:8080
// @BasePath	/
func main() {

	//////////////////////////////////
	// Configs

	var cfg Config
	readFile(&cfg)
	readEnv(&cfg)
	//fmt.Printf("%+v\n", cfg)

	TalosApiEndpoint = cfg.Talosctl.Endpoint
	// Export Sync loop var
	if cfg.Schedule.SyncMembers != "" {
		ss, err := strconv.Atoi(cfg.Schedule.SyncMembers)
		if err != nil {
			log.Printf("Schedule sync Conversion Error")
		}
		SyncSched = time.Duration(ss) * time.Minute
	} else {
		SyncSched = (5 * time.Minute)
	}

	// Export Upgrade loop var
	if cfg.Schedule.SysUpgrade != "" {
		su, err := strconv.Atoi(cfg.Schedule.SysUpgrade)
		if err != nil {
			log.Printf("Schedule Update Conversion Error")
		}
		UpgradeSched = time.Duration(su) * time.Minute
	} else {
		UpgradeSched = (10 * time.Minute)
	}
	////// CFG Images
	if cfg.Images.Installer != "" {
		TalosImageInstaller = cfg.Images.Installer
	} else {
		TalosImageInstaller = "ghcr.io/siderolabs/installer"
	}
	log.Println("Talos image installer: ", TalosImageInstaller)

	if cfg.Images.CustomRegistryPath != "" {
		r := cfg.Images.CustomRegistryPath
		UpgradeK8SOptions = "--apiserver-image " + r + "/kube-apiserver --controller-manager-image " + r + "/kube-controller-manager --kubelet-image " + r + "/kubelet --scheduler-image " + r + "/kube-scheduler"
	}
	if cfg.Images.KubeProxyEnabled {
		r := cfg.Images.CustomRegistryPath
		UpgradeK8SOptions = UpgradeK8SOptions + " --proxy-image " + r + "/kube-proxy"
	}
	if !cfg.Images.PrePull {
		UpgradeK8SOptions = UpgradeK8SOptions + " --pre-pull-images=false"
	}
	log.Println("K8S upgrade options: ", UpgradeK8SOptions)
	////// CFG Templates
	if cfg.Templates.LayoutPath != "" && cfg.Templates.IncludePath != "" {
		log.Println("layout path: ", cfg.Templates.LayoutPath)
		log.Println("include path: ", cfg.Templates.IncludePath)
		templmanager.SetTemplateConfig(cfg.Templates.LayoutPath, cfg.Templates.IncludePath)
	} else {
		log.Println("Default layout path: ", "/app/templates/layouts")
		log.Println("Default include path: ", "/app/templates/")
		templmanager.SetTemplateConfig("/app/templates/layouts", "/app/templates/")
	}
	////// CFG Static files
	if cfg.Static.Path != "" {
		log.Println("static path: ", cfg.Static.Path)
		StaticDir = cfg.Static.Path
	} else {
		log.Println("Default static path: /app/static")
		StaticDir = "/app/static"
	}
	//////////////////////////////////
	// Templating

	templmanager.LoadTemplates()

	//////////////////////////////////
	// Github

	// Get Githup token from envvars
	githubToken := os.Getenv("GITHUB_TOKEN")
	//if githubToken == "" {
	//	log.Fatal("Github Token is required. Set GITHUB_TOKEN.")
	//}

	//Create local database folder
	dbDir := filepath.Join(os.Getenv("HOME"), ".talos-cockpit")

	// Open or create Database
	dbPath := filepath.Join(dbDir, "talos_clusters.db")
	db, _ := sql.Open("sqlite3", dbPath)

	// Create New Manager Instance
	manager, err := NewTalosCockpit(githubToken)
	if err != nil {
		log.Fatalf("Échec de l'initialisation du gestionnaire : %v", err)
	}

	////////////////////////////
	//  URIs
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(StaticDir))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleIndex(w, manager)
	})

	http.HandleFunc("/inventory", func(w http.ResponseWriter, r *http.Request) {
		handleClusterInventory(w, manager)
	})

	http.HandleFunc("/edit", func(w http.ResponseWriter, r *http.Request) {
		handleNodeEdit(w, r, db)
	})
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		handleNodeDashboard(w, r, manager)
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
		labeledUpgradeHandler(w)
	})
	http.HandleFunc("/labelupgrade", func(w http.ResponseWriter, r *http.Request) {
		performLabeledUpgradeHandler(w, r)
	})
	http.HandleFunc("/spatch", func(w http.ResponseWriter, r *http.Request) {
		patchHandler(w, manager)
	})
	http.HandleFunc("/mpatch", func(w http.ResponseWriter, r *http.Request) {
		multiPatchHandler(w, manager)
	})
	http.HandleFunc("/drypatch", func(w http.ResponseWriter, r *http.Request) {
		performPatchHandler(w, r, "--dry-run")
	})
	http.HandleFunc("/patch", func(w http.ResponseWriter, r *http.Request) {
		performPatchHandler(w, r, "")
	})

	//////////////////////////////////
	// API

	//http.Handle("/swagger/*", httpSwagger.Handler(
	//	httpSwagger.URL("swagger.json"), //The url pointing to API definition
	//))
	//http.Handle("/swagger/", httpSwagger.WrapHandler)

	// Manage auto upgrade enablement
	http.HandleFunc("/api/sysupdate", func(w http.ResponseWriter, r *http.Request) {
		ApiNodeEdit(w, r, db)
	})

	// Fetch last 4 Releases and last Pre-release
	versions, err := fetchLastTalosReleases(githubToken)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	//fmt.Println("Last 5 Talos Releases:")
	log.Println(versions)
	//for _, version := range versions {
	//	log.Printf(version)
	//}

	//////////////////////////////////
	// talos/talosctl Calls

	// TODO: define if there is a usecase to get kubeconfig from talosAPI
	//
	//k8scfg := manager.getKubeConfig(TalosApiEndpoint)
	//if err != nil {
	//	log.Fatalf("Can't get kubeconfig from TalosAPI : %v", err)
	//}
	//log.Printf("Get Kubeconfig : %s", k8scfg)

	// Get cluster ID
	clusterID, err := manager.getClusterID(TalosApiEndpoint)
	if err != nil {
		log.Fatalf("Can't get ClusterID  : %v", err)
	}
	log.Printf("ID du cluster : %s", clusterID)

	// Member sync
	_, err = manager.listAndStoreClusterMembers(TalosApiEndpoint)
	if err != nil {
		log.Fatalf("Member sync failure : %v", err)
	}

	// Fetch last Release
	if err := manager.fetchLatestRelease(); err != nil {
		log.Fatalf("Can't fetch last release  : %v", err)
	}

	// Get Talos Installed version
	if err := manager.getTalosVersion(TalosApiEndpoint); err != nil {
		log.Fatalf("Can't get installed Talos version : %v", err)
	}

	// WebServer Start
	manager.startWebServer()

	// Schedules

	manager.scheduleClusterSync(SyncSched, TalosApiEndpoint)
	manager.scheduleClusterUpgrade(UpgradeSched, TalosApiEndpoint)

	//////////////////////////////////
	// K8S API Calls
	//

	//////////////////////////////////
	// Application Close
	//
	// Waiting for a SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Stop webserver properly
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.webServer.Shutdown(ctx); err != nil {
		log.Printf("Erreur lors de l'arrêt du serveur web : %v", err)
	}
}
