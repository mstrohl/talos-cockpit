package services

import (
	"context"
	"fmt"

	"talos-cockpit/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodService struct {
	clientset *kubernetes.Clientset
}

func NewPodService() *PodService {
	// Use the centralized client manager
	clientManager := k8s.NewK8sClientManager()
	return &PodService{
		clientset: clientManager.GetClientset(),
	}
}

func (s *PodService) ListPodsByLabel(namespace, labelSelector string) error {
	pods, err := s.clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("error listing pods: %v", err)
	}

	for _, pod := range pods.Items {
		fmt.Printf("Pod: %s in Namespace: %s\n", pod.Name, pod.Namespace)
		// Additional processing
	}

	return nil
}
