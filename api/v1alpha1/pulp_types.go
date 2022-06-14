/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PulpSpec defines the desired state of Pulp
type PulpSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Pulp. Edit pulp_types.go to remove/update
	Foo      string   `json:"foo,omitempty"`
	Api      Api      `json:"api"`
	Database Database `json:"database"`
	Content  Content  `json:"content"`
	Worker   Worker   `json:"worker"`
}

type Api struct {
	// Size is the size of number of pulp-api replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas"`
}

type Content struct {
	// Size is the size of number of pulp-content replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas"`
}

type Worker struct {
	// Size is the size of number of pulp-worker replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas"`
}

type Database struct {
	// Size is the size of number of db replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32  `json:"replicas"`
	Type     string `json:"type"`
}

// PulpStatus defines the observed state of Pulp
type PulpStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Nodes []string `json:"nodes"`
}

// Pulp is the Schema for the pulps API
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
type Pulp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PulpSpec   `json:"spec,omitempty"`
	Status PulpStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PulpList contains a list of Pulp
type PulpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pulp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pulp{}, &PulpList{})
}
