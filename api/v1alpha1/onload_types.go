// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnloadSpec defines the desired state of Onload
type OnloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Onload. Edit onload_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// OnloadStatus defines the observed state of Onload
type OnloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Onload is the Schema for the onloads API
type Onload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OnloadSpec   `json:"spec,omitempty"`
	Status OnloadStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OnloadList contains a list of Onload
type OnloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Onload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Onload{}, &OnloadList{})
}
