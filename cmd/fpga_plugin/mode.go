package main

import (
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getModeOverrideFromCluster(kubeconfig, master, mode string) (string, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags(master, kubeconfig)
	}
	if err != nil {
		return mode, err
	}

	nodeName := os.Getenv("NODE_NAME")
	if len(nodeName) == 0 {
		// NODE_NAME is not set so try to use hostname
		hostName, err2 := os.Hostname()
		if err2 != nil {
			return mode, err2
		}
		hostName = strings.TrimSpace(hostName)
		if len(hostName) == 0 {
			return mode, fmt.Errorf("empty system hostname")
		}
		hostName = strings.ToLower(hostName)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return mode, err
	}

	node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return mode, err
	}

	if nodeMode, ok := node.ObjectMeta.Annotations["fpga.intel.com/device-plugin-mode"]; ok {
		fmt.Println("Overriding mode to ", nodeMode)
		return nodeMode, nil
	}

	return mode, nil
}
