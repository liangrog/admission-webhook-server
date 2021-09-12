package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/admit"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/annotations/namespace"
	"github.com/trilogy-group/admission-webhook-server/pkg/admission/annotations/pod"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	namespaceAnnotationsPath          = "/namespace-annotations"
	namespaceAnnotationsToAddSingle   = "{\"iam.amazonaws.com/permitted\":\".*\"}"
	namespaceAnnotationsToAddMultiple = "{\"iam.amazonaws.com/permitted\":\".*\",\"another\":\"annotation\"}"

	podAnnotationsPath          = "/namespace-annotations"
	podAnnotationsToAddSingle   = "{\"iam.amazonaws.com/role\":\"arn:aws:iam::346945241475:role/test-role\"}"
	podAnnotationsToAddMultiple = "{\"iam.amazonaws.com/role\":\"arn:aws:iam::346945241475:role/test-role\"," +
		"\"another\":\"annotation\"}"

	setenvErrorMessage = "Error while setting environment variables: %s"
)

var (
	namespaceAnnotationsToAdd = namespaceAnnotationsToAddSingle
	podAnnotationsToAdd       = podAnnotationsToAddSingle
)

func TestNamespaceAnnotationsHandler(t *testing.T) {
	testCases := []TestCase{
		{
			name:         "namespaceAnnotations - add annotation",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/namespace/addAnnotation-request.json",
			responseFile: "testdata/annotations/namespace/addAnnotation-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "namespaceAnnotations - annotation exists, different value",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/namespace/annotationExists-differentValue-request.json",
			responseFile: "testdata/annotations/namespace/annotationExists-differentValue-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "namespaceAnnotations - annotation exists, same value",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/namespace/annotationExists-sameValue-request.json",
			responseFile: "testdata/annotations/namespace/annotationExists-sameValue-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "namespaceAnnotations - no existing annotations",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/namespace/noExistingAnnotations-request.json",
			responseFile: "testdata/annotations/namespace/noExistingAnnotations-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "namespaceAnnotations - wrong resource type",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/namespace/wrongResourceType-request.json",
			responseFile: "testdata/annotations/namespace/wrongResourceType-response.json",
			statusCode:   http.StatusOK,
		},
	}
	runHttpTests(t, testCases, namespaceAnnotationsPath, admit.AdmitFuncHandler(namespace.Handler))

	initialNamespaceAnnotationsToAdd := namespaceAnnotationsToAdd
	namespaceAnnotationsToAdd = namespaceAnnotationsToAddMultiple
	testCases = []TestCase{
		{
			name:         "namespaceAnnotations - add multiple annotations",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/namespace/addMultipleAnnotations-request.json",
			responseFile: "testdata/annotations/namespace/addMultipleAnnotations-response.json",
			statusCode:   http.StatusOK,
		},
	}
	runHttpTests(t, testCases, namespaceAnnotationsPath, admit.AdmitFuncHandler(namespace.Handler))
	namespaceAnnotationsToAdd = initialNamespaceAnnotationsToAdd
}

func TestPodAnnotationsHandler(t *testing.T) {
	testCases := []TestCase{
		{
			name:         "podAnnotations - add annotation",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/pod/addAnnotation-request.json",
			responseFile: "testdata/annotations/pod/addAnnotation-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "podAnnotations - annotation exists, different value",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/pod/annotationExists-differentValue-request.json",
			responseFile: "testdata/annotations/pod/annotationExists-differentValue-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "podAnnotations - annotation exists, same value",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/pod/annotationExists-sameValue-request.json",
			responseFile: "testdata/annotations/pod/annotationExists-sameValue-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "podAnnotations - no existing annotations",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/pod/noExistingAnnotations-request.json",
			responseFile: "testdata/annotations/pod/noExistingAnnotations-response.json",
			statusCode:   http.StatusOK,
		},
		{
			name:         "podAnnotations - wrong resource type",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/pod/wrongResourceType-request.json",
			responseFile: "testdata/annotations/pod/wrongResourceType-response.json",
			statusCode:   http.StatusOK,
		},
	}
	runHttpTests(t, testCases, podAnnotationsPath, admit.AdmitFuncHandler(pod.Handler))

	initialPodAnnotationsToAdd := podAnnotationsToAdd
	podAnnotationsToAdd = podAnnotationsToAddMultiple
	testCases = []TestCase{
		{
			name:         "podAnnotations - add multiple annotations",
			method:       http.MethodPost,
			requestFile:  "testdata/annotations/pod/addMultipleAnnotations-request.json",
			responseFile: "testdata/annotations/pod/addMultipleAnnotations-response.json",
			statusCode:   http.StatusOK,
		},
	}
	runHttpTests(t, testCases, podAnnotationsPath, admit.AdmitFuncHandler(pod.Handler))
	podAnnotationsToAdd = initialPodAnnotationsToAdd
}

