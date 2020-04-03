// +build !ignore_autogenerated

/*
Copyright 2019 The Tekton Authors

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CELInterceptor) DeepCopyInto(out *CELInterceptor) {
	*out = *in
	if in.Overlays != nil {
		in, out := &in.Overlays, &out.Overlays
		*out = make([]CELOverlay, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CELInterceptor.
func (in *CELInterceptor) DeepCopy() *CELInterceptor {
	if in == nil {
		return nil
	}
	out := new(CELInterceptor)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CELOverlay) DeepCopyInto(out *CELOverlay) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CELOverlay.
func (in *CELOverlay) DeepCopy() *CELOverlay {
	if in == nil {
		return nil
	}
	out := new(CELOverlay)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTriggerBinding) DeepCopyInto(out *ClusterTriggerBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTriggerBinding.
func (in *ClusterTriggerBinding) DeepCopy() *ClusterTriggerBinding {
	if in == nil {
		return nil
	}
	out := new(ClusterTriggerBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTriggerBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTriggerBindingList) DeepCopyInto(out *ClusterTriggerBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterTriggerBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTriggerBindingList.
func (in *ClusterTriggerBindingList) DeepCopy() *ClusterTriggerBindingList {
	if in == nil {
		return nil
	}
	out := new(ClusterTriggerBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTriggerBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventInterceptor) DeepCopyInto(out *EventInterceptor) {
	*out = *in
	if in.Webhook != nil {
		in, out := &in.Webhook, &out.Webhook
		*out = new(WebhookInterceptor)
		(*in).DeepCopyInto(*out)
	}
	if in.GitHub != nil {
		in, out := &in.GitHub, &out.GitHub
		*out = new(GitHubInterceptor)
		(*in).DeepCopyInto(*out)
	}
	if in.GitLab != nil {
		in, out := &in.GitLab, &out.GitLab
		*out = new(GitLabInterceptor)
		(*in).DeepCopyInto(*out)
	}
	if in.CEL != nil {
		in, out := &in.CEL, &out.CEL
		*out = new(CELInterceptor)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventInterceptor.
func (in *EventInterceptor) DeepCopy() *EventInterceptor {
	if in == nil {
		return nil
	}
	out := new(EventInterceptor)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListener) DeepCopyInto(out *EventListener) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListener.
func (in *EventListener) DeepCopy() *EventListener {
	if in == nil {
		return nil
	}
	out := new(EventListener)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EventListener) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListenerBinding) DeepCopyInto(out *EventListenerBinding) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListenerBinding.
func (in *EventListenerBinding) DeepCopy() *EventListenerBinding {
	if in == nil {
		return nil
	}
	out := new(EventListenerBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListenerConfig) DeepCopyInto(out *EventListenerConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListenerConfig.
func (in *EventListenerConfig) DeepCopy() *EventListenerConfig {
	if in == nil {
		return nil
	}
	out := new(EventListenerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListenerList) DeepCopyInto(out *EventListenerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EventListener, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListenerList.
func (in *EventListenerList) DeepCopy() *EventListenerList {
	if in == nil {
		return nil
	}
	out := new(EventListenerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EventListenerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListenerSpec) DeepCopyInto(out *EventListenerSpec) {
	*out = *in
	if in.Triggers != nil {
		in, out := &in.Triggers, &out.Triggers
		*out = make([]EventListenerTrigger, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListenerSpec.
func (in *EventListenerSpec) DeepCopy() *EventListenerSpec {
	if in == nil {
		return nil
	}
	out := new(EventListenerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListenerStatus) DeepCopyInto(out *EventListenerStatus) {
	*out = *in
	in.Status.DeepCopyInto(&out.Status)
	in.AddressStatus.DeepCopyInto(&out.AddressStatus)
	out.Configuration = in.Configuration
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListenerStatus.
func (in *EventListenerStatus) DeepCopy() *EventListenerStatus {
	if in == nil {
		return nil
	}
	out := new(EventListenerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListenerTemplate) DeepCopyInto(out *EventListenerTemplate) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListenerTemplate.
func (in *EventListenerTemplate) DeepCopy() *EventListenerTemplate {
	if in == nil {
		return nil
	}
	out := new(EventListenerTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventListenerTrigger) DeepCopyInto(out *EventListenerTrigger) {
	*out = *in
	if in.Bindings != nil {
		in, out := &in.Bindings, &out.Bindings
		*out = make([]*EventListenerBinding, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(EventListenerBinding)
				**out = **in
			}
		}
	}
	out.Template = in.Template
	if in.Interceptors != nil {
		in, out := &in.Interceptors, &out.Interceptors
		*out = make([]*EventInterceptor, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(EventInterceptor)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.ServiceAccount != nil {
		in, out := &in.ServiceAccount, &out.ServiceAccount
		*out = new(v1.ObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventListenerTrigger.
func (in *EventListenerTrigger) DeepCopy() *EventListenerTrigger {
	if in == nil {
		return nil
	}
	out := new(EventListenerTrigger)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitHubInterceptor) DeepCopyInto(out *GitHubInterceptor) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(SecretRef)
		**out = **in
	}
	if in.EventTypes != nil {
		in, out := &in.EventTypes, &out.EventTypes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitHubInterceptor.
func (in *GitHubInterceptor) DeepCopy() *GitHubInterceptor {
	if in == nil {
		return nil
	}
	out := new(GitHubInterceptor)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitLabInterceptor) DeepCopyInto(out *GitLabInterceptor) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(SecretRef)
		**out = **in
	}
	if in.EventTypes != nil {
		in, out := &in.EventTypes, &out.EventTypes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitLabInterceptor.
func (in *GitLabInterceptor) DeepCopy() *GitLabInterceptor {
	if in == nil {
		return nil
	}
	out := new(GitLabInterceptor)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretRef) DeepCopyInto(out *SecretRef) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretRef.
func (in *SecretRef) DeepCopy() *SecretRef {
	if in == nil {
		return nil
	}
	out := new(SecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerBinding) DeepCopyInto(out *TriggerBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerBinding.
func (in *TriggerBinding) DeepCopy() *TriggerBinding {
	if in == nil {
		return nil
	}
	out := new(TriggerBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TriggerBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerBindingList) DeepCopyInto(out *TriggerBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TriggerBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerBindingList.
func (in *TriggerBindingList) DeepCopy() *TriggerBindingList {
	if in == nil {
		return nil
	}
	out := new(TriggerBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TriggerBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerBindingSpec) DeepCopyInto(out *TriggerBindingSpec) {
	*out = *in
	if in.Params != nil {
		in, out := &in.Params, &out.Params
		*out = make([]v1beta1.Param, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerBindingSpec.
func (in *TriggerBindingSpec) DeepCopy() *TriggerBindingSpec {
	if in == nil {
		return nil
	}
	out := new(TriggerBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerBindingStatus) DeepCopyInto(out *TriggerBindingStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerBindingStatus.
func (in *TriggerBindingStatus) DeepCopy() *TriggerBindingStatus {
	if in == nil {
		return nil
	}
	out := new(TriggerBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerResourceTemplate) DeepCopyInto(out *TriggerResourceTemplate) {
	*out = *in
	in.RawExtension.DeepCopyInto(&out.RawExtension)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerResourceTemplate.
func (in *TriggerResourceTemplate) DeepCopy() *TriggerResourceTemplate {
	if in == nil {
		return nil
	}
	out := new(TriggerResourceTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerTemplate) DeepCopyInto(out *TriggerTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerTemplate.
func (in *TriggerTemplate) DeepCopy() *TriggerTemplate {
	if in == nil {
		return nil
	}
	out := new(TriggerTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TriggerTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerTemplateList) DeepCopyInto(out *TriggerTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TriggerTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerTemplateList.
func (in *TriggerTemplateList) DeepCopy() *TriggerTemplateList {
	if in == nil {
		return nil
	}
	out := new(TriggerTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TriggerTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerTemplateSpec) DeepCopyInto(out *TriggerTemplateSpec) {
	*out = *in
	if in.Params != nil {
		in, out := &in.Params, &out.Params
		*out = make([]v1beta1.ParamSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ResourceTemplates != nil {
		in, out := &in.ResourceTemplates, &out.ResourceTemplates
		*out = make([]TriggerResourceTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerTemplateSpec.
func (in *TriggerTemplateSpec) DeepCopy() *TriggerTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(TriggerTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerTemplateStatus) DeepCopyInto(out *TriggerTemplateStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerTemplateStatus.
func (in *TriggerTemplateStatus) DeepCopy() *TriggerTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(TriggerTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookInterceptor) DeepCopyInto(out *WebhookInterceptor) {
	*out = *in
	if in.ObjectRef != nil {
		in, out := &in.ObjectRef, &out.ObjectRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
	if in.Header != nil {
		in, out := &in.Header, &out.Header
		*out = make([]v1beta1.Param, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookInterceptor.
func (in *WebhookInterceptor) DeepCopy() *WebhookInterceptor {
	if in == nil {
		return nil
	}
	out := new(WebhookInterceptor)
	in.DeepCopyInto(out)
	return out
}
