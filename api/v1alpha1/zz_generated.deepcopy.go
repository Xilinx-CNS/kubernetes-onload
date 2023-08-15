//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DevicePluginBuildSpec) DeepCopyInto(out *DevicePluginBuildSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DevicePluginBuildSpec.
func (in *DevicePluginBuildSpec) DeepCopy() *DevicePluginBuildSpec {
	if in == nil {
		return nil
	}
	out := new(DevicePluginBuildSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DevicePluginSpec) DeepCopyInto(out *DevicePluginSpec) {
	*out = *in
	if in.DevicePluginBuild != nil {
		in, out := &in.DevicePluginBuild, &out.DevicePluginBuild
		*out = new(DevicePluginBuildSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DevicePluginSpec.
func (in *DevicePluginSpec) DeepCopy() *DevicePluginSpec {
	if in == nil {
		return nil
	}
	out := new(DevicePluginSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DevicePluginStatus) DeepCopyInto(out *DevicePluginStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DevicePluginStatus.
func (in *DevicePluginStatus) DeepCopy() *DevicePluginStatus {
	if in == nil {
		return nil
	}
	out := new(DevicePluginStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Onload) DeepCopyInto(out *Onload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Onload.
func (in *Onload) DeepCopy() *Onload {
	if in == nil {
		return nil
	}
	out := new(Onload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Onload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OnloadKernelBuildSpec) DeepCopyInto(out *OnloadKernelBuildSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OnloadKernelBuildSpec.
func (in *OnloadKernelBuildSpec) DeepCopy() *OnloadKernelBuildSpec {
	if in == nil {
		return nil
	}
	out := new(OnloadKernelBuildSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OnloadKernelMapping) DeepCopyInto(out *OnloadKernelMapping) {
	*out = *in
	if in.Build != nil {
		in, out := &in.Build, &out.Build
		*out = new(OnloadKernelBuildSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OnloadKernelMapping.
func (in *OnloadKernelMapping) DeepCopy() *OnloadKernelMapping {
	if in == nil {
		return nil
	}
	out := new(OnloadKernelMapping)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OnloadList) DeepCopyInto(out *OnloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Onload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OnloadList.
func (in *OnloadList) DeepCopy() *OnloadList {
	if in == nil {
		return nil
	}
	out := new(OnloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OnloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OnloadSpec) DeepCopyInto(out *OnloadSpec) {
	*out = *in
	if in.KernelMappings != nil {
		in, out := &in.KernelMappings, &out.KernelMappings
		*out = make([]OnloadKernelMapping, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Build != nil {
		in, out := &in.Build, &out.Build
		*out = new(OnloadUserBuildSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OnloadSpec.
func (in *OnloadSpec) DeepCopy() *OnloadSpec {
	if in == nil {
		return nil
	}
	out := new(OnloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OnloadStatus) DeepCopyInto(out *OnloadStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OnloadStatus.
func (in *OnloadStatus) DeepCopy() *OnloadStatus {
	if in == nil {
		return nil
	}
	out := new(OnloadStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OnloadUserBuildSpec) DeepCopyInto(out *OnloadUserBuildSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OnloadUserBuildSpec.
func (in *OnloadUserBuildSpec) DeepCopy() *OnloadUserBuildSpec {
	if in == nil {
		return nil
	}
	out := new(OnloadUserBuildSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Spec) DeepCopyInto(out *Spec) {
	*out = *in
	in.Onload.DeepCopyInto(&out.Onload)
	in.DevicePlugin.DeepCopyInto(&out.DevicePlugin)
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Spec.
func (in *Spec) DeepCopy() *Spec {
	if in == nil {
		return nil
	}
	out := new(Spec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Status) DeepCopyInto(out *Status) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Onload = in.Onload
	out.DevicePlugin = in.DevicePlugin
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Status.
func (in *Status) DeepCopy() *Status {
	if in == nil {
		return nil
	}
	out := new(Status)
	in.DeepCopyInto(out)
	return out
}