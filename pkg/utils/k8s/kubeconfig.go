package k8s

import (
	"errors"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubeConfigMode modes for fetching kube config
type KubeConfigMode int

// KubeConfigMode Constants
const (
	InCluster        KubeConfigMode = 0
	FromFile         KubeConfigMode = 1
	FromClusterToken KubeConfigMode = 2
)

func (mode KubeConfigMode) String() string {

	modes := [...]string{
		"IN_CLUSTER",
		"FROM_FILE",
		"FROM_CLUSTER_TOKEN"}

	return modes[mode]
}

//GetKubeConfig returns config based on mode
func GetKubeConfig(mode KubeConfigMode) (*rest.Config, error) {

	var config *rest.Config
	var err error = nil

	switch mode {

	case InCluster:
		config, err = rest.InClusterConfig()

	case FromClusterToken:
		config = &rest.Config{}
		config.Insecure = true
		if config.Host = os.Getenv("EKS_API_SERVER"); config.Host == "" {
			err = errors.New("Env var EKS_API_SERVER needs to be set")
		}
		if config.BearerToken = os.Getenv("CLUSTER_TOKEN"); config.BearerToken == "" {
			err = errors.New("Env var CLUSTER_TOKEN needs to be set")
		}

	case FromFile:
		var kubeconfig string

		if kubeconfig = os.Getenv("KUBE_CONFIG_FILE_PATH"); kubeconfig == "" {
			kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

	default:
		err = errors.New("Invalid mode specified")
	}

	return config, err
}

//GetKubeConfigBasedOnEnv returns config based on KUBE_CONFIG_MODE env var
func GetKubeConfigBasedOnEnv() (*rest.Config, error) {

	var kubeConfigMode KubeConfigMode

	switch os.Getenv("KUBE_CONFIG_MODE") {
	case FromFile.String():
		kubeConfigMode = FromFile
	case InCluster.String():
		kubeConfigMode = InCluster
	case FromClusterToken.String():
		kubeConfigMode = FromClusterToken
	default:
		kubeConfigMode = FromClusterToken
	}

	return GetKubeConfig(kubeConfigMode)
}
