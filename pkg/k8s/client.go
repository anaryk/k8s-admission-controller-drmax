package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// PrepareInClusterK8SClient initializes a Kubernetes client for use within a Kubernetes cluster
func PrepareInClusterK8SClient() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating in-cluster config: %w", err)
	}
	return config, nil
}

// PrepareLocalKubeconfigK8SClient initializes a Kubernetes client for use with a local kubeconfig file
func PrepareLocalKubeconfigK8SClient() (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir(), ".kube", "config"))
	if err != nil {
		return nil, fmt.Errorf("error creating kubeconfig: %w", err)
	}
	return config, nil
}

// homeDir returns the home directory for the current user
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
