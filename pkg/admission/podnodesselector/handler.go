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

	namespaceAnnotationSeperator  = ","
	blacklistedNamespaceSeperator = ","

	ENV_IGNORE_PODS_WITH_LABELS          = "IGNORE_PODS_WITH_LABELS"
	ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS = "NAMESPACE_ANNOTATIONS_TO_PROCESS"
	ENV_BLACKLISTED_NAMESPACES           = "BLACKLISTED_NAMESPACES"
)

var (
	podResource           = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
	labelsToIgnore        labels.Set
	annotationsToProcess  []string
	blacklistedNamespaces []string
)

func init() {
	log.Printf("Annotations to process : %s", os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS))
	log.Printf("Labels to ignore : %s", os.Getenv(ENV_IGNORE_PODS_WITH_LABELS))

	var err error
	if labelsToIgnore, err = getLabelsToIgnore(); err != nil {
		panic(err.Error())
	}

	if annotationsToProcess, err = getAnnotationsToProcess(); err != nil {
		panic(err.Error())
	}

	if blacklistedNamespaces, err = getBlacklistedNamespaces(); err != nil {
		panic(err.Error())
	}
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

	// If pod belongs to blacklisted namespaces then return immediately
	for _, item := range blacklistedNamespaces {
		if item == req.Namespace {
			log.Printf("Not adding node selectors as pod : %s belongs to namespace : %s", podName, req.Namespace)
			return patches, nil
		}
	}

	// If pod has atleast one label present that belongs to list of labels to ignore
	// Then it does not make it eligible for adding node selector
	// So we return immediately
	for k, v := range labelsToIgnore {
		if val, ok := pod.Labels[k]; ok {
			if val == v {
				log.Printf("Not adding node selectors as pod : %s has label : %s=%s", podName, k, v)
				return patches, nil
			}
		}
	}

	nsAnnotations, err := k8s.GetNamespaceAnnotations(req.Namespace)
	if err != nil {
		return patches, fmt.Errorf("Could not get annotations for namespace : %s because : %v", req.Namespace, err)
	}

	// Find which annotations are to be processed from pod's namespace
	// Prepare labels map from the values of the annotations that are present
	labelSet := make(labels.Set)
	for _, k := range annotationsToProcess {
		if val, ok := nsAnnotations[k]; ok {
			curSet, err := labels.ConvertSelectorToLabelsMap(val)
			if err != nil {
				return patches, fmt.Errorf("Could not process value of annotation : %s for namespace : %s", k, req.Namespace)
			}
			if labels.Conflicts(curSet, labelSet) {
				return patches, fmt.Errorf("There are conflicting labels specified across namespace annotations for %s", req.Namespace)
			}
			labelSet = labels.Merge(curSet, labelSet)
		}
	}

	// Prepare a patch that adds labels map as node selectors
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

// getLabelsToIgnore returns map of labels that disallows Node Selector to be added to pods
func getLabelsToIgnore() (labels.Set, error) {

	// Don't process if it is not set
	if os.Getenv(ENV_IGNORE_PODS_WITH_LABELS) == "" {
		return nil, nil
	}

	labelSet, err := labels.ConvertSelectorToLabelsMap(os.Getenv(ENV_IGNORE_PODS_WITH_LABELS))
	if err != nil {
		return nil, err
	}

	return labelSet, nil
}

// getAnnotationsToProcess returns list of annotations that is to be watched on namespace
func getAnnotationsToProcess() ([]string, error) {

	// Don't process if it is not set
	if os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS) == "" {
		return nil, nil
	}

	annotations := strings.Split(os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS), namespaceAnnotationSeperator)

	return annotations, nil
}

// getBlacklistedNamespaces returns list of namespaces that are blacklisted so pods belong to this namespaces won't be processed
func getBlacklistedNamespaces() ([]string, error) {

	// Don't process if it is not set
	if os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS) == "" {
		return nil, nil
	}

	namespaces := strings.Split(os.Getenv(ENV_BLACKLISTED_NAMESPACES), blacklistedNamespaceSeperator)

	return namespaces, nil
}
