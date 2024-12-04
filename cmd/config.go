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
		Installer string `yaml:"installer" envconfig:"TALOS_IMAGE_INSTALLER"`
	} `yaml:"images"`
	Schedule struct {
		SyncMembers string `yaml:"sync_members" envconfig:"COCKPIT_SCHED_SYNC"`
		SysUpgrade  string `yaml:"sys_upgrade" envconfig:"COCKPIT_SCHED_SYS_UPGRADE"`
	} `yaml:"schedule"`
	Talosctl struct {
		Endpoint string `yaml:"endpoint" envconfig:"TALOS_API_ENDPOINT"`
	} `yaml:"talosctl"`
	Kubernetes struct {
		ConfigPath string `yaml:"config" envconfig:"KUBECONFIG"`
	} `yaml:"kubernetes"`
	Database struct {
		Username string `yaml:"user" envconfig:"DB_USERNAME"`
		Password string `yaml:"pass" envconfig:"DB_PASSWORD"`
	} `yaml:"database"`
	Notifications struct {
		Mail struct {
			Recipient string `yaml:"recipient" envconfig:"MAIL_RECIPIENT"`
			Cc        string `yaml:"cc" envconfig:"MAIL_CC"`
			Host      string `yaml:"host" envconfig:"MAIL_HOST"`
			User      string `yaml:"username" envconfig:"MAIL_USERNAME"`
			Password  string `yaml:"password" envconfig:"MAIL_PASSWORD"`
		} `yaml:"mail"`
	} `yaml:"notifications"`
}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func readFile(cfg *Config) {
	if _, err := os.Stat("/app/config.yml"); err == nil {
		f, err := os.Open("/app/config.yml")
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
