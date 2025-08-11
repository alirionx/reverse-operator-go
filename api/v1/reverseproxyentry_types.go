/*
Copyright 2025.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Target struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Items=pattern=`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4}$`
	Endpoints []string `json:"endpoints"`
	// +kubebuilder:validation:Required
	Port int `json:"port"`
}
type Ingress struct {
	// +kubebuilder:default="nginx"
	ClassName string `json:"className"`
	// +kubebuilder:default="http"
	TargetProtocol string `json:"targetProtocol"`
	// +kubebuilder:validation:Required
	Host *string `json:"host"`
	// +kubebuilder:default=false
	TLS         bool              `json:"tls"`
	SecretName  string            `json:"secretName,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
type ReverseProxyEntrySpec struct {
	Target  Target  `json:"target"`
	Ingress Ingress `json:"ingress"`
}

// ReverseProxyEntryStatus defines the observed state of ReverseProxyEntry.
type ReverseProxyEntryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ReverseProxyEntry is the Schema for the reverseproxyentries API
type ReverseProxyEntry struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ReverseProxyEntry
	// +required
	Spec ReverseProxyEntrySpec `json:"spec"`

	// status defines the observed state of ReverseProxyEntry
	// +optional
	Status ReverseProxyEntryStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ReverseProxyEntryList contains a list of ReverseProxyEntry
type ReverseProxyEntryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReverseProxyEntry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReverseProxyEntry{}, &ReverseProxyEntryList{})
}
