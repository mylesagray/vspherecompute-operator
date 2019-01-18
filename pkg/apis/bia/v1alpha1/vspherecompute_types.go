package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VSphereComputeSpec defines the desired state of VSphereCompute
type VSphereComputeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Name                string         `json:"name"`
	Template            string         `json:"template"`
	VMCpu               int            `json:"vmCpu"`
	VMMemory            int            `json:"vmMemory"`
	VMReservedResources bool           `json:"vmReservedResources"`
	Replicas            int            `json:"replicas"`
	Affinity            VMAffinitySpec `json:"affinity"`
	Networks            VMNetworkSpec  `json:"networks"`
}

// VSphereComputeStatus defines the observed state of VSphereCompute
type VSphereComputeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Host   string `json:"host"`
	VMName string `json:"vmname"`
	Status string `json:"status"`
	IP     string `json:"ip"`
	CPU    int    `json:"cpu"`
	Memory int    `json:"memory"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VSphereCompute is the Schema for the vspherecomputes API
// +k8s:openapi-gen=true
type VSphereCompute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VSphereComputeSpec   `json:"spec,omitempty"`
	Status VSphereComputeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VSphereComputeList contains a list of VSphereCompute
type VSphereComputeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VSphereCompute `json:"items"`
}

type VMNetworkSpec struct {
	Name string `json:"name"`
}

type VMAffinitySpec struct {
	VMAntiAffinity VMAntiAffinitySpec `json:"vmAntiAffinity"`
}

type VMAntiAffinitySpec struct {
	TopologyKey string `json:"topologyKey"`
	Enforced    bool   `json:"enforced"`
}

func init() {
	SchemeBuilder.Register(&VSphereCompute{}, &VSphereComputeList{})
}
