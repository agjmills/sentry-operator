package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SentryProjectRefSpec defines the desired state of a SentryProjectRef.
// Unlike SentryProject, this resource references an existing Sentry project
// and never creates or deletes it.
type SentryProjectRefSpec struct {
	// Organization is the Sentry organization slug.
	// Falls back to the operator's --default-organization flag if unset.
	// +optional
	Organization string `json:"organization,omitempty"`

	// ProjectSlug is the slug of the existing Sentry project to reference.
	// +kubebuilder:validation:Required
	ProjectSlug string `json:"projectSlug"`

	// SecretName is the name of the Kubernetes Secret created in this namespace.
	// Defaults to "<metadata.name>-sentry".
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Keys defines the Sentry DSN keys to fetch for this project.
	// Each key must already exist in Sentry; keys are not created.
	// If empty, the first existing key is used and written as SENTRY_DSN.
	// +optional
	// +listType=atomic
	Keys []KeySpec `json:"keys,omitempty"`

	// SecretKeys controls the key names written into the output Secret.
	// Only applies when spec.keys is empty (single-key fallback mode).
	// +optional
	SecretKeys *SecretKeysSpec `json:"secretKeys,omitempty"`

	// DefaultRateLimit applies to all keys in spec.keys that do not have their
	// own rateLimit set.
	// +optional
	DefaultRateLimit *RateLimitSpec `json:"defaultRateLimit,omitempty"`
}

// SentryProjectRefStatus defines the observed state of a SentryProjectRef.
type SentryProjectRefStatus struct {
	// Conditions represent the latest available observations of the resource's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ProjectSlug is the slug of the referenced Sentry project.
	// +optional
	ProjectSlug string `json:"projectSlug,omitempty"`

	// SecretName is the name of the Kubernetes Secret that was created.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// LastSyncTime is the timestamp of the last successful reconciliation.
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// ObservedGeneration is the generation last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// Condition reason specific to SentryProjectRef.
const (
	ReasonRefProjectNotFound = "ProjectNotFound"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=spr
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.status.projectSlug`
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.status.secretName`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SentryProjectRef references an existing Sentry project and writes its DSN
// into a Kubernetes Secret. Unlike SentryProject, it never creates or deletes
// the Sentry project itself.
type SentryProjectRef struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SentryProjectRefSpec   `json:"spec,omitempty"`
	Status SentryProjectRefStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SentryProjectRefList contains a list of SentryProjectRef.
type SentryProjectRefList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SentryProjectRef `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SentryProjectRef{}, &SentryProjectRefList{})
}
