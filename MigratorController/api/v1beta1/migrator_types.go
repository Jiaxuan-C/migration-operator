/*
Copyright 2023 Jiaxuan Chen.

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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Migrating states
const (
	StateCreatingSourcePod = "CreatingSourcePod"
	StateCreatingTargetPod = "CreatingTargetPod"
	StateRunning           = "Running"
	StateMigrating         = "Migrating"
	StateMigrated          = "Migrated"
	StateMigrationFailed   = "MigrationFailed"
)

// MigratorSpec defines the desired state of Migrator
type MigratorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	TargetNode       string             `json:"targetNode"`
	MigrationTrigger bool               `json:"migrationTrigger"`
	Template         v1.PodTemplateSpec `json:"template"`
}

// MigratorStatus defines the observed state of Migrator

type MigratorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	MigrationState string `json:"migrationState,omitempty"`
	SourcePod      string `json:"sourcePod,omitempty"`
	TargetPod      string `json:"targetPod,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Migrator is the Schema for the migrators API
type Migrator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MigratorSpec   `json:"spec,omitempty"`
	Status MigratorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MigratorList contains a list of Migrator
type MigratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Migrator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Migrator{}, &MigratorList{})
}
