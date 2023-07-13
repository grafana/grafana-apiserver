//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by deepcopy-gen. DO NOT EDIT.

package kinds

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceDefinition) DeepCopyInto(out *GrafanaResourceDefinition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceDefinition.
func (in *GrafanaResourceDefinition) DeepCopy() *GrafanaResourceDefinition {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceDefinition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GrafanaResourceDefinition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceDefinitionCondition) DeepCopyInto(out *GrafanaResourceDefinitionCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceDefinitionCondition.
func (in *GrafanaResourceDefinitionCondition) DeepCopy() *GrafanaResourceDefinitionCondition {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceDefinitionCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceDefinitionList) DeepCopyInto(out *GrafanaResourceDefinitionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GrafanaResourceDefinition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceDefinitionList.
func (in *GrafanaResourceDefinitionList) DeepCopy() *GrafanaResourceDefinitionList {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceDefinitionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GrafanaResourceDefinitionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceDefinitionNames) DeepCopyInto(out *GrafanaResourceDefinitionNames) {
	*out = *in
	if in.ShortNames != nil {
		in, out := &in.ShortNames, &out.ShortNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Categories != nil {
		in, out := &in.Categories, &out.Categories
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceDefinitionNames.
func (in *GrafanaResourceDefinitionNames) DeepCopy() *GrafanaResourceDefinitionNames {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceDefinitionNames)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceDefinitionSpec) DeepCopyInto(out *GrafanaResourceDefinitionSpec) {
	*out = *in
	in.Names.DeepCopyInto(&out.Names)
	if in.Subresources != nil {
		in, out := &in.Subresources, &out.Subresources
		*out = new(GrafanaResourceSubresources)
		(*in).DeepCopyInto(*out)
	}
	if in.Versions != nil {
		in, out := &in.Versions, &out.Versions
		*out = make([]GrafanaResourceDefinitionVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PreserveUnknownFields != nil {
		in, out := &in.PreserveUnknownFields, &out.PreserveUnknownFields
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceDefinitionSpec.
func (in *GrafanaResourceDefinitionSpec) DeepCopy() *GrafanaResourceDefinitionSpec {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceDefinitionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceDefinitionStatus) DeepCopyInto(out *GrafanaResourceDefinitionStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]GrafanaResourceDefinitionCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.AcceptedNames.DeepCopyInto(&out.AcceptedNames)
	if in.StoredVersions != nil {
		in, out := &in.StoredVersions, &out.StoredVersions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceDefinitionStatus.
func (in *GrafanaResourceDefinitionStatus) DeepCopy() *GrafanaResourceDefinitionStatus {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceDefinitionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceDefinitionVersion) DeepCopyInto(out *GrafanaResourceDefinitionVersion) {
	*out = *in
	if in.DeprecationWarning != nil {
		in, out := &in.DeprecationWarning, &out.DeprecationWarning
		*out = new(string)
		**out = **in
	}
	if in.Subresources != nil {
		in, out := &in.Subresources, &out.Subresources
		*out = new(GrafanaResourceSubresources)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceDefinitionVersion.
func (in *GrafanaResourceDefinitionVersion) DeepCopy() *GrafanaResourceDefinitionVersion {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceDefinitionVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceSubresourceHistory) DeepCopyInto(out *GrafanaResourceSubresourceHistory) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceSubresourceHistory.
func (in *GrafanaResourceSubresourceHistory) DeepCopy() *GrafanaResourceSubresourceHistory {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceSubresourceHistory)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceSubresourceRef) DeepCopyInto(out *GrafanaResourceSubresourceRef) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceSubresourceRef.
func (in *GrafanaResourceSubresourceRef) DeepCopy() *GrafanaResourceSubresourceRef {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceSubresourceRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceSubresourceScale) DeepCopyInto(out *GrafanaResourceSubresourceScale) {
	*out = *in
	if in.LabelSelectorPath != nil {
		in, out := &in.LabelSelectorPath, &out.LabelSelectorPath
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceSubresourceScale.
func (in *GrafanaResourceSubresourceScale) DeepCopy() *GrafanaResourceSubresourceScale {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceSubresourceScale)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceSubresourceStatus) DeepCopyInto(out *GrafanaResourceSubresourceStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceSubresourceStatus.
func (in *GrafanaResourceSubresourceStatus) DeepCopy() *GrafanaResourceSubresourceStatus {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceSubresourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaResourceSubresources) DeepCopyInto(out *GrafanaResourceSubresources) {
	*out = *in
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(GrafanaResourceSubresourceStatus)
		**out = **in
	}
	if in.Scale != nil {
		in, out := &in.Scale, &out.Scale
		*out = new(GrafanaResourceSubresourceScale)
		(*in).DeepCopyInto(*out)
	}
	if in.History != nil {
		in, out := &in.History, &out.History
		*out = new(GrafanaResourceSubresourceHistory)
		**out = **in
	}
	if in.Ref != nil {
		in, out := &in.Ref, &out.Ref
		*out = new(GrafanaResourceSubresourceRef)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaResourceSubresources.
func (in *GrafanaResourceSubresources) DeepCopy() *GrafanaResourceSubresources {
	if in == nil {
		return nil
	}
	out := new(GrafanaResourceSubresources)
	in.DeepCopyInto(out)
	return out
}
