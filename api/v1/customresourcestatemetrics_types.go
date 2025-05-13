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
	"k8s.io/apimachinery/pkg/runtime"
	// ksm "k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

// +kubebuilder:object:root=true

// CustomResourceStateMetricsList contains a list of CustomResourceStateMetrics.
type CustomResourceStateMetricsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomResourceStateMetrics `json:"items"`
}

//nolint:lll
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=ksm,shortName=crsm
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status",description="Ready condition"

// CustomResourceStateMetrics is the Schema for the customresourcestatemetrics API.
type CustomResourceStateMetrics struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the CustomResourceStateMetrics resource.
	Spec CustomResourceStateMetricsSpec `json:"spec,omitempty"`

	// Status of the CustomResourceStateMetrics resource.
	Status CustomResourceStateMetricsStatus `json:"status,omitempty"`
}

// CustomResourceStateMetricsSpec defines the desired state of CustomResourceStateMetrics.
type CustomResourceStateMetricsSpec struct {
	// Details of the ConfigMap where the resources will be written into.
	ConfigMap CustomResourceStateMetricsConfigMap `json:"configMap"`

	// List of custom resources to be monitored. The content list items can
	// be arbitrary object that should follow the structure described in the
	// kube-state-metrics exporter
	// (https://github.com/kubernetes/kube-state-metrics/blob/main/docs/metrics/extend/customresourcestate-metrics.md).
	// This operator doesn't analyze nor modifies its content. It just
	// writes its content into a ConfigMap as is. This is mainly because the kube-state-metrics package
	// lacks the "omitempty" JSON tag flag as well as the "DeepCopy*"
	// methods for the individual types.
	Resources []runtime.RawExtension `json:"resources,omitempty"`

	// Resources []ksm.Resource `json:"resources,omitempty"`
}

type CustomResourceStateMetricsConfigMap struct {
	// Name of the ConfigMap where the resources will be written into.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Namespace of the ConfigMap where the resources will be written into.
	// If not specified, the Namespace of the CustomResourceStateMetrics
	// will be used instead.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Namespace string `json:"namespace,omitempty"`

	// ConfigMap key under which the CustomResourceStateMetrics resources
	// are stored. Default: config.yaml.
	// +kubebuilder:default=config.yaml
	Key string `json:"key,omitempty"`
}

// CustomResourceStateMetricsStatus defines the observed state of CustomResourceStateMetrics.
type CustomResourceStateMetricsStatus struct {
	// State conditions that will indicate whether the resource is ready to
	// be used in the destination ConfigMap.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func init() {
	SchemeBuilder.Register(&CustomResourceStateMetrics{}, &CustomResourceStateMetricsList{})
}
