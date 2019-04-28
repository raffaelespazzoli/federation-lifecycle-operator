package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MultipleNamespaceFederationSpec defines the desired state of MultipleNamespaceFederation
// +k8s:openapi-gen=true
type MultipleNamespaceFederationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	NamespaceFederationSpec NamespaceFederationSpec `json:"namespaceFederationSpec,omitempty"`
	NamespaceSelector       *metav1.LabelSelector   `json:"namespaceSelector,omitempty"`
	GlobalLoadBalancer      GlobalLoadBalancer      `json:"globalLoadBalancer,omitempty"`
}

type GlobalLoadBalancer struct {
	// accepted values are cloud-provider and self-hosted
	// +kubebuilder:validation:Enum=cloud-provider,self-hosted
	GlobalLoadBalancerType string `json:"type,omitempty"`
	// +kubebuilder:validation:UniqueItems=false
	ExternalDNSArgs []string `json:"externalDNSArgs,omitempty"`
	// ControllerURL represents the master endpoint for this cluster, used only when type=Self-Hosted
	ControllerURL string `json:"controllerURL,omitempty"`
	Provider      string `json:"provider,omitempty"`
}

// MultipleNamespaceFederationStatus defines the observed state of MultipleNamespaceFederation
// +k8s:openapi-gen=true
type MultipleNamespaceFederationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultipleNamespaceFederation is the Schema for the multiplenamespacefederations API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Cluster Registry Namespace",type="string",JSONPath=".spec.namespaceFederationSpec.clusterRegistryNamespace",description="namespace of the clustere registry for this federation"
// +kubebuilder:printcolumn:name="Global Load Balancer Type",type="string",JSONPath=".spec.globalLoadBalancer.type",description="type of the global load balancer"
type MultipleNamespaceFederation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultipleNamespaceFederationSpec   `json:"spec,omitempty"`
	Status MultipleNamespaceFederationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultipleNamespaceFederationList contains a list of MultipleNamespaceFederation
type MultipleNamespaceFederationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultipleNamespaceFederation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultipleNamespaceFederation{}, &MultipleNamespaceFederationList{})
}
