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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SourceSpec defines the desired state of Source
type SourceSpec struct {
	URL string `json:"url"`
	// +optional
	SkipSync bool `json:"skipSync"`
	// TODO make this pull from a secret
	// +optional
	Username string `json:"username"`
	// +optional
	Password string `json:"password"`
	// TODO make this pull from a secret
	// +optional
	CertFile string `json:"certFile"`
	// +optional
	KeyFile string `json:"keyFile"`
	// +optional
	CAFile string `json:"caFile"`

	// Duration to sleep after updating before running again. This is a naive frequency, it doesn't make any guarantees
	// about the time between updates.
	UpdateFrequency string `json:"updateFrequency,omitempty"`
}

// SourceStatus defines the observed state of Source
type SourceStatus struct {
	State SourceSyncState `json:"state"`
	// +optional
	Reason     string      `json:"reason"`
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	// +optional
	AppCount int `json:"appCount"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Source sync state"
// +kubebuilder:printcolumn:name="App Count",type=integer,JSONPath=`.status.appCount`
// +kubebuilder:printcolumn:name="Last Update",type=date,JSONPath=`.status.lastUpdate`
// +kubebuilder:printcolumn:name="Update Frequency",type=string,JSONPath=`.spec.updateFrequency`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Source is the Schema for the sources API
type Source struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SourceSpec   `json:"spec,omitempty"`
	Status SourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SourceList contains a list of Source
type SourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Source `json:"items"`
}

type SourceSyncState string

const (
	SyncStateSuccess  SourceSyncState = "success"
	SyncStateError    SourceSyncState = "error"
	SyncStateUnknown  SourceSyncState = "unknown"
	SyncStateUpdating SourceSyncState = "updating"
)

func init() {
	SchemeBuilder.Register(&Source{}, &SourceList{})
}
