// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// NOTE: Update sample properties in:
//       - config/samples/onload/base/onload_v1alpha1_onload.yaml
//       - config/samples/onload/overlays/in-cluster-build-ocp/patch-onload.yaml

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
	// versions of the nodes in the cluster. Use also in place of literal strings.
	Regexp string `json:"regexp"`

	// KernelModuleImage is the image that contains the out-of-tree kernel
	// modules used by Onload. Absent image tags may be built by KMM.
	KernelModuleImage string `json:"kernelModuleImage"`

	// +optional
	// SFC optionally specifies that the controller will manage the SFC
	// kernel module. Incompatible with boot-time loading approaches.
	SFC *SFCSpec `json:"sfc,omitempty"`

	// +optional
	// Build specifies the parameters that are to be passed to the Kernel Module
	// Management operator when building the images that contain the module.
	// The build process creates a new image which will be written to the
	// location specified by the `KernelModuleImage` parameter.
	// If empty, no builds will take place.
	Build *OnloadKernelBuild `json:"build,omitempty"`
}

// OnloadSpec defines the desired state of Onload
type OnloadSpec struct {

	// KernelMappings is a list of pairs of kernel versions and container
	// images. This allows for flexibility when there are heterogenous kernel
	// versions on the nodes in the cluster.
	KernelMappings []OnloadKernelMapping `json:"kernelMappings"`

	// UserImage is the image that contains the built userland objects, used
	// within the Onload Device Plugin DaemonSet.
	UserImage string `json:"userImage"`

	// Version string to associate with this Onload CR.
	Version string `json:"version"`

	// +optional
	// ImagePullPolicy is the policy used when pulling images.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// Currently unimplemented
// +kubebuilder:validation:XValidation:message="SetPreload and MountOnload mutually exclusive",rule="!(self.setPreload && self.mountOnload)"
type DevicePluginSpec struct {

	// +optional
	// ImagePullPolicy is the policy used when pulling images.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	// MaxPodsPerNode is the number of Kubernetes devices that the Onload
	// Device Plugin should register with the kubelet. Notionally this is
	// equivalent to the number of pods that can request an Onload resource on
	// each node.
	// +kubebuilder:default:=100
	MaxPodsPerNode *int `json:"maxPodsPerNode,omitempty"`

	// +optional
	// Preload determines whether the Onload Device Plugin will set LD_PRELOAD
	// for pods using Onload.
	// Mutually exclusive with MountOnload
	// +kubebuilder:default:=true
	SetPreload *bool `json:"setPreload,omitempty"`

	// +optional
	// MountOnload is used by the Onload Device Plugin to decide whether to
	// mount the `onload` script as a file in the container's filesystem.
	// `onload` is mounted at `<baseMountPath>/<binMountpath>`
	// Mutually exclusive with Preload
	// +kubebuilder:default:=false
	MountOnload *bool `json:"mountOnload,omitempty"`

	// +optional
	// HostOnloadPath is the base location of Onload files on the host
	// filesystem.
	// +kubebuilder:default=/opt/onload/
	HostOnloadPath *string `json:"hostOnloadPath,omitempty"`

	// +optional
	// BaseMountPath is a prefix to be applied to all Onload file mounts in the
	// container's filesystem.
	// +kubebuilder:default=/opt/onload
	BaseMountPath *string `json:"baseMountPath,omitempty"`

	// +optional
	// BinMountPath is the location to mount Onload binaries in the container's
	// filesystem.
	// +kubebuilder:default=/usr/bin
	BinMountPath *string `json:"binMountPath,omitempty"`

	// +optional
	// LibMountPath is the location to mount Onload libraries in the container's
	// filesystem.
	// +kubebuilder:default=/usr/lib64
	LibMountPath *string `json:"libMounthPath,omitempty"`
}

// Spec is the top-level specification for Onload and related products that are
// controlled by the Onload Operator
type Spec struct {
	// Onload is the specification of the version of Onload to be used by this
	// CR
	Onload OnloadSpec `json:"onload"`

	// DevicePlugin is further specification for the Onload Device Plugin which
	// uses the device plugin framework to provide an `amd.com/onload` resource.
	// Image location is not configured here; see Onload Operator deployment.
	DevicePlugin DevicePluginSpec `json:"devicePlugin"`

	// Selector defines the set of nodes that this Onload CR will run on.
	Selector map[string]string `json:"selector"`

	// ServiceAccountName is the name of the service account that the objects
	// created by the Onload Operator will use.
	ServiceAccountName string `json:"serviceAccountName"`
}

// OnloadStatus defines the observed state of Onload
type OnloadStatus struct {
}

type DevicePluginStatus struct {
}

// Status contains the statuses for Onload and related products that are
// controlled by the Onload Operator
type Status struct {
	// Conditions store the status conditions of Onload
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Status of Onload components
	Onload OnloadStatus `json:"onload"`

	// Status of Onload Device Plugin
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
