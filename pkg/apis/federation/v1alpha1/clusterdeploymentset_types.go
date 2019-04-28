package v1alpha1

import (
	hivev1aplha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterDeploymentSetSpec defines the desired state of ClusterDeploymentSet
// +k8s:openapi-gen=true
type ClusterDeploymentSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Template                ClusterDeploymentTemplate `json:"template,omitempty"`
	Replicas                int                       `json:"replicas,omitempty"`
	EnsureNoOverlappingCIDR bool                      `json:"ensureNoOverlappingCIDR,omitempty"`
	Regions                 []string                  `json:"regions,omitempty"`
	RegisterClusters        bool                      `json:"registerClusters,omitempty"`
}

type ClusterDeploymentTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              hivev1aplha1.ClusterDeploymentSpec `json:"spec,omitempty"`
}

// ClusterDeploymentSetStatus defines the observed state of ClusterDeploymentSet
// +k8s:openapi-gen=true
type ClusterDeploymentSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterDeploymentSet is the Schema for the clusterdeploymentsets API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.Replicas",description="The number of clusters to be created"
// +kubebuilder:printcolumn:name="Regions",type="string",JSONPath=".spec.regions",description="The regions where the clusters will be created"
type ClusterDeploymentSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDeploymentSetSpec   `json:"spec,omitempty"`
	Status ClusterDeploymentSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterDeploymentSetList contains a list of ClusterDeploymentSet
type ClusterDeploymentSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDeploymentSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterDeploymentSet{}, &ClusterDeploymentSetList{})
}
