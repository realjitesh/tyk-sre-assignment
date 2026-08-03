package main

import (
	"flag"
	"fmt"
	"net/http"
	"log"
    "encoding/json"
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type DeploymentHealth struct {
    Namespace string `json:"namespace"`
    Name      string `json:"name"`
    Desired   int32  `json:"desired"`
    Ready     int32  `json:"ready"`
    Healthy   bool   `json:"healthy"`
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig, leave empty for in-cluster")
	listenAddr := flag.String("address", ":8080", "HTTP server listen address")

	flag.Parse()

	kConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(kConfig)
	if err != nil {
		panic(err)
	}

	version, err := getKubernetesVersion(clientset)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Connected to Kubernetes %s\n", version)

	if err := startServer(*listenAddr, clientset); err != nil {
		panic(err)
	}
}

// getKubernetesVersion returns a string GitVersion of the Kubernetes server defined by the clientset.
//
// If it can't connect an error will be returned, which makes it useful to check connectivity.
func getKubernetesVersion(clientset kubernetes.Interface) (string, error) {
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}

	return version.String(), nil
}

func getDeploymentHealth(clientset kubernetes.Interface) ([]DeploymentHealth, error) {
	deployments := []DeploymentHealth{}
	deploymentList, err := clientset.
    AppsV1().
    Deployments(metav1.NamespaceAll).
    List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, deployment := range deploymentList.Items {
		health := DeploymentHealth{
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
			Desired:   *deployment.Spec.Replicas,
			Ready:     deployment.Status.ReadyReplicas,
			Healthy:   *deployment.Spec.Replicas == deployment.Status.ReadyReplicas,
		}
		deployments = append(deployments, health)
	}

	return deployments, nil
}


// startServer launches an HTTP server with defined handlers and blocks until it's terminated or fails with an error.
//
// Expects a listenAddr to bind to.
func startServer(listenAddr string, clientset kubernetes.Interface) error {
	http.HandleFunc("/healthz", healthHandler(clientset))
	http.HandleFunc("/deployments", deploymentHandler(clientset))

	fmt.Printf("Server listening on %s\n", listenAddr)

	return http.ListenAndServe(listenAddr, nil)
}


func healthHandler(clientset kubernetes.Interface) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
	
    kubernetesVersion, err := getKubernetesVersion(clientset)
	if err != nil {
		log.Printf("Error getting Kubernetes version: %v\n", err)
		http.Error(w, "Failed to get Kubernetes version", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)

	_, err = fmt.Fprintf(w, "Kubernetes version: %s\n", kubernetesVersion)
	if err != nil {
		log.Printf("failed writing to response")
	}
    }
}

func deploymentHandler(clientset kubernetes.Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deployments, err := getDeploymentHealth(clientset)
		if err != nil {
			log.Printf("Error getting deployment health: %v", err)
			http.Error(w, "Failed to get deployment health", http.StatusServiceUnavailable)
			return
		}
        w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(deployments)
		if err != nil {
			log.Printf("failed writing response: %v", err)
		}
	}
}

// healthHandler responds with the health status of the application.
// func healthHandler(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusOK)

// 	_, err := w.Write([]byte("ok"))
// 	if err != nil {
// 		fmt.Println("failed writing to response")
// 	}
// }
