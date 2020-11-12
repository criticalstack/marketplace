package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=app;apps

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ApplicationSpec `json:"spec"`
}

type ApplicationSpec struct {
	// ProperName is the stylized name, i.e. "MySQL"
	ProperName  string            `json:"proper_name,omitempty"`
	AppName     string            `json:"app_name"`
	Description string            `json:"description"`
	License     string            `json:"license"`
	Author      string            `json:"author"`
	Website     string            `json:"website"`
	SourceName  string            `json:"source_name"`
	Version     string            `json:"version"`
	Categories  []string          `json:"categories"`
	Icon        string            `json:"icon"`
	Deprecated  bool              `json:"deprecated,omitempty"`
	Documents   map[string]string `json:"documents,omitempty"`
	URL         string            `json:"url"`
	// TODO(ktravis): AppVersion?
}

// +kubebuilder:object:root=true

// ApplicationList is a list of Application resources
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Application `json:"items"`
}
