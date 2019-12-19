package podnodesselector

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/liangrog/admission-webhook-server/pkg/admission/admit"
	"github.com/liangrog/admission-webhook-server/pkg/utils"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	ENV_POD_NODES_SELECTOR_PATH = "POD_NODES_SELECTOR_PATH"
	podNodesSelectorPath        = "pod-nodes-selector"

	// Configuration for specify nodes to namespace.
	// The string format for each namespace follows https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/labels/labels.go
	// Examples:
	//   namespace:label-name=label-value,label-name=label-value;namespace:label-name=label-value
	ENV_POD_NODES_SELECTOR_CONFIG = "POD_NODES_SELECTOR_CONFIG"

	namespaceSeperator      = ";"
	namespaceLabelSeperator = ":"
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

// Register handler
func Register(mux *http.ServeMux) {
	pdsPath := filepath.Join(
		admit.GetBasePath(),
		utils.GetEnvVal(ENV_POD_NODES_SELECTOR_PATH, podNodesSelectorPath),
	)

	mux.Handle(
		pdsPath,
		admit.AdmitFuncHandler(handler),
	)
}

// Handling pod node selector request
func handler(req *v1beta1.AdmissionRequest) ([]admit.PatchOperation, error) {
	if req.Resource != podResource {
		log.Printf("expect resource to be %s", podResource)
		return nil, nil
	}

	// Parse the Pod object.
	raw := req.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := admit.UniversalDeserializer.Decode(raw, nil, &pod); err != nil {
		return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	}

	var patches []admit.PatchOperation

	selectors, err := getConfiguredSelectorMap()
	if err != nil {
		log.Fatal(err)
	}

	if selectors != nil {
		if labelSet, ok := selectors[req.Namespace]; ok {
			op := "replace"
			if pod.Spec.NodeSelector == nil {
				op = "add"
			}

			if labels.Conflicts(labelSet, labels.Set(pod.Spec.NodeSelector)) {
				return patches, errors.New(fmt.Sprintf("pod node label selector conflicts with its namespace node label selector for pod %s", pod.Name))
			}
			podNodeSelectorLabels := labels.Merge(labelSet, labels.Set(pod.Spec.NodeSelector))

			patches = append(patches, admit.PatchOperation{
				Op:    op,
				Path:  "/spec/nodeselector",
				Value: fmt.Sprintf("%s", podNodeSelectorLabels),
			})
		}
	}

	return patches, nil
}

// Get configuration map
func getConfiguredSelectorMap() (map[string]labels.Set, error) {
	// Don't process if no configuration is set
	if len(os.Getenv(ENV_POD_NODES_SELECTOR_CONFIG)) == 0 {
		return nil, nil
	}

	selectors := make(map[string]labels.Set)
	for _, ns := range strings.Split(os.Getenv(ENV_POD_NODES_SELECTOR_CONFIG), namespaceSeperator) {
		conf := strings.Split(ns, namespaceLabelSeperator)

		// If no namespace name or label not set, move on
		if len(conf) != 2 || len(conf[0]) == 0 || len(conf[1]) == 0 {
			continue
		}

		set, err := labels.ConvertSelectorToLabelsMap(conf[1])
		if err != nil {
			return nil, err
		}

		selectors[conf[0]] = set
	}

	return selectors, nil
}
