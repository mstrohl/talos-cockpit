package k8s

import (
	"os"
	"path/filepath"
)

// GetKubeConfigPath determines the kubeconfig path
func GetKubeConfigPath() string {
	// Priority:
	// 1. KUBECONFIG environment variable
	// 2. Default kubeconfig in home directory
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath != "" {
		return kubeconfigPath
	}

	// Fallback to default kubeconfig
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kube", "config")
}
