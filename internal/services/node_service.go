package services

import (
	"context"
	"fmt"
	"log"

	"talos-cockpit/internal/k8s"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NodeService struct {
	clientset *kubernetes.Clientset
}

// NewNodeService Create new k8s client for node data management
func NewNodeService() *NodeService {
	// Use the centralized client manager
	clientManager := k8s.NewK8sClientManager()
	return &NodeService{
		clientset: clientManager.GetClientset(),
	}
}

// ListNodesByLabel get a list of node matching label
func (s *NodeService) ListNodesByLabel(labelSelector string) ([]string, []v1.Node, error) {
	nodes, err := s.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error listing nodes: %v", err)
	}

	var nodeNames []string
	//var nodeData []v1.Node
	for _, node := range nodes.Items {
		log.Printf("%s\n", node.Name)
		for _, condition := range node.Status.Conditions {
			log.Printf("\t%s: %s\n", condition.Type, condition.Status)
			//if condition.Status != "False" {
			//	nodeAlerts = append(nodeAlerts, condition.Type)
			//}
		}
		nodeNames = append(nodeNames, node.Name)
	}

	return nodeNames, nodes.Items, err
}
