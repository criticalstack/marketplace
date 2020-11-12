package controllers

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"compress/gzip"
	"encoding/base64"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	marketplacev1alpha2 "github.com/criticalstack/marketplace/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ReleaseController", func() {

	const timeout = time.Second * 5
	const interval = time.Millisecond * 10

	ctx := context.Background()

	var appName string
	var key types.NamespacedName
	var secret corev1.Secret

	BeforeEach(func() {
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "critical-stack"},
		}
		k8sClient.Create(ctx, &ns)

		appName = "exampleapp-v0.0.1-" + randString(6)

		key = types.NamespacedName{
			Name:      "sh.helm.release.v1." + appName + ".v1",
			Namespace: "critical-stack",
		}

		releaseSpec := fmt.Sprint(`{"name":"` + appName + `", "info": { "status": "deployed" }, "version":1}`)
		releaseEnc, err := encodeSecret(releaseSpec)
		Expect(err).Should(BeNil())
		secret = createSecretJSON(key, appName, releaseEnc)
		Expect(k8sClient.Create(ctx, &secret)).Should(Succeed())

	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, &secret)).Should(Succeed())
		time.Sleep(time.Second * 1)

	})

	Context("Release Object Autocreation", func() {
		It("Should create release object correctly", func() {
			release := &marketplacev1alpha2.Release{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: appName, Namespace: "critical-stack"}, release)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		Context("When the release object already exists", func() {
			var supersededSecret corev1.Secret

			BeforeEach(func() {
				supersededReleaseSpec := fmt.Sprint(`{"name":"` + appName + `", "info": { "status": "superseded" }, "version":1}`)
				supersededReleaseEnc, err := encodeSecret(supersededReleaseSpec)
				Expect(err).Should(BeNil())
				supersededSecret = createSecretJSON(key, appName, supersededReleaseEnc)
			})

			It("Should update release object correctly", func() {
				initialRelease := &marketplacev1alpha2.Release{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: appName, Namespace: "critical-stack"}, initialRelease)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				newKey := types.NamespacedName{
					Name:      "sh.helm.release.v1." + appName + ".v2",
					Namespace: "critical-stack",
				}

				By("Creating updated encoded test release spec")

				newReleaseSpec := fmt.Sprint(`{"name":"` + appName + `", "info": { "status": "deployed" }, "version":2}`)
				newReleaseEnc, err := encodeSecret(newReleaseSpec)
				Expect(err).Should(BeNil())
				newSecret := createSecretJSON(newKey, appName, newReleaseEnc)

				By("Creating updated test Secret from updated test release spec and updating status of old Secret")
				Expect(k8sClient.Update(ctx, &supersededSecret)).Should(Succeed())
				time.Sleep(interval)
				Expect(k8sClient.Create(ctx, &newSecret)).Should(Succeed())
				time.Sleep(interval)

				By("Fetching updated CRD created from updating Secret")

				updatedRelease := &marketplacev1alpha2.Release{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: appName, Namespace: "critical-stack"}, updatedRelease)
					return err == nil && updatedRelease != initialRelease
				}, timeout, interval).Should(BeTrue())

				By("Checking comparing updated release to initial release")
				Expect(updatedRelease).ShouldNot(Equal(initialRelease))
				Expect(updatedRelease.Name).Should(Equal(initialRelease.Name))
				Expect(updatedRelease.Spec.Version).Should(BeNumerically(">", initialRelease.Spec.Version))
			})
		})
	})
})

func encodeSecret(releaseSpec string) (string, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write([]byte(releaseSpec))
	if err != nil {
		return "", err
	}
	gz.Close()
	sEnc := base64.StdEncoding.EncodeToString(b.Bytes())
	return sEnc, nil
}

func createSecretJSON(key types.NamespacedName, appName string, releaseEnc string) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
			Labels: map[string]string{
				"name": appName,
			},
		},
		Data: map[string][]byte{
			"release": []byte(releaseEnc),
		},
		Type: "helm.sh/release.v1",
	}

}
