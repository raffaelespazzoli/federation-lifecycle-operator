package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NamespaceFederationSpec defines the desired state of NamespaceFederation
// +k8s:openapi-gen=true
type NamespaceFederationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	// These are cluster name ref to cluster defined in the cluster registry
	Clusters       []Cluster         `json:"clusters,omitempty"`
	FederatedTypes []metav1.TypeMeta `json:"federatedTypes,omitempty"`
	// +kubebuilder:validation:Pattern=(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]
	Domains                  []string `json:"domains,omitempty"`
	ClusterRegistryNamespace string   `json:"clusterRegistryNamespace,omitempty"`
}

type Cluster struct {
	Name           string              `json:"name,omitempty"`
	AdminSecretRef NamespacedObjectRef `json:"adminSecretRef,omitempty"`
}

type NamespacedObjectRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// NamespaceFederationStatus defines the observed state of NamespaceFederation
// +k8s:openapi-gen=true
type NamespaceFederationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	ClusterRegistrationStatuses []ClusterRegistrationStatus `json:"clusterRegistrationStatuses,omitempty"`
}

// ClusterRegistrationStatus reports whether the status of the cluster registration to the namespace controller
type ClusterRegistrationStatus struct {
	Cluster string `json:"cluster,omitempty"`
	Status  string `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceFederation is the Schema for the namespacefederations API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="ClusterRegistryNamespace",type="string",JSONPath=".spec.clusterRegistryNamespace",description="namespace of the clustere registry for this federation"
type NamespaceFederation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceFederationSpec   `json:"spec,omitempty"`
	Status NamespaceFederationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceFederationList contains a list of NamespaceFederation
type NamespaceFederationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespaceFederation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespaceFederation{}, &NamespaceFederationList{})
}
