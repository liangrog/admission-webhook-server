package annotations

import (
	"encoding/json"
	"fmt"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/admit"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type ResourceDeserializer func([]byte) (metav1.ObjectMetaAccessor, error)

func PackageInit(pathEnv string, annotationsEnv string) (string, map[string]string) {
	urlPath := os.Getenv(pathEnv)
	annotations := map[string]string{}
	err := json.Unmarshal([]byte(os.Getenv(annotationsEnv)), &annotations)
	if err != nil && isHandlerEnabled(urlPath) {
		panic(err)
	}
	return urlPath, annotations
}

func Register(handlerName string, mux *http.ServeMux, urlPath string, handler admit.AdmitFunc) {
	if !isHandlerEnabled(urlPath) {
		log.Printf("%v is disabled", handlerName)
		return
	}
	serverPath := filepath.Join(admit.GetBasePath(), urlPath)
	mux.Handle(serverPath, admit.AdmitFuncHandler(handler))
	log.Printf("%v registered using path %v", handlerName, serverPath)
}

func Handler(handlerName string, req *v1beta1.AdmissionRequest, resourceType metav1.GroupVersionResource,
	resourceDeserializer ResourceDeserializer, desiredAnnotations map[string]string) ([]admit.PatchOperation, error) {
	if req.Resource != resourceType {
		log.Printf("%v: Ignoring admission request %v as it's not an expected resource", handlerName, req.UID)
		return nil, nil
	}

	resource, err := resourceDeserializer(req.Object.Raw)
	if err != nil {
		return nil, fmt.Errorf("%v: could not deserialize resource: %v", handlerName, err)
	}

	var patches []admit.PatchOperation
	resourceMeta := resource.GetObjectMeta()
	resourceAnnotations := resourceMeta.GetAnnotations()
	if labels.Conflicts(resourceAnnotations, desiredAnnotations) {
		return patches,
			fmt.Errorf("%v: existing annotations conflict with desired annotations - existing: %v, desired: %v",
				handlerName, resourceAnnotations, desiredAnnotations)
	}

	patchOp := "replace"
	if len(resourceAnnotations) == 0 {
		patchOp = "add"
	}
	patchValue := labels.Merge(resourceAnnotations, desiredAnnotations)
	patches = append(patches, admit.PatchOperation{
		Op:    patchOp,
		Path:  "/metadata/annotations",
		Value: patchValue,
	})
	log.Printf("%v processed %v with annotations: %v", handlerName, resourceMeta.GetName(), patchValue)
	return patches, nil
}

func isHandlerEnabled(urlPath string) bool {
	return urlPath != ""
}
