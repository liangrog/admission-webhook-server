package pod

import (
	"fmt"
	"net/http"

	"github.com/trilogy-group/admission-webhook-server/pkg/admission/admit"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/annotations"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EnvPodAnnotationsPath  = "POD_ANNOTATIONS_PATH"
	EnvPodAnnotationsToAdd = "POD_ANNOTATIONS_TO_ADD"
	HandlerName            = "PodAnnotationsHandler"
)

var (
	urlPath          string
	annotationsToAdd map[string]string
	podResource      = metav1.GroupVersionResource{Resource: "pods", Version: "v1"}
)

func Register(mux *http.ServeMux) {
	Init()
	annotations.Register(HandlerName, mux, urlPath, Handler)
}

func Init() {
	urlPath, annotationsToAdd = annotations.PackageInit(EnvPodAnnotationsPath, EnvPodAnnotationsToAdd)
}

func Handler(req *v1beta1.AdmissionRequest) ([]admit.PatchOperation, error) {
	return annotations.Handler(HandlerName, req, podResource, deserializePod, annotationsToAdd)
}

func deserializePod(rawResource []byte) (metav1.ObjectMetaAccessor, error) {
	pod := corev1.Pod{}
	if _, _, err := admit.UniversalDeserializer.Decode(rawResource, nil, &pod); err != nil {
		return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	}
	return &pod, nil
}
