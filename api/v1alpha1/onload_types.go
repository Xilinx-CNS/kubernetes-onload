// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type OnloadKernelMapping struct {
	// Regexp is a regular expression that is used to match against the kernel
	// versions of the nodes in the cluster
	Regexp string `json:"regexp"`

	// KernelImage is the image that contains the out-of-tree kernel modules
	// used by onload, including the sfc driver.
	KernelModuleImage string `json:"kernelModuleImage"`
}

// OnloadSpec defines the desired state of Onload
type OnloadSpec struct {

	// KernelMappings is a list of pairs of kernel versions and container
	// images. This allows for flexibility when there are heterogenous kernel
	// versions on the nodes in the cluster.
	KernelMappings []OnloadKernelMapping `json:"kernelMappings"`

	// UserImage is the image that contains the built userland objects, used
	// by the cplane and deviceplugin DaemonSets.
	UserImage string `json:"userImage"`

	// Version string to associate with this onload CR.
	Version string `json:"version"`

	// +optional
	// ImagePullPolicy is the policy used when pulling images.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// Currently unimplemented
type DevicePluginSpec struct {
	// DevicePluginImage
	DevicePluginImage string `json:"devicePluginImage"`

	// +optional
	// ImagePullPolicy is the policy used when pulling images.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// Spec is the top-level specification for onload and related products that are
// controlled by the onload operator
type Spec struct {
	// Onload is the specification of the version of onload to be used by this
	// CR
	Onload OnloadSpec `json:"onload"`

	// DevicePlugin is the specification of the device plugin that will add a
	// new onload resource into the cluster.
	DevicePlugin DevicePluginSpec `json:"devicePlugin"`

	// Selector defines the set of nodes that this onload CR will run on.
	Selector map[string]string `json:"selector"`

	// ServiceAccountName is the name of the service account that the objects
	// created by the onload operator will use.
	ServiceAccountName string `json:"serviceAccountName"`
}

// OnloadStatus defines the observed state of Onload
type OnloadStatus struct {
}

type DevicePluginStatus struct {
}

// Status contains the statuses for onload and related products that are
// controlled by the onload operator
type Status struct {
	// Conditions store the status conditions of Onload
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Status of onload components
	Onload OnloadStatus `json:"onload"`

	// Status of the device plugin
	DevicePlugin DevicePluginStatus `json:"devicePlugin"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Onload is the Schema for the onloads API
type Onload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Spec   `json:"spec,omitempty"`
	Status Status `json:"status,omitempty"`
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
