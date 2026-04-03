package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RateLimitSpec configures the inbound event rate limit for a Sentry DSN key.
type RateLimitSpec struct {
	// Count is the maximum number of events allowed per window.
	// +kubebuilder:validation:Minimum=1
	Count int `json:"count"`

	// Window is the rate limit window in seconds.
	// +kubebuilder:default=3600
	// +kubebuilder:validation:Minimum=1
	Window int `json:"window"`
}

// KeySpec defines a Sentry DSN key to create or manage.
type KeySpec struct {
	// Name is the label given to this key in Sentry.
	Name string `json:"name"`

	// SecretKey is the key name written into the Kubernetes Secret.
	// Defaults to "SENTRY_DSN" for the first key, "SENTRY_DSN_<NAME>" for subsequent ones.
	// +optional
	SecretKey string `json:"secretKey,omitempty"`

	// RateLimit configures the inbound event rate limit for this key.
	// Overrides spec.defaultRateLimit if set.
	// +optional
	RateLimit *RateLimitSpec `json:"rateLimit,omitempty"`
}

// SentryProjectSpec defines the desired state of a SentryProject.
type SentryProjectSpec struct {
	// Organization is the Sentry organization slug.
	// Falls back to the operator's --default-organization flag if unset.
	// +optional
	Organization string `json:"organization,omitempty"`

	// Team is the Sentry team slug that owns this project.
	// Falls back to the operator's --default-team flag if unset.
	// +optional
	Team string `json:"team,omitempty"`

	// Platform is the Sentry platform identifier (e.g. "go", "python-django", "javascript").
	// See https://docs.sentry.io/platforms/ for valid values.
	// Falls back to the operator's --default-platform flag if unset.
	// +optional
	Platform string `json:"platform,omitempty"`

	// ProjectSlug overrides the Sentry project slug.
	// Defaults to the resource's metadata.name.
	// +optional
	ProjectSlug string `json:"projectSlug,omitempty"`

	// SecretName is the name of the Kubernetes Secret created in this namespace.
	// Defaults to "<metadata.name>-sentry".
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// RetainOnDelete controls whether the Sentry project is deleted when this
	// resource is deleted. Defaults to true (project is retained).
	// Set to false to cascade-delete the Sentry project.
	// +optional
	// +kubebuilder:default=true
	RetainOnDelete *bool `json:"retainOnDelete,omitempty"`

	// SecretKeys controls the key names written into the output Secret.
	// Only applies when spec.keys is empty (single-key fallback mode).
	// +optional
	SecretKeys *SecretKeysSpec `json:"secretKeys,omitempty"`

	// Keys defines the Sentry DSN keys to create and manage for this project.
	// Each key is created in Sentry if it does not exist, and its DSN is written
	// into the output Secret under the configured key name.
	// If empty, the first existing key is used and written as SENTRY_DSN.
	// +optional
	// +listType=atomic
	Keys []KeySpec `json:"keys,omitempty"`

	// DefaultRateLimit applies to all keys in spec.keys that do not have their
	// own rateLimit set.
	// +optional
	DefaultRateLimit *RateLimitSpec `json:"defaultRateLimit,omitempty"`
}

// SecretKeysSpec configures the key names written into the output Secret.
// All fields default to standard environment variable names used by Sentry SDKs.
type SecretKeysSpec struct {
	// DSN is the key name for the Sentry DSN value.
	// Defaults to "SENTRY_DSN".
	// +optional
	// +kubebuilder:default="SENTRY_DSN"
	DSN string `json:"dsn,omitempty"`

	// Environment is the key name for the Sentry environment value.
	// Only written if spec.environment is set.
	// Defaults to "SENTRY_ENVIRONMENT".
	// +optional
	// +kubebuilder:default="SENTRY_ENVIRONMENT"
	Environment string `json:"environment,omitempty"`

	// Release is the key name for the Sentry release value.
	// Only written if spec.release is set.
	// Defaults to "SENTRY_RELEASE".
	// +optional
	// +kubebuilder:default="SENTRY_RELEASE"
	Release string `json:"release,omitempty"`
}

// KeyStatus records the Sentry key ID for a managed DSN key.
// This allows the operator to match keys by ID rather than label on subsequent
// reconciles, so externally renamed keys are adopted rather than duplicated.
type KeyStatus struct {
	// Name is the key label as specified in spec.keys.
	Name string `json:"name"`

	// ID is the Sentry key ID.
	ID string `json:"id"`

	// SecretKey is the key name written into the Kubernetes Secret.
	SecretKey string `json:"secretKey"`
}

// SentryProjectStatus defines the observed state of a SentryProject.
type SentryProjectStatus struct {
	// Conditions represent the latest available observations of the resource's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ProjectSlug is the slug of the provisioned Sentry project.
	// +optional
	ProjectSlug string `json:"projectSlug,omitempty"`

	// SecretName is the name of the Kubernetes Secret that was created.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Keys tracks the Sentry key IDs for each managed DSN key.
	// Used to match keys by ID on subsequent reconciles, surviving label renames.
	// +optional
	// +listType=atomic
	Keys []KeyStatus `json:"keys,omitempty"`

	// LastSyncTime is the timestamp of the last successful reconciliation.
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// ObservedGeneration is the generation of the resource last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// Condition type constants.
const (
	ConditionReady = "Ready"

	ReasonProjectProvisioned = "ProjectProvisioned"
	ReasonProvisionFailed    = "ProvisionFailed"
	ReasonSecretSynced       = "SecretSynced"
	ReasonDeleting           = "Deleting"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=sp
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.status.projectSlug`
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.status.secretName`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SentryProject is the Schema for the sentryprojects API.
// Creating a SentryProject causes the operator to provision a project in Sentry
// and write a Secret containing SENTRY_DSN (and other optional values) into the
// same namespace.
type SentryProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SentryProjectSpec   `json:"spec,omitempty"`
	Status SentryProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SentryProjectList contains a list of SentryProject.
type SentryProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SentryProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SentryProject{}, &SentryProjectList{})
}
