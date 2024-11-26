// Package mutator for handling mutation requests.
package mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Annotation used to declare where a Pod should be scheduled.
const annotation = "k8s-mutate-nodeselector.skpr.io/namespace"

// Handler for responding to mutation requests.
type Handler struct {
	logger *slog.Logger
	client clientcorev1.NamespaceInterface
}

// NewHandler for mutating requests.
func NewHandler(logger *slog.Logger, client clientcorev1.NamespaceInterface) *Handler {
	return &Handler{
		logger: logger,
		client: client,
	}
}

// Handle mutate requests.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("received webhook mutate request")

	var admissionReviewRequest admissionv1.AdmissionReview

	if err := json.NewDecoder(r.Body).Decode(&admissionReviewRequest); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode admission review: %v", err), http.StatusBadRequest)
		return
	}

	pod := &corev1.Pod{}

	if err := json.Unmarshal(admissionReviewRequest.Request.Object.Raw, pod); err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal pod: %v", err), http.StatusBadRequest)
		return
	}

	// Lookup the Pod's Namespace and check if it has the annotation set.
	nodeSelector, err := getNodeSelectorFromNamespace(context.TODO(), h.client, pod.ObjectMeta.Namespace)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get node selector: %v", err), http.StatusBadRequest)
		return
	}

	if len(nodeSelector) == 0 {
		// No changes needed; return Allowed response without a patch
		admissionReviewResponse := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     admissionReviewRequest.Request.UID,
				Allowed: true,
			},
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(admissionReviewResponse); err != nil {
			http.Error(w, fmt.Sprintf("failed to encode admission review response: %v", err), http.StatusInternalServerError)
		}

		return
	}

	patchBytes, err := json.Marshal([]map[string]interface{}{
		{
			"op":    "add",
			"path":  "/spec/nodeSelector",
			"value": nodeSelector,
		},
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create patch: %v", err), http.StatusInternalServerError)
		return
	}

	// Construct the AdmissionReview response
	admissionReviewResponse := admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			UID:       admissionReviewRequest.Request.UID,
			Allowed:   true,
			Patch:     patchBytes,
			PatchType: func() *admissionv1.PatchType { pt := admissionv1.PatchTypeJSONPatch; return &pt }(),
		},
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(admissionReviewResponse); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode admission review response: %v", err), http.StatusInternalServerError)
	}

	h.logger.Info("completed webhook mutate request")
}

// Get the node selector for a given namespace.
func getNodeSelectorFromNamespace(ctx context.Context, client clientcorev1.NamespaceInterface, name string) (map[string]string, error) {
	nodeSelector := make(map[string]string)

	namespace, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nodeSelector, fmt.Errorf("failed to get namespace: %w", err)
	}

	if len(namespace.ObjectMeta.Annotations) == 0 {
		return nodeSelector, nil
	}

	if _, ok := namespace.ObjectMeta.Annotations[annotation]; !ok {
		return nodeSelector, nil
	}

	for _, flatSelector := range strings.Split(namespace.ObjectMeta.Annotations[annotation], ",") {
		kv := strings.Split(flatSelector, "=")

		if len(kv) != 2 {
			continue
		}

		nodeSelector[kv[0]] = kv[1]
	}

	return nodeSelector, nil
}
