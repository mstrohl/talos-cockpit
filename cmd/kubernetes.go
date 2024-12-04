package main

import (
	"flag"
	"log"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

//
// Uncomment to load all auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth"
//
// Or uncomment to load specific auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

// NewKubernetesClient creates a new Kubernetes client
func (m *TalosCockpit) NewKubernetesClient() (*TalosCockpit, error) {
	var config *rest.Config
	var err error
	var cfg Config

	readFile(&cfg)
	readEnv(&cfg)
	if cfg.Kubernetes.ConfigPath != "" {
		kubeconfig = flag.String("kubeconfig", cfg.Kubernetes.ConfigPath, "(optional) absolute path to the kubeconfig file")
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "talos-kubeconfig"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	log.Printf("kubeconfig: %s", cfg.Kubernetes.ConfigPath)
	log.Println("%s\n", kubeconfig)
	// use the current context in kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf(err.Error())
	}
	log.Printf("conf OK")

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("ClientSet %s", err.Error())
	}
	log.Printf("clientSet OK")
	// Create the clientset
	//clientset, err := kubernetes.NewForConfig(config)
	//if err != nil {
	//	return nil, fmt.Errorf("error creating kubernetes clientset: %v", err)
	//}

	return &TalosCockpit{clientset: clientset}, nil
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

//func (m *TalosCockpit) getNodeByLabel(label string) []string {
//	//	var cfg Config
//	//
//	//	readFile(&cfg)
//	//	readEnv(&cfg)
//	//	if cfg.Kubernetes.ConfigPath != "" {
//	//		kubeconfig = flag.String("kubeconfig", cfg.Kubernetes.ConfigPath, "(optional) absolute path to the kubeconfig file")
//	//	} else if home := homedir.HomeDir(); home != "" {
//	//		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "talos-kubeconfig"), "(optional) absolute path to the kubeconfig file")
//	//	} else {
//	//		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
//	//	}
//	//	flag.Parse()
//	//	log.Printf("kubeconfig: %s", cfg.Kubernetes.ConfigPath)
//	//	log.Println("%s\n", kubeconfig)
//	//	// use the current context in kubeconfig
//	//	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
//	//	if err != nil {
//	//		log.Printf(err.Error())
//	//	}
//	//	log.Printf("conf OK")
//	//
//	//	// creates the clientset
//	//	clientset, err := kubernetes.NewForConfig(config)
//	//	if err != nil {
//	//		log.Printf("ClientSet %s", err.Error())
//	//	}
//	//	log.Printf("clientSet OK")
//	for {
//		nodes, err := m.clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{LabelSelector: label})
//		if err != nil {
//			log.Printf(err.Error())
//		}
//		log.Printf("There are %d nodes in the cluster matching the label %s \n", len(nodes.Items), label)
//		//fmt.Printf("List", nodes)
//		var nodeNames []string
//		for _, node := range nodes.Items {
//			log.Printf("%s\n", node.Name)
//			for _, condition := range node.Status.Conditions {
//				log.Printf("\t%s: %s\n", condition.Type, condition.Status)
//			}
//			nodeNames = append(nodeNames, node.Name)
//		}
//		// Examples for error handling:
//		// - Use helper functions e.g. errors.IsNotFound()
//		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
//
//		//time.Sleep(10 * time.Second)
//		return nodeNames
//	}
//
//}
