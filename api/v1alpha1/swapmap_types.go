/*
Copyright 2023 The Webroot, Inc.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SwapRef defines the information to reference one or more images to be swapped
type SwapRef struct {
	// Registry is the registry to target (e.g. "docker.io", "quay.io", "ghcr.io")
	// +kubebuilder:validation:Optional
	Registry string `json:"registry"`
	// Project is the project to target (e.g. "nginx", "library", "team1/project2")
	// +kubebuilder:validation:Optional
	Project string `json:"project"`
	// Image is the image to target (e.g. "nginx", "nginx:latest", "nginx:1.19.6")
	// +kubebuilder:validation:Optional
	Image string `json:"image"`
}

// Map defines a single swap map
type Map struct {
	// Name is the name of the swap map
	// +kubebuilder:validation:Required
	// +kubebuilder:default="default"
	Name string `json:"name"`
	// Type is the type of swap map (e.g. "default", "swap", "exact", "replace")
	// +kubebuilder:default="swap"
	// +kubebuilder:validation:Enum={"default","swap","exact","replace"}
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// SwapFrom defines the information to target one or more images to be swapped
	// +kubebuilder:validation:Optional
	SwapFrom SwapRef `json:"swapFrom,omitempty"`
	// SwapTo defines how the target image(s) should be swapped
	// +kubebuilder:validation:Optional
	SwapTo SwapRef `json:"swapTo,omitempty"`
	// Wildcards is a list of wildcard strings used to greedy match target one or more target images
	// +kubebuilder:validation:Optional
	Wildcards []string `json:"wildcards,omitempty"`
	// NoSwap is a boolean that, when true, prevents swapping of the target image(s)
	// +kubebuilder:validation:Optional
	NoSwap bool `json:"noSwap,omitempty"`
}

// SwapMapSpec defines the desired state of SwapMap
type SwapMapSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Maps is a list of Swap mappings to control how ImageSwap operates
	// +kubebuilder:validation:Required
	// +listType=map
	// +listMapKey=name
	Maps []Map `json:"maps"`
}

// SwapMapStatus defines the observed state of SwapMap
type SwapMapStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SwapMap is the Schema for the swapmaps API
// +kubebuilder:resource:shortName=sm,singular=swapmap,scope=Namespaced,categories={"all","imageswap","imgswap","imgswp"}
type SwapMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwapMapSpec   `json:"spec,omitempty"`
	Status SwapMapStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SwapMapList contains a list of SwapMap
type SwapMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwapMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwapMap{}, &SwapMapList{})
}
