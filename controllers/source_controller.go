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

package controllers

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	marketplacev1alpha2 "github.com/criticalstack/marketplace/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultCategoriesConfigMapName = "marketplace-app-categories"
)

// SourceReconciler reconciles a Source object
type SourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	recorder record.EventRecorder

	defaultCategories map[string][]string
}

func parseCategories(s string) (map[string][]string, error) {
	dec := yaml.NewYAMLOrJSONDecoder(strings.NewReader(s), 128)
	var byCategory map[string][]string
	if err := dec.Decode(&byCategory); err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for cat, apps := range byCategory {
		cat = strings.ToLower(cat)
		for _, app := range apps {
			result[app] = append(result[app], cat)
		}
	}
	return result, nil
}

func (r *SourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&marketplacev1alpha2.Source{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
	if err != nil {
		return err
	}
	r.recorder = mgr.GetEventRecorderFor("source-controller")
	return nil
}

func copyMaintainers(mm []*chart.Maintainer) (out []*marketplacev1alpha2.Maintainer) {
	for _, m := range mm {
		out = append(out, &marketplacev1alpha2.Maintainer{
			Name:  m.Name,
			Email: m.Email,
			URL:   m.URL,
		})
	}
	return
}

func copyDependencies(dd []*chart.Dependency) (out []*marketplacev1alpha2.Dependency) {
	for _, d := range dd {
		iv := make([]string, 0)
		for _, v := range d.ImportValues {
			iv = append(iv, fmt.Sprintf("%v", v))
		}
		out = append(out, &marketplacev1alpha2.Dependency{
			Name:         d.Name,
			Version:      d.Version,
			Repository:   d.Repository,
			Condition:    d.Condition,
			Tags:         d.Tags,
			Enabled:      d.Enabled,
			ImportValues: iv,
			Alias:        d.Alias,
		})
	}
	return
}

func fixURLs(log logr.Logger, src string, urls []string) []string {
	parsed, err := url.Parse(src)
	if err != nil {
		log.Error(err, "failed to process source url for app")
	}
	for i, u := range urls {
		if !strings.HasPrefix(u, src) {
			if parsed != nil {
				parsed.Path = path.Join(parsed.Path, u)
				urls[i] = parsed.String()
			} else {
				urls[i] = path.Join(src, u)
			}
		}
	}
	return urls
}

// +kubebuilder:rbac:groups=marketplace.criticalstack.com,resources=sources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marketplace.criticalstack.com,resources=sources/status,verbs=get;update;patch

func (r *SourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("source", req.NamespacedName)

	log.Info("reconcile source")

	var src marketplacev1alpha2.Source
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &src); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if src.Spec.SkipSync {
		return ctrl.Result{}, nil
	}

	var cm corev1.ConfigMap
	if err := r.Get(context.TODO(), client.ObjectKey{Name: defaultCategoriesConfigMapName, Namespace: "critical-stack"}, &cm); err == nil {
		for _, d := range cm.Data {
			cats, err := parseCategories(d)
			if err != nil {
				r.Log.Error(err, "failed to parse categories in configmap", "name", cm.Name)
				break
			}
			r.defaultCategories = cats
			break
		}
	} else if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to retrieve default categories map")
	}

	if src.Status.State != marketplacev1alpha2.SyncStateUpdating {
		return ctrl.Result{Requeue: true}, r.setSourceStatus(ctx, src, "Reconcile", marketplacev1alpha2.SourceStatus{
			State:  marketplacev1alpha2.SyncStateUpdating,
			Reason: "object changed",
		})
	}

	result := func() ctrl.Result {
		return ctrl.Result{}
	}
	if src.Spec.UpdateFrequency != "" {
		d, err := time.ParseDuration(src.Spec.UpdateFrequency)
		if err != nil {
			return ctrl.Result{}, r.setSourceStatus(ctx, src, "Reconcile", marketplacev1alpha2.SourceStatus{
				State:  marketplacev1alpha2.SyncStateError,
				Reason: fmt.Sprintf("spec.updateFrequency is invalid: %v", err),
			})
		}
		result = func() ctrl.Result {
			log.Info(fmt.Sprintf("next run in %s", d))
			return ctrl.Result{RequeueAfter: d}
		}
	}

	settings := &cli.EnvSettings{}
	entry := &repo.Entry{
		Name:     src.Name,
		URL:      src.Spec.URL,
		Username: src.Spec.Username,
		Password: src.Spec.Password,
		CertFile: src.Spec.CertFile,
		KeyFile:  src.Spec.KeyFile,
		CAFile:   src.Spec.CAFile,
	}
	cr, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return result(), r.setSourceStatus(ctx, src, "SyncRepo", marketplacev1alpha2.SourceStatus{
			State:  marketplacev1alpha2.SyncStateError,
			Reason: err.Error(),
		})
	}
	start := metav1.Now()
	idx, err := cr.DownloadIndexFile()
	if err != nil {
		return result(), r.setSourceStatus(ctx, src, "SyncRepo", marketplacev1alpha2.SourceStatus{
			State:      marketplacev1alpha2.SyncStateError,
			Reason:     err.Error(),
			LastUpdate: start,
		})
	}

	repoIndex, err := repo.LoadIndexFile(idx)
	if err != nil {
		return result(), r.setSourceStatus(ctx, src, "SyncRepo", marketplacev1alpha2.SourceStatus{
			State:      marketplacev1alpha2.SyncStateError,
			Reason:     err.Error(),
			LastUpdate: start,
		})
	}

	var existingApps marketplacev1alpha2.ApplicationList
	if err := r.List(ctx, &existingApps, client.MatchingLabels{"marketplace.criticalstack.com/source.name": src.Name}); err != nil {
		return result(), r.setSourceStatus(ctx, src, "ListApps", marketplacev1alpha2.SourceStatus{
			State:      marketplacev1alpha2.SyncStateError,
			Reason:     err.Error(),
			LastUpdate: start,
		})
	}
	have := make(map[string]marketplacev1alpha2.Application)
	for _, app := range existingApps.Items {
		have[app.Name] = app
	}

	for chartName, items := range repoIndex.Entries {
		needsUpdate := false
		name := fmt.Sprintf("%s.%s", src.Name, chartName)
		app, ok := have[name]
		if !ok {
			// create new app and add versions
			app.Name = name
			app.AppName = chartName
			app.Labels = map[string]string{
				"marketplace.criticalstack.com/source.name":      src.Name,
				"marketplace.criticalstack.com/application.name": chartName,
			}
			r.recorder.Eventf(&src, corev1.EventTypeNormal, "AppUpdate", "new app: %s", chartName)
		}

		// check apiVersion v1 vs v2
		if cats := r.defaultCategories[app.AppName]; len(cats) > 0 {
			if app.Labels == nil {
				app.Labels = make(map[string]string)
			}
			for _, c := range cats {
				app.Labels["marketplace.criticalstack.com/application.category."+c] = ""
			}
		}

	L:
		for _, cv := range items {
			for _, v := range app.Versions {
				if v.Version == cv.Version {
					continue L
				}
			}
			if ok {
				r.recorder.Eventf(&src, corev1.EventTypeNormal, "AppUpdate", "new version found: %s %s", chartName, cv.Version)
			}
			needsUpdate = true
			app.AppName = chartName

			app.Versions = append(app.Versions, marketplacev1alpha2.ChartVersion{
				Home:         cv.Home,
				Sources:      cv.Sources,
				Version:      cv.Version,
				Description:  cv.Description,
				Keywords:     cv.Keywords,
				Maintainers:  copyMaintainers(cv.Maintainers),
				Icon:         cv.Icon,
				APIVersion:   cv.APIVersion,
				AppVersion:   cv.AppVersion,
				Deprecated:   cv.Deprecated,
				Annotations:  cv.Annotations,
				KubeVersion:  cv.KubeVersion,
				Dependencies: copyDependencies(cv.Dependencies),
				Type:         cv.Type,
				URLs:         fixURLs(log, src.Spec.URL, cv.URLs),
				Created:      metav1.NewTime(cv.Created),
				Removed:      &cv.Removed,
				Digest:       cv.Digest,
			})
		}

		if needsUpdate {
			vv := app.Versions
			for _, v := range vv {
				if v.Deprecated {
					app.Labels["marketplace.criticalstack.com/app.deprecated"] = "true"
					break
				}
			}
			l := app.Labels
			_, err := ctrl.CreateOrUpdate(ctx, r.Client, &app, controllerutil.MutateFn(func() error {
				if err := ctrl.SetControllerReference(&src, &app, r.Scheme); err != nil {
					return err
				}
				app.Versions = vv
				for k, v := range l {
					app.Labels[k] = v
				}
				return nil
			}))
			if err != nil {
				return result(), r.setSourceStatus(ctx, src, "AppUpdate", marketplacev1alpha2.SourceStatus{
					State:      marketplacev1alpha2.SyncStateError,
					Reason:     err.Error(),
					LastUpdate: start,
				})
			}
		}
	}

	return result(), r.setSourceStatus(ctx, src, "Reconcile", marketplacev1alpha2.SourceStatus{
		State:      marketplacev1alpha2.SyncStateSuccess,
		LastUpdate: start,
	})
}

func (r *SourceReconciler) setSourceStatus(ctx context.Context, src marketplacev1alpha2.Source, op string, status marketplacev1alpha2.SourceStatus) error {
	old := src
	var all marketplacev1alpha2.ApplicationList
	if err := r.List(ctx, &all, client.MatchingLabels{"marketplace.criticalstack.com/source.name": src.Name}); err != nil {
		return err
	}
	status.AppCount = 0
	for _, x := range all.Items {
		if ref := metav1.GetControllerOf(&x); ref == nil || ref.Name != src.Name {
			continue
		}
		status.AppCount++
	}
	src.Status = status
	if err := r.Status().Patch(ctx, &src, client.MergeFrom(&old)); err != nil {
		return errors.Wrap(err, "FAILED during status update")
	}
	et := corev1.EventTypeNormal
	msg := "done"
	if status.State == marketplacev1alpha2.SyncStateError {
		et = corev1.EventTypeWarning
		msg = status.Reason
	}
	r.recorder.Event(&src, et, op, msg)
	return nil
}
