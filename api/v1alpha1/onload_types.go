// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Currently unimplemented
type SFCSpec struct {
}

// BuildArg represents a build argument used when building a container image.
type BuildArg struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Build is a subset of the build options presented by the Kernel Module
// Management operator.
type OnloadKernelBuild struct {
	// +optional
	// BuildArgs is an array of build variables that are provided to the image building backend.
	BuildArgs []BuildArg `json:"buildArgs"`

	// ConfigMap that holds Dockerfile contents
	DockerfileConfigMap *v1.LocalObjectReference `json:"dockerfileConfigMap"`
}

type OnloadKernelMapping struct {
	// Regexp is a regular expression that is used to match against the kernel
	// versions of the nodes in the cluster
	Regexp string `json:"regexp"`

	// KernelModuleImage is the image that contains the out-of-tree kernel
	// modules used by Onload.
	KernelModuleImage string `json:"kernelModuleImage"`

	// +optional
	// SFC optionally specifies that the controller will manage the SFC
	// kernel module.
	SFC *SFCSpec `json:"sfc,omitempty"`

	// +optional
	// Build specifies the parameters that are to be passed to the Kernel Module
	// Management operator when building the images that contain the module.
	// The build process creates a new image which will be written to the
	// location specified by the `KernelModuleImage` parameter.
	// If empty no builds will take place.
	Build *OnloadKernelBuild `json:"build,omitempty"`
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

	// +optional
	// MaxPodsPerNode is the number of Kubernetes devices that the Onload
	// Device Plugin should register with the kubelet. Notionally this is
	// equivalent to the number of pods that can request an onload resource on
	// each node.
	// +kubebuilder:default:=100
	MaxPodsPerNode *int `json:"maxPodsPerNode,omitempty"`
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
