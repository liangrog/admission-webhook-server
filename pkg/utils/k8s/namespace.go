package k8s

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var clientset *kubernetes.Clientset

func init() {
	config, err := GetKubeConfigBasedOnEnv()
	if err != nil {
		panic(err.Error())
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
}

// GetNamespaceAnnotations returns all annotations of namespace
func GetNamespaceAnnotations(namespace string) (map[string]string, error) {
	ns, err := clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	if ns == nil {
		return nil, fmt.Errorf("namespace : %s does not exist", namespace)
	}

	return ns.Annotations, nil
}
