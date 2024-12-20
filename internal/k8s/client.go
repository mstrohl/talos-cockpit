package k8s

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Global Kubernetes client manager
type K8sClientManager struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
	once      sync.Once
}

// Singleton instance of the client manager
var (
	clientManagerInstance *K8sClientManager
	initOnce              sync.Once
)

// NewK8sClientManager creates or returns an existing Kubernetes client manager
func NewK8sClientManager() *K8sClientManager {
	initOnce.Do(func() {
		manager, err := initializeK8sClientManager()
		if err != nil {
			log.Fatalf("Failed to initialize Kubernetes client: %v", err)
		}
		clientManagerInstance = manager
	})
	return clientManagerInstance
}

// initializeK8sClientManager handles the creation of Kubernetes client
func initializeK8sClientManager() (*K8sClientManager, error) {
	var config *rest.Config
	var err error

	// Try in-cluster configuration first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			// Default to home directory kubeconfig
			homeDir, _ := os.UserHomeDir()
			kubeconfigPath = filepath.Join(homeDir, ".kube", "talos-kubeconfig")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sClientManager{
		clientset: clientset,
		config:    config,
	}, nil
}

// GetClientset returns the Kubernetes clientset
func (m *K8sClientManager) GetClientset() *kubernetes.Clientset {
	return m.clientset
}

// GetConfig returns the Kubernetes REST config
func (m *K8sClientManager) GetConfig() *rest.Config {
	return m.config
}
