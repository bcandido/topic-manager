/*


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

type TopicStatusValue string

const (
	TopicStatusOutOfSync TopicStatusValue = "OutOfSync"
	TopicStatusCreated   TopicStatusValue = "Created"
	TopicStatusFailure   TopicStatusValue = "Failure"
)

type TopicConfiguration struct {
	// +kubebuilder:validation:Required
	Partitions int `json:"partitions,omitempty"`

	// +kubebuilder:validation:Required
	ReplicationFactor int `json:"replicationFactor,omitempty"`
}

// TopicSpec defines the desired state of Topic
type TopicSpec struct {
	// +kubebuilder:validation:Required
	Broker string `json:"broker,omitempty"`

	// +kubebuilder:validation:Required
	Configuration TopicConfiguration `json:"configuration,omitempty"`
}

// TopicStatus defines the observed state of Topic
type TopicStatus struct {
	Status TopicStatusValue `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Broker",type=string,JSONPath=`.spec.broker`
// +kubebuilder:printcolumn:name="Partitions",type=string,JSONPath=`.spec.configuration.partitions`
// +kubebuilder:printcolumn:name="Replication",type=string,JSONPath=`.spec.configuration.replicationFactor`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`

// Topic is the Schema for the topics API
type Topic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TopicSpec   `json:"spec,omitempty"`
	Status TopicStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TopicList contains a list of Topic
type TopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Topic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Topic{}, &TopicList{})
}
