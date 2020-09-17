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

// +kubebuilder:validation:Enum=Creating,Created,Failure
type BrokerStatusValue string

const (
	BrokerOnline  = "Online"
	BrokerOffline = "Offline"
)

// +kubebuilder:validation:Enum=kafka
type BrokerType string

const (
	KafkaBroker BrokerType = "kafka"
)

// BrokerSpec defines the desired state of Broker
type BrokerSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Required
	Type BrokerType `json:"type,omitempty"`

	// +kubebuilder:validation:Required
	Configuration BrokerConfiguration `json:"configuration"`
}

type BrokerConfiguration struct {
	// +kubebuilder:validation:Required
	BootstrapServers []string `json:"bootstrapServers,omitempty"`
}

// BrokerStatus defines the observed state of Broker
type BrokerStatus struct {
	Status BrokerStatusValue
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Broker is the Schema for the brokers API
type Broker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BrokerSpec   `json:"spec,omitempty"`
	Status BrokerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BrokerList contains a list of Broker
type BrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Broker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Broker{}, &BrokerList{})
}
