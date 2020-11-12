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
	"bytes"
	"context"
	"strings"

	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	marketplacev1alpha2 "github.com/criticalstack/marketplace/api/v1alpha2"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=marketplace.criticalstack.com,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marketplace.criticalstack.com,resources=releases/status,verbs=get;update;patch

func (r *ReleaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("secret", req.NamespacedName)

	log.Info("reconcile release")

	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get secret")
		return ctrl.Result{}, err
	}

	if secret.Type != "helm.sh/release.v1" {
		return ctrl.Result{}, nil
	}

	decodedSecret, err := decodeSecretRelease(secret)
	if err != nil {
		log.Error(err, "unable to decode Secret")
		return ctrl.Result{}, err
	}

	if decodedSecret.Info.Status == marketplacev1alpha2.StatusSuperseded {
		return ctrl.Result{}, nil
	}

	var release marketplacev1alpha2.Release
	release.Namespace = secret.Namespace
	release.Labels = make(map[string]string)
	for k, v := range secret.Labels {
		if strings.HasPrefix(k, "marketplace.criticalstack.com/") {
			release.Labels[k] = v
		}
	}
	release.Name = secret.Labels["name"]
	if release.Name == "" {
		log.V(1).Info("unable to get release name from secret")
		return ctrl.Result{}, nil
	}

	release.Spec = decodedSecret
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, &release, controllerutil.MutateFn(func() error {
		if err := controllerutil.SetOwnerReference(&secret, &release, r.Scheme); err != nil {
			return err
		}
		release.Spec = decodedSecret

		return nil
	}))
	if err != nil {
		log.Error(err, "unable to create or update Release CRD")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func decodeSecretRelease(secret corev1.Secret) (marketplacev1alpha2.ReleaseSpec, error) {
	var releaseSpec marketplacev1alpha2.ReleaseSpec

	encodedRelease := secret.Data["release"]
	strRel := string(encodedRelease)
	dst, err := base64.StdEncoding.DecodeString(strRel)
	if err != nil {
		return releaseSpec, err
	}

	reader, err := gzip.NewReader(bytes.NewBuffer(dst))
	if err != nil {
		return releaseSpec, err
	}
	defer reader.Close()

	bRel, err := ioutil.ReadAll(reader)
	if err != nil {
		return releaseSpec, err
	}

	if err := json.Unmarshal(bRel, &releaseSpec); err != nil {
		return releaseSpec, err
	}

	return releaseSpec, nil
}

func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Owns(&marketplacev1alpha2.Release{}).
		Complete(r)
}
