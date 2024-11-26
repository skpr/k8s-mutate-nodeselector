package mutator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// MockNamespaceClient for testing our logic.
type MockNamespaceClient struct {
	clientcorev1.NamespaceInterface
	Namespace *corev1.Namespace
	Error     error
}

// Get a mock response of namespace and error.
func (m *MockNamespaceClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Namespace, error) {
	return m.Namespace, m.Error
}

func TestHandle(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: corev1.PodSpec{},
	}

	podRaw, err := json.Marshal(pod)
	if err != nil {
		t.Fatalf("failed to marshal pod: %v", err)
	}

	admissionReviewRequest := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:       "12345",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Resource:  metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Namespace: "default",
			Object:    runtime.RawExtension{Raw: podRaw},
		},
	}

	requestBody, err := json.Marshal(admissionReviewRequest)
	if err != nil {
		t.Fatalf("failed to marshal admission review request: %v", err)
	}

	// Create a new HTTP request for the handler
	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to capture the response
	rr := httptest.NewRecorder()

	client := &MockNamespaceClient{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					annotation: "foo=bar",
				},
			},
		},
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{}))

	// Setup and call our handler to execute our code.
	handler := NewHandler(logger, client)
	handler.Handle(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var admissionReviewResponse admissionv1.AdmissionReview

	if err := json.Unmarshal(rr.Body.Bytes(), &admissionReviewResponse); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	assert.Equal(t, admissionReviewResponse.Response.UID, admissionReviewRequest.Request.UID)
	assert.NotNil(t, admissionReviewResponse.Response.Patch)

	var patch []map[string]interface{}

	if err := json.Unmarshal(admissionReviewResponse.Response.Patch, &patch); err != nil {
		t.Fatalf("failed to unmarshal patch: %v", err)
	}

	assert.Equal(t, []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/spec/nodeSelector",
			"value": map[string]interface{}{"foo": "bar"},
		},
	}, patch)
}

func TestGetNodeSelectorFromNamespace(t *testing.T) {
	// Annotations are not set.
	client := &MockNamespaceClient{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		},
	}
	nodeSelector, err := getNodeSelectorFromNamespace(context.TODO(), client, "test")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{}, nodeSelector)

	// Annotations are set.
	client = &MockNamespaceClient{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					annotation: "foo=bar",
				},
			},
		},
	}
	nodeSelector, err = getNodeSelectorFromNamespace(context.TODO(), client, "test")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"foo": "bar",
	}, nodeSelector)

	// Namespace does not exist.
	client = &MockNamespaceClient{
		Error: fmt.Errorf("namespace not found"),
	}
	nodeSelector, err = getNodeSelectorFromNamespace(context.TODO(), client, "test")
	assert.Error(t, err)
	assert.Equal(t, map[string]string{}, nodeSelector)

	// Annotation was set incorrectly.
	client = &MockNamespaceClient{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					annotation: "foo",
				},
			},
		},
	}
	nodeSelector, err = getNodeSelectorFromNamespace(context.TODO(), client, "test")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{}, nodeSelector)
}
