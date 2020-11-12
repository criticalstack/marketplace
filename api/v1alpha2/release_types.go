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
	"k8s.io/apimachinery/pkg/runtime"
)

// ReleaseSpec defines the desired state of Release
type ReleaseSpec struct {
	// Name is the name of the release
	Name string `json:"name"`
	// Info provides information about a release
	Info *Info `json:"info,omitempty"`
	// Chart is the chart that was released.
	Chart *Chart `json:"chart,omitempty"`
	// Config is the set of extra Values added to the chart.
	// These values override the default values inside of the chart.
	Config *runtime.RawExtension `json:"config,omitempty"`
	// Manifest is the string representation of the rendered template.
	Manifest string `json:"manifest,omitempty"`
	// Hooks are all of the hooks declared for this release.
	Hooks []*Hook `json:"hooks,omitempty"`
	// Version is an int which represents the version of the release.
	Version int `json:"version"`
	// Namespace is the kubernetes namespace of the release.
	Namespace string `json:"namespace"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="App",type="string",JSONPath=".spec.chart.metadata.name",description="App Name"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.chart.metadata.version",description="App Version"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".spec.info.status",description="Deployment Status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Release is the Schema for the releases API
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ReleaseSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

// Info describes release information.
type Info struct {
	// FirstDeployed is when the release was first deployed.
	FirstDeployed string `json:"first_deployed,omitempty"`
	// LastDeployed is when the release was last deployed.
	LastDeployed string `json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted string `json:"deleted,omitempty"`
	// Description is human-friendly "log entry" about this release.
	Description string `json:"description,omitempty"`
	// Status is the current state of the release
	Status Status `json:"status,omitempty"`
	// Contains the rendered templates/NOTES.txt if available
	Notes string `json:"notes,omitempty"`
}

// Status is the status of a release
type Status string

// Describe the status of a release
// NOTE: Make sure to update cmd/helm/status.go when adding or modifying any of these statuses.
const (
	// StatusUnknown indicates that a release is in an uncertain state.
	StatusUnknown Status = "unknown"
	// StatusDeployed indicates that the release has been pushed to Kubernetes.
	StatusDeployed Status = "deployed"
	// StatusUninstalled indicates that a release has been uninstalled from Kubernetes.
	StatusUninstalled Status = "uninstalled"
	// StatusSuperseded indicates that this release object is outdated and a newer one exists.
	StatusSuperseded Status = "superseded"
	// StatusFailed indicates that the release was not successfully deployed.
	StatusFailed Status = "failed"
	// StatusUninstalling indicates that a uninstall operation is underway.
	StatusUninstalling Status = "uninstalling"
	// StatusPendingInstall indicates that an install operation is underway.
	StatusPendingInstall Status = "pending-install"
	// StatusPendingUpgrade indicates that an upgrade operation is underway.
	StatusPendingUpgrade Status = "pending-upgrade"
	// StatusPendingRollback indicates that an rollback operation is underway.
	StatusPendingRollback Status = "pending-rollback"
)

// Chart is a helm package that contains metadata, a default config, zero or more
// optionally parameterizable templates, and zero or more charts (dependencies).
type Chart struct {
	// Metadata is the contents of the Chartfile.
	Metadata *Metadata `json:"metadata"`
	// Lock is the contents of Chart.lock.
	Lock *Lock `json:"lock,omitempty"`
	// Templates for this chart.
	Templates []*File `json:"templates,omitempty"`
	// Schema is an optional JSON schema for imposing structure on Values
	Schema []byte `json:"schema,omitempty"`
}

type File struct {
	// Name is the path-like name of the template.
	Name string `json:"name"`
	// Data is the template as byte data.
	Data []byte `json:"data"`
}

// Lock is a lock file for dependencies.
//
// It represents the state that the dependencies should be in.
type Lock struct {
	// Generated is the date the lock file was last generated.
	Generated string `json:"generated,omitempty"`
	// Digest is a hash of the dependencies in Chart.yaml.
	Digest string `json:"digest"`
	// Dependencies is the list of dependencies that this lock file has locked.
	Dependencies []*ChartDependency `json:"dependencies"`
}

// Metadata for a Chart file. This models the structure of a Chart.yaml file.
type Metadata struct {
	// The name of the chart
	Name string `json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	Sources []string `json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart
	Version string `json:"version,omitempty"`
	// A one-sentence description of the chart
	Description string `json:"description,omitempty"`
	// A list of string keywords
	Keywords []string `json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	Maintainers []*ChartMaintainer `json:"maintainers,omitempty"`
	// The URL to an icon file.
	Icon string `json:"icon,omitempty"`
	// The API Version of this chart.
	APIVersion string `json:"apiVersion,omitempty"`
	// The condition to check to enable chart
	Condition string `json:"condition,omitempty"`
	// The tags to check to enable chart
	Tags string `json:"tags,omitempty"`
	// The version of the application enclosed inside of this chart.
	AppVersion string `json:"appVersion,omitempty"`
	// Whether or not this chart is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
	// Annotations are additional mappings uninterpreted by Helm,
	// made available for inspection by other applications.
	Annotations map[string]string `json:"annotations,omitempty"`
	// KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion,omitempty"`
	// Dependencies are a list of dependencies for a chart.
	Dependencies []*ChartDependency `json:"dependencies,omitempty"`
	// Specifies the chart type: application or library
	Type string `json:"type,omitempty"`
}