func runHttpTests(t *testing.T, testCases []TestCase, endpoint string, handler http.Handler) {
	setupEnvVariables(t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody, err := ioutil.ReadFile(tc.requestFile)
			if err != nil {
				t.Errorf("Could not read request file %v: %v", tc.requestFile, err)
			}
			wantResponseBody, err := ioutil.ReadFile(tc.responseFile)
			if err != nil {
				t.Errorf("Could not read response file %v: %v", tc.responseFile, err)
			}

			request := httptest.NewRequest(tc.method, endpoint, strings.NewReader(string(requestBody)))
			request.Header.Add("Content-Type", "application/json")
			responseRecorder := httptest.NewRecorder()
			handler.ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != tc.statusCode {
				t.Errorf("Want status %d, got %d with message %s", tc.statusCode, responseRecorder.Code,
					responseRecorder.Body.String())
			}

			responseBodyWithDecodedPatch, err := decodePatch(responseRecorder.Body.Bytes())
			if err != nil {
				t.Errorf("Could not decode patch from response body: %v", responseRecorder.Body.String())
			}
			var gotResponseBody bytes.Buffer
			err = json.Indent(&gotResponseBody, responseBodyWithDecodedPatch, "", "  ")
			if err != nil {
				t.Errorf("Could not indent response body: %v", err)
			}
			if diff := cmp.Diff(string(wantResponseBody), gotResponseBody.String()); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", endpoint, diff)
			}
		})
	}
}

func setupEnvVariables(t *testing.T) {
	err := os.Setenv(namespace.EnvNamespaceAnnotationsPath, namespaceAnnotationsPath)
	if err != nil {
		t.Errorf(setenvErrorMessage, err)
	}
	err = os.Setenv(namespace.EnvNamespaceAnnotationsToAdd, namespaceAnnotationsToAdd)
	if err != nil {
		t.Errorf(setenvErrorMessage, err)
	}
	err = os.Setenv(pod.EnvPodAnnotationsPath, namespaceAnnotationsPath)
	if err != nil {
		t.Errorf(setenvErrorMessage, err)
	}
	err = os.Setenv(pod.EnvPodAnnotationsToAdd, podAnnotationsToAdd)
	if err != nil {
		t.Errorf(setenvErrorMessage, err)
	}
	namespace.Init()
	pod.Init()
}

func decodePatch(admissionReviewJSON []byte) ([]byte, error) {
	admissionReview := v1beta1.AdmissionReview{}
	err := json.Unmarshal(admissionReviewJSON, &admissionReview)
	if err != nil {
		return nil, err
	}

	var patch []Patch
	if admissionReview.Response.Patch != nil {
		err = json.Unmarshal(admissionReview.Response.Patch, &patch)
		if err != nil {
			return nil, err
		}
	}

	admissionReviewWithDecodedPatch := AdmissionReviewWithDecodedPatch{
		Response: AdmissionResponseWithDecodedPatch{
			Allowed: admissionReview.Response.Allowed,
			Patch:   patch,
			Status:  admissionReview.Response.Result,
			UID:     admissionReview.Response.UID,
		},
	}
	admissionReviewWithDecodedPatchJSON, err := json.Marshal(admissionReviewWithDecodedPatch)
	if err != nil {
		return nil, err
	}
	return admissionReviewWithDecodedPatchJSON, nil
}

type AdmissionReviewWithDecodedPatch struct {
	Response AdmissionResponseWithDecodedPatch `json:"response"`
}

type AdmissionResponseWithDecodedPatch struct {
	Allowed bool           `json:"allowed"`
	Patch   []Patch        `json:"patch"`
	Status  *metav1.Status `json:"status,omitempty"`
	UID     types.UID      `json:"uid"`
}

type Patch struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value map[string]string `json:"value"`
}

type TestCase struct {
	name         string
	method       string
	requestFile  string
	responseFile string
	statusCode   int
}
