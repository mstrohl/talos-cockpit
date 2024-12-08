package services

import (
	"context"
	"fmt"
	"log"

	"talos-cockpit/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NodeService struct {
	clientset *kubernetes.Clientset
}

func NewNodeService() *NodeService {
	// Use the centralized client manager
	clientManager := k8s.NewK8sClientManager()
	return &NodeService{
		clientset: clientManager.GetClientset(),
	}
}

func (s *NodeService) ListNodesByLabel(labelSelector string) ([]string, error) {
	nodes, err := s.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %v", err)
	}

	var nodeNames []string
	for _, node := range nodes.Items {
		log.Printf("%s\n", node.Name)
		for _, condition := range node.Status.Conditions {
			log.Printf("\t%s: %s\n", condition.Type, condition.Status)
		}
		nodeNames = append(nodeNames, node.Name)
	}

	return nodeNames, err
}
