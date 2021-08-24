package namespace

import (
	"fmt"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/admit"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/annotations"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

const (
	EnvNamespaceAnnotationsPath  = "NAMESPACE_ANNOTATIONS_PATH"
	EnvNamespaceAnnotationsToAdd = "NAMESPACE_ANNOTATIONS_TO_ADD"
	HandlerName                  = "NamespaceAnnotationsHandler"
)

var (
	urlPath           string
	annotationsToAdd  map[string]string
	namespaceResource = metav1.GroupVersionResource{Resource: "namespaces", Version: "v1"}
)

func Register(mux *http.ServeMux) {
	Init()
	annotations.Register(HandlerName, mux, urlPath, Handler)
}

func Init() {
	urlPath, annotationsToAdd = annotations.PackageInit(EnvNamespaceAnnotationsPath, EnvNamespaceAnnotationsToAdd)
}

func Handler(req *v1beta1.AdmissionRequest) ([]admit.PatchOperation, error) {
	return annotations.Handler(HandlerName, req, namespaceResource, deserializeNamespace, annotationsToAdd)
}

func deserializeNamespace(rawResource []byte) (metav1.ObjectMetaAccessor, error) {
	namespace := corev1.Namespace{}
	if _, _, err := admit.UniversalDeserializer.Decode(rawResource, nil, &namespace); err != nil {
		return nil, fmt.Errorf("could not deserialize namespace object: %v", err)
	}
	return &namespace, nil
}