// Maintainer describes a Chart maintainer.
type ChartMaintainer struct {
	// Name is a user name or organization name
	Name string `json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	Email string `json:"email,omitempty"`
	// URL is an optional URL to an address for the named maintainer
	URL string `json:"url,omitempty"`
}

// Dependency describes a chart upon which another chart depends.
//
// Dependencies can be used to express developer intent, or to capture the state
// of a chart.
type ChartDependency struct {
	// Name is the name of the dependency.
	//
	// This must mach the name in the dependency's Chart.yaml.
	Name string `json:"name"`
	// Version is the version (range) of this chart.
	//
	// A lock file will always produce a single version, while a dependency
	// may contain a semantic version range.
	Version string `json:"version,omitempty"`
	// The URL to the repository.
	//
	// Appending `index.yaml` to this string should result in a URL that can be
	// used to fetch the repository index.
	Repository string `json:"repository"`
	// A yaml path that resolves to a boolean, used for enabling/disabling charts (e.g. subchart1.enabled )
	Condition string `json:"condition,omitempty"`
	// Tags can be used to group charts for enabling/disabling together
	Tags []string `json:"tags,omitempty"`
	// Enabled bool determines if chart should be loaded
	Enabled bool `json:"enabled,omitempty"`
	// ImportValues holds the mapping of source values to parent key to be imported. Each item can be a
	// string or pair of child/parent sublist items.
	ImportValues []byte `json:"import-values,omitempty"`
	// Alias usable alias to be used for the chart
	Alias string `json:"alias,omitempty"`
}

// Hook defines a hook object.
type Hook struct {
	Name string `json:"name,omitempty"`
	// Kind is the Kubernetes kind.
	Kind string `json:"kind,omitempty"`
	// Path is the chart-relative path to the template.
	Path string `json:"path,omitempty"`
	// Manifest is the manifest contents.
	Manifest string `json:"manifest,omitempty"`
	// Events are the events that this hook fires on.
	Events []HookEvent `json:"events,omitempty"`
	// LastRun indicates the date/time this was last run.
	LastRun HookExecution `json:"last_run,omitempty"`
	// Weight indicates the sort order for execution among similar Hook type
	Weight int `json:"weight,omitempty"`
	// DeletePolicies are the policies that indicate when to delete the hook
	DeletePolicies []HookDeletePolicy `json:"delete_policies,omitempty"`
}

// HookEvent specifies the hook event
type HookEvent string

// Hook event types
const (
	HookPreInstall   HookEvent = "pre-install"
	HookPostInstall  HookEvent = "post-install"
	HookPreDelete    HookEvent = "pre-delete"
	HookPostDelete   HookEvent = "post-delete"
	HookPreUpgrade   HookEvent = "pre-upgrade"
	HookPostUpgrade  HookEvent = "post-upgrade"
	HookPreRollback  HookEvent = "pre-rollback"
	HookPostRollback HookEvent = "post-rollback"
	HookTest         HookEvent = "test"
)

// A HookExecution records the result for the last execution of a hook for a given release.
type HookExecution struct {
	// StartedAt indicates the date/time this hook was started
	StartedAt string `json:"started_at,omitempty"`
	// CompletedAt indicates the date/time this hook was completed.
	CompletedAt string `json:"completed_at,omitempty"`
	// Phase indicates whether the hook completed successfully
	Phase HookPhase `json:"phase"`
}

// A HookPhase indicates the state of a hook execution
type HookPhase string

const (
	// HookPhaseUnknown indicates that a hook is in an unknown state
	HookPhaseUnknown HookPhase = "Unknown"
	// HookPhaseRunning indicates that a hook is currently executing
	HookPhaseRunning HookPhase = "Running"
	// HookPhaseSucceeded indicates that hook execution succeeded
	HookPhaseSucceeded HookPhase = "Succeeded"
	// HookPhaseFailed indicates that hook execution failed
	HookPhaseFailed HookPhase = "Failed"
)

// HookDeletePolicy specifies the hook delete policy
type HookDeletePolicy string

// Hook delete policy types
const (
	HookSucceeded          HookDeletePolicy = "hook-succeeded"
	HookFailed             HookDeletePolicy = "hook-failed"
	HookBeforeHookCreation HookDeletePolicy = "before-hook-creation"
)

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
