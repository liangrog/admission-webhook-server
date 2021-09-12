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

	"github.com/scylladb/go-set/strset"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/admit"
	"github.com/trilogy-group/admission-webhook-server/pkg/utils"
	"github.com/trilogy-group/admission-webhook-server/pkg/utils/k8s"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

//LabelsMap is a map of label-key:[value1, value2]
type LabelsMap map[string]*strset.Set

const (
	handlerName = "PodNodesSelector"

	// Path for kube api server to call
	ENV_POD_NODES_SELECTOR_PATH = "POD_NODES_SELECTOR_PATH"
	podNodesSelectorPath        = "pod-nodes-selector"

	commaSeperator = ","

	ENV_IGNORE_PODS_WITH_LABELS          = "IGNORE_PODS_WITH_LABELS"
	ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS = "NAMESPACE_ANNOTATIONS_TO_PROCESS"
	ENV_BLACKLISTED_NAMESPACES           = "BLACKLISTED_NAMESPACES"
)

var (
	podResource           = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
	labelsToIgnore        LabelsMap
	annotationsToProcess  []string
	blacklistedNamespaces []string
)

func init() {
	log.Printf("Annotations to process : %s", os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS))
	log.Printf("Blacklisted namespaces : %s", os.Getenv(ENV_BLACKLISTED_NAMESPACES))
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

	// If pod is being controlled by Daemonset then do not add node selectors
	for _, ownerReference := range pod.GetOwnerReferences() {
		if ownerReference.Kind == "DaemonSet" {
			log.Printf("Not adding node selectors as pod : %s is controlled by DaemonSet", podName)
			return patches, nil
		}
	}

	// If pod belongs to blacklisted namespaces then do not add node selectors
	for _, namespace := range blacklistedNamespaces {
		if namespace == req.Namespace {
			log.Printf("Not adding node selectors as pod : %s belongs to namespace : %s", podName, req.Namespace)
			return patches, nil
		}
	}

	// If pod has atleast one label present that belongs to list of labels to ignore
	// Then it does not make it eligible for adding node selector
	for k, v := range labelsToIgnore {
		if val, ok := pod.Labels[k]; ok {
			if v.Has(val) {
				log.Printf("Not adding node selectors as pod : %s has label : %s=%s", podName, k, val)
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
				return patches,
					fmt.Errorf("There are conflicting labels specified across namespace annotations for %s", req.Namespace)
			}
			labelSet = labels.Merge(curSet, labelSet)
		}
	}

	// Prepare a patch that adds labels map as node selectors
	patches, shouldReturn, returnValue, returnValue1 := addNodeLabels(labelSet, pod, patches, podName)
	if shouldReturn {
		return returnValue, returnValue1
	}

	return patches, nil
}

func addNodeLabels(labelSet labels.Set,
	pod corev1.Pod,
	patches []admit.PatchOperation,
	podName string) ([]admit.PatchOperation, bool, []admit.PatchOperation, error) {
	if labelSet != nil {
		op := "replace"
		if pod.Spec.NodeSelector == nil {
			op = "add"
		}

		if labels.Conflicts(labelSet, labels.Set(pod.Spec.NodeSelector)) {
			return nil, true, patches, fmt.Errorf(`pod node label selector conflicts with its 
			namespace node label selector for pod %s`, podName)
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
	return patches, false, nil, nil
}

// getLabelsToIgnore returns map of labels that disallows Node Selector to be added to pods
func getLabelsToIgnore() (LabelsMap, error) {
	// Don't process if it is not set
	if os.Getenv(ENV_IGNORE_PODS_WITH_LABELS) == "" {
		return nil, nil
	}

	labelsMap, err := convertToLabelsMap(os.Getenv(ENV_IGNORE_PODS_WITH_LABELS))
	if err != nil {
		return nil, err
	}

	return labelsMap, nil
}

// getAnnotationsToProcess returns list of annotations that is to be watched on namespace
func getAnnotationsToProcess() ([]string, error) {
	// Don't process if it is not set
	if os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS) == "" {
		return nil, nil
	}

	annotations := strings.Split(os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS), commaSeperator)

	return annotations, nil
}

// getBlacklistedNamespaces returns list of namespaces
// that are blacklisted so pods belong to this namespaces won't be processed
func getBlacklistedNamespaces() ([]string, error) {
	// Don't process if it is not set
	if os.Getenv(ENV_NAMESPACE_ANNOTATIONS_TO_PROCESS) == "" {
		return nil, nil
	}

	namespaces := strings.Split(os.Getenv(ENV_BLACKLISTED_NAMESPACES), commaSeperator)

	return namespaces, nil
}

//convertToLabelsMap converts comma separated labels (k1=v1,k2=v2,k1=v3) to LabelsMap (k1=[v1,v3],k2=[v2]))
func convertToLabelsMap(expression string) (LabelsMap, error) {
	labelsMap := LabelsMap{}

	if len(expression) == 0 {
		return labelsMap, nil
	}

	labels := strings.Split(expression, ",")
	for _, label := range labels {
		l := strings.Split(label, "=")
		//nolint
		if len(l) != 2 {
			return labelsMap, fmt.Errorf("invalid expression: %s", l)
		}
		key := strings.TrimSpace(l[0])
		value := strings.TrimSpace(l[1])

		if set, ok := labelsMap[key]; ok {
			set.Add(value)
		} else {
			labelsMap[key] = strset.New(value)
		}
	}
	return labelsMap, nil
}
