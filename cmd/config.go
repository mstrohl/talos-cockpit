package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Global struct {
		Debug bool `yaml:"debug" envconfig:"COCKPIT_DEBUG"`
	} `yaml:"global"`
	Images struct {
		CustomRegistryPath string `yaml:"custom_registry" envconfig:"COCKPIT_CUSTOM_REGISTRY"`
		Installer          string `yaml:"installer" envconfig:"TALOS_IMAGE_INSTALLER"`
		//Kubelet: string `yaml:"kubelet" envconfig:"K8S_IMAGE_KUBELET"`
		//ApiServer: string `yaml:"apiserver" envconfig:"K8S_IMAGE_APISERVER"`
		//ControllerManager: string `yaml:"controller_manager" envconfig:"K8S_IMAGE_CONTROLLER_MANAGER"`
		//Scheduler: string `yaml:"scheduler" envconfig:"K8S_IMAGE_SCHEDULER"`
		KubeProxyEnabled bool `yaml:"kube_proxy_enables" envconfig:"K8S_PROXY_ENABLED"`
		PrePull          bool `yaml:"prepull" envconfig:"K8S_IMAGE_PREPULL"`
	} `yaml:"images"`
	Schedule struct {
		MaintenanceWindow struct {
			Duration float32 `yaml:"duration" envconfig:"COCKPIT_MRO_WINDOW_SIZE"`
			Cron     string  `yaml:"cron" envconfig:"COCKPIT_MRO_CRON_WINDOW"`
			//Daily    string `yaml:"daily_cron" envconfig:"COCKPIT_MRO_DAILY_WINDOW"`
			//Weekly   string `yaml:"weekly_cron" envconfig:"COCKPIT_MRO_WEEKLY_WINDOW"`
			//Biweekly string `yaml:"biweekly_cron" envconfig:"COCKPIT_MRO_BIWEEKLY_WINDOW"`
			//Monthly  string `yaml:"monthly_cron" envconfig:"COCKPIT_MRO_MONTHLY_WINDOW"`
		} `yaml:"mro_window"`
		SyncMembers       string `yaml:"sync_members" envconfig:"COCKPIT_SCHED_SYNC"`
		SysUpgrade        string `yaml:"sys_upgrade" envconfig:"COCKPIT_SCHED_SYS_UPGRADE"`
		UpgradeSafePeriod int    `yaml:"upgrade_safe_period" envconfig:"COCKPIT_UPGRADE_SAFE_PERIOD"`
	} `yaml:"schedule"`
	Talosctl struct {
		Endpoint string `yaml:"endpoint" envconfig:"TALOS_API_ENDPOINT"`
	} `yaml:"talosctl"`
	Kubernetes struct {
		ConfigPath string `yaml:"config" envconfig:"KUBECONFIG"`
	} `yaml:"kubernetes"`
	//Database struct {
	//	Username string `yaml:"user" envconfig:"DB_USERNAME"`
	//	Password string `yaml:"pass" envconfig:"DB_PASSWORD"`
	//} `yaml:"database"`
	Notifications struct {
		Mail struct {
			Recipient string `yaml:"recipient" envconfig:"MAIL_RECIPIENT"`
			Cc        string `yaml:"cc" envconfig:"MAIL_CC"`
			Host      string `yaml:"host" envconfig:"MAIL_HOST"`
			User      string `yaml:"username" envconfig:"MAIL_USERNAME"`
			Password  string `yaml:"password" envconfig:"MAIL_PASSWORD"`
		} `yaml:"mail"`
	} `yaml:"notifications"`
	Templates struct {
		LayoutPath  string `yaml:"layout_path" envconfig:"TMPL_LAYOUT_PATH"`
		IncludePath string `yaml:"include_path" envconfig:"TMPL_INCLUDE_PATH"`
	} `yaml:"templates"`
	Static struct {
		Path string `yaml:"path" envconfig:"STATIC_PATH"`
	} `yaml:"static"`
}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func readFile(cfg *Config) {
	var cfgPath string
	if os.Getenv("COCKPIT_CONFIG") != "" {
		cfgPath = os.Getenv("COCKPIT_CONFIG")
	} else {
		cfgPath = "/app/config.yml"
	}
	log.Printf(cfgPath)
	if _, err := os.Stat(cfgPath); err == nil {
		f, err := os.Open(cfgPath)
		if err != nil {
			processError(err)
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(cfg)
		if err != nil {
			processError(err)
		}
	} else {
		log.Printf("Config file not found at : /app/config.yml")
	}
}

func readEnv(cfg *Config) {
	err := envconfig.Process("", cfg)
	if err != nil {
		processError(err)
	}
}
