/**
 * Admission webhooks callback function for Dynamic Admission Control
 * (https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).
 *
 * NOTE: Only kubernetes v1.16+ are supported since we are using "admissionregistration.k8s.io/v1".
 */
package admit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	admv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/liangrog/admission-webhook-server/pkg/utils"
)

const (
	// Query base path
	// Env var name for base path
	ENV_BASE_PATH = "BASE_PATH"
	// default base path
	defaultBasePath = "/mutate"

	// Request supported content type
	supportedContentType = "application/json"
)

var (
	// Deserializer
	UniversalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

// patchOperation is an operation of a JSON patch, see https://tools.ietf.org/html/rfc6902 .
type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// admitFunc is a callback for admission controller logic. Given an AdmissionRequest,
// it returns the sequence of patch operations to be applied in case of success,
// or the error that will be shown when the operation is rejected.
type AdmitFunc func(*admv1.AdmissionRequest) ([]PatchOperation, error)

// Get server base path
func GetBasePath() string {
	return utils.GetEnvVal(ENV_BASE_PATH, defaultBasePath)
}

var (
	// Namespaces come with default kubernetes
	namespaceByKube = []string{
		metav1.NamespacePublic,
		metav1.NamespaceSystem,
		metav1.NamespaceDefault,
	}
)

// isKubeNamespace checks if the given namespace is a Kubernetes-owned namespace.
func isKubeNamespace(ns string) bool {
	for _, n := range namespaceByKube {
		if ns == n {
			return true
		}
	}

	return false
}

// doServeAdmitFunc parses the HTTP request for an admission controller webhook, and in case
// of a well-formed request, delegates the admission control logic to the given admitFunc.
// The response body is then returned as raw bytes.
func doServeAdmitFunc(w http.ResponseWriter, r *http.Request, adm AdmitFunc) ([]byte, error) {
	// Step 1: Request validation. Only handle POST requests with a body and json content type.

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("invalid method %s, only POST requests are allowed", r.Method)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not read request body: %v", err)
	}

	if contentType := r.Header.Get("Content-Type"); contentType != supportedContentType {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("unsupported content type %s, only %s is supported", contentType, jsonContentType)
	}

	// Step 2: Parse the AdmissionReview request.

	var admissionReviewReq admv1.AdmissionReview

	if _, _, err := UniversalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("malformed admission review: request is nil")
	}

	// Step 3: Construct the AdmissionReview response.

	admissionReviewResponse := admv1.AdmissionReview{
		Response: &admv1.AdmissionResponse{
			UID: admissionReviewReq.Request.UID,
		},
	}

	var patchOps []PatchOperation
	// Apply the admit() function only for non-Kubernetes owned namespaces.
	// For objects in Kubernetes namespaces, return an empty set of patch operations.
	if !isKubeNamespace(admissionReviewReq.Request.Namespace) {
		patchOps, err = adm(admissionReviewReq.Request)
	}

	if err != nil {
		// If the handler returned an error, incorporate the error message
		// into the response and deny the object creation.
		admissionReviewResponse.Response.Allowed = false
		admissionReviewResponse.Response.Result = &metav1.Status{
			Message: err.Error(),
		}
	} else {
		// Otherwise, encode the patch operations to JSON and return a positive response.
		patchBytes, err := json.Marshal(patchOps)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, fmt.Errorf("could not marshal JSON patch: %v", err)
		}

		admissionReviewResponse.Response.Allowed = true
		admissionReviewResponse.Response.Patch = patchBytes
	}

	// Return the AdmissionReview with a response as JSON.
	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		return nil, fmt.Errorf("marshaling response: %v", err)
	}

	return bytes, nil
}

// serveAdmitFunc is a wrapper around doServeAdmitFunc that adds error handling and logging.
func serveAdmitFunc(w http.ResponseWriter, r *http.Request, adm AdmitFunc) {
	log := utils.GetLogger("pkg/admission/admit/serveAdmitFunc")

	var writeErr error
	if bytes, err := doServeAdmitFunc(w, r, adm); err != nil {
		log.Error().Msgf("Error handling webhook request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr = w.Write([]byte(err.Error()))
	} else {
		_, writeErr = w.Write(bytes)
	}

	if writeErr != nil {
		log.Error().Msgf("Could not write response: %v", writeErr)
	}
}

// admitFuncHandler takes an admitFunc and wraps it into a
// http.Handler by means of calling serveAdmitFunc.
func AdmitFuncHandler(adm AdmitFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveAdmitFunc(w, r, adm)
	})
}
