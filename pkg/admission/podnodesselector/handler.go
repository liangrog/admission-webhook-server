/**
 * Mutate pod manifest field nodeSelector with proper key values so the pod can be scheduled to
 * designate nodes.
 */
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
	handlerName = "PodNodesSelector"

	// Path for kube api server to call
	ENV_POD_NODES_SELECTOR_PATH = "POD_NODES_SELECTOR_PATH"
	podNodesSelectorPath        = "pod-nodes-selector"

	// Configuration for specify nodes to namespace.
	// The string format for each namespace follows https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/labels/labels.go
	// Examples:
	//   namespace:label-name=label-value,label-name=label-value;namespace:label-name=label-value
	ENV_POD_NODES_SELECTOR_CONFIG = "POD_NODES_SELECTOR_CONFIG"

	namespaceSeperator      = ";"
	namespaceLabelSeperator = ":"

	ENV_IGNORED_NAMESPACES = "IGNORED_NAMESPACES"
	ENV_DEFAULT_LABELS     = "DEFAULT_LABELS"
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

// Register handler to server
func Register(mux *http.ServeMux) {
	// Sub path
	pdsPath := filepath.Join(
		admit.GetBasePath(),
		utils.GetEnvVal(ENV_POD_NODES_SELECTOR_PATH, podNodesSelectorPath),
	)

	mux.Handle(
		pdsPath,
		admit.AdmitFuncHandler(handler),
	)

	log.Printf("%s registered using path %s", handlerName, pdsPath)
}

// Handling pod node selector request
func handler(req *v1beta1.AdmissionRequest) ([]admit.PatchOperation, error) {
	if req.Resource != podResource {
		log.Printf("Ignore admission request %s as it's not a pod resource", string(req.UID))
		return nil, nil
	}

	// Parse the Pod object.
	raw := req.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := admit.UniversalDeserializer.Decode(raw, nil, &pod); err != nil {
		return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	}

	// Get the pod name for info
	podName := strings.TrimSpace(pod.Name + " " + pod.GenerateName)

	var patches []admit.PatchOperation

	// Get configuration
	selectors, err := getConfiguredSelectorMap()
	if err != nil {
		log.Fatal(err)
	}

	if ignoreNamespace(req.Namespace, strings.Split(utils.GetEnvVal(ENV_IGNORED_NAMESPACES, ""), ",")) {
		log.Printf("Namespace %s is configured to be ignored, so applying no labels\n", req.Namespace)
	} else {
		labelSet := labels.Set{}
		labelsDefinedForNamespace := false
		if labelSet, labelsDefinedForNamespace = selectors[req.Namespace]; labelsDefinedForNamespace {
			log.Printf("Applying isolation labels to %s", req.Namespace)
		} else {
			log.Printf("Applying default labels to %s\n", req.Namespace)
			labelSet, err = labels.ConvertSelectorToLabelsMap(utils.GetEnvVal(ENV_DEFAULT_LABELS, ""))
			if err != nil {
				log.Fatal(err)
			}
		}
		op := "replace"
		if pod.Spec.NodeSelector == nil {
			op = "add"
		}

		if labels.Conflicts(labelSet, labels.Set(pod.Spec.NodeSelector)) {
			return patches, errors.New(fmt.Sprintf("pod node label selector conflicts with its namespace node label selector for pod %s", podName))
		}

		podNodeSelectorLabels := labels.Merge(labelSet, labels.Set(pod.Spec.NodeSelector))

		patches = append(patches, admit.PatchOperation{
			Op:    op,
			Path:  "/spec/nodeSelector",
			Value: podNodeSelectorLabels,
		})

		log.Printf("%s processed pod %s with selectors: %s",
			handlerName,
			podName,
			fmt.Sprintf("%v", podNodeSelectorLabels),
		)
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

func ignoreNamespace(lookup string, ignored []string) bool {
	for _, val := range ignored {
		if val == lookup {
			return true
		}
	}
	return false
}
