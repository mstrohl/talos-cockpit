package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

//
// Uncomment to load all auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth"
//
// Or uncomment to load specific auth plugins
// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

func (m *TalosCockpit) getNodeByLabel(label string) any {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "talos-kubeconfig"), "(optional) absolute path to the kubeconfig file")
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf(err.Error())
	}
	for {
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{LabelSelector: label})
		if err != nil {
			log.Printf(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(nodes.Items))
		//fmt.Printf("Liste brute", nodes)
		var nodeNames []string
		for _, node := range nodes.Items {
			log.Printf("%s\n", node.Name)
			for _, condition := range node.Status.Conditions {
				log.Printf("\t%s: %s\n", condition.Type, condition.Status)
			}
			nodeNames = append(nodeNames, node.Name)
		}
		// Examples for error handling:
		// - Use helper functions e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message

		//time.Sleep(10 * time.Second)
		return nodeNames
	}

}
