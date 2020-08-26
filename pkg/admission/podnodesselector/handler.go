/**
 * Mutate pod manifest field nodeSelector with proper key values so the pod can be scheduled to
 * designate nodes.
 */
package podnodesselector

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/trilogy-group/admission-webhook-server/pkg/admission/admit"
	"github.com/trilogy-group/admission-webhook-server/pkg/utils"
	"github.com/trilogy-group/admission-webhook-server/pkg/utils/k8s"
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

	ENV_IGNORE_PODS_WITH_LABELS          = "IGNORE_PODS_WITH_LABELS"
	ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS = "NAMESPACE_ANNOTATIONS_TO_PROCESS"
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

func init() {
	log.Printf("Annotations to process : %s", os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS))
	log.Printf("Labels to ignore : %s", os.Getenv(ENV_IGNORE_PODS_WITH_LABELS))
}

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

	// If pod has atleast one label present that belongs to list of labels to ignore
	// Then it does not make it eligible for adding node selector
	// So we return immediately
	labelsToIgnore, err := getLabelsToIgnore()
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range labelsToIgnore {
		if val, ok := pod.Labels[k]; ok {
			if val == v {
				log.Printf("Not adding node selectors as pod has label : %s=%s", k, v)
				return patches, nil
			}
		}
	}

	// Get which annotations are to be processed from pod's namespace
	// Put values of annotations that are present as Node Selectors
	annotationsToProcess, err := getAnnotationsToProcess()
	if err != nil {
		log.Fatal(err)
	}

	nsAnnotations, err := k8s.GetNamespaceAnnotations(req.Namespace)
	if err != nil {
		log.Fatal(err)
	}

	labelSet := make(labels.Set)
	for i, k := range annotationsToProcess {
		if val, ok := nsAnnotations[k]; ok {
			curSet, err := labels.ConvertSelectorToLabelsMap(val)
			if err != nil {
				log.Fatal(err)
			}
			if labels.Conflicts(curSet, labelSet) {
				return patches, fmt.Errorf("There are conflicting labels specified across namespace annotations")
			}
			labelSet = labels.Merge(curSet, labelSet)
		}
	}

	if labelSet != nil {
		op := "replace"
		if pod.Spec.NodeSelector == nil {
			op = "add"
		}

		if labels.Conflicts(labelSet, labels.Set(pod.Spec.NodeSelector)) {
			return patches, fmt.Errorf("pod node label selector conflicts with its namespace node label selector for pod %s", podName)
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

// Get map of labels that disallows Node Selector to be added to pods (if label is present on pod)
func getLabelsToIgnore() (labels.Set, error) {

	// Don't process if no labels are provided
	if len(os.Getenv(ENV_IGNORE_PODS_WITH_LABELS)) == 0 {
		return nil, nil
	}

	// Converts string in format (x=y,a=b) to map[string]->string
	labelsMap, err := labels.ConvertSelectorToLabelsMap(os.Getenv(ENV_IGNORE_PODS_WITH_LABELS))
	if err != nil {
		return nil, err
	}

	return labelsMap, nil
}

// getAnnotationsToProcess returns list of annotations that is to be watched on namespace
func getAnnotationsToProcess() ([]string, error) {

	// Don't process if it is not set
	if len(os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS)) == 0 {
		return nil, nil
	}

	return strings.Split(os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS), ",")

}
