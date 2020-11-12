package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	marketplacev1alpha2 "github.com/criticalstack/marketplace/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("SourceController", func() {

	const timeout = time.Second * 5
	const interval = time.Millisecond * 10

	ctx := context.Background()
	srcAddr := fmt.Sprintf("localhost:%d", 8089)

	var sourceServer serverWithCancel
	var cm corev1.ConfigMap
	var src marketplacev1alpha2.Source

	BeforeEach(func() {
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "critical-stack"},
		}
		k8sClient.Create(ctx, &ns)

		By("Creating basic test source object")
		src = marketplacev1alpha2.Source{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Source",
				APIVersion: "marketplace.criticalstack.com/v1alpha2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: randString(16),
			},
			Spec: marketplacev1alpha2.SourceSpec{},
		}
		src.Spec.URL = "http://" + srcAddr

		By("Creating basic test configMap object")
		cm = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "marketplace-app-categories",
				Namespace: "critical-stack",
			},
			Data: map[string]string{},
		}

	})

	JustBeforeEach(func() {
		// Create local server to host Source
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir("testdata/marketplace-source/")))
		sourceServer = newServerWithCancel(mux, srcAddr)
		go sourceServer.Run()
	})

	AfterEach(func() {
		// Tear down local source server
		err := sourceServer.Cancel(3 * time.Second)
		Expect(err).Should(BeNil())

		// Cleanup k8s resources
		Expect(k8sClient.Delete(ctx, &cm)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, &src)).Should(Succeed())
	})

	Context("When Source Status != Updating", func() {
		It("Should requeue source object", func() {

			// Create Source object
			Expect(k8sClient.Create(ctx, &src)).Should(Succeed())
			Expect(src.Status.State).Should(BeZero())

			// Create ConfigMap object
			Expect(k8sClient.Create(ctx, &cm)).Should(Succeed())

			fetchedSrc := &marketplacev1alpha2.Source{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: src.Name, Namespace: ""}, fetchedSrc)
				return err == nil && fetchedSrc.Status.State != ""
			}, timeout, interval).Should(BeTrue())
			Expect(fetchedSrc.Status.State).Should(Equal(marketplacev1alpha2.SyncStateUpdating))
			Expect(fetchedSrc.Status.Reason).Should(Equal("object changed"))
		})
	})
	Context("When Source Status == Updating", func() {

		src.Status.State = marketplacev1alpha2.SyncStateUpdating

		Context("When the Source UpdateFrequency is invalid", func() {

			It("Should reconcile with error state", func() {

				src.Spec.UpdateFrequency = "invalid"

				// Create Source object
				Expect(k8sClient.Create(ctx, &src)).Should(Succeed())

				// Create ConfigMap object
				Expect(k8sClient.Create(ctx, &cm)).Should(Succeed())

				//time.Sleep(interval)
				fetchedSrc := &marketplacev1alpha2.Source{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: src.Name, Namespace: ""}, fetchedSrc)
					return err == nil && fetchedSrc.Status.State != marketplacev1alpha2.SyncStateUpdating && fetchedSrc.Status.State != ""
				}, timeout, interval).Should(BeTrue())
				Expect(fetchedSrc.Status.State).Should(Equal(marketplacev1alpha2.SyncStateError))
				Expect(fetchedSrc.Status.Reason).Should(ContainSubstring("spec.updateFrequency is invalid:"))
			})
		})

		Context("When the Source UpdateFrequency is empty", func() {
			src.Spec.UpdateFrequency = ""

			Context("When there are no existing applications", func() {

				It("Should successfully create the application objects", func() {
					// Create Source object
					Expect(k8sClient.Create(ctx, &src)).Should(Succeed())

					// Create ConfigMap object
					Expect(k8sClient.Create(ctx, &cm)).Should(Succeed())

					fetchedSrc := &marketplacev1alpha2.Source{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, types.NamespacedName{Name: src.Name, Namespace: ""}, fetchedSrc)
						return err == nil && fetchedSrc.Status.State == marketplacev1alpha2.SyncStateSuccess
					}, timeout, interval).Should(BeTrue())

					appList := &marketplacev1alpha2.ApplicationList{}
					Eventually(func() bool {
						err := k8sClient.List(ctx, appList)
						return err == nil && len(appList.Items) > 0
					}, timeout, interval).Should(BeTrue())

					srcAppNum := 0
					for _, v := range appList.Items {
						if v.Labels["marketplace.criticalstack.com/source.name"] == src.Name {
							srcAppNum += 1
						}
					}
					Expect(srcAppNum).Should(Equal(2))
				})
			})

			Context("When there are existing applications", func() {
				It("Should successfully update the existing application objects", func() {
					// Create Source object
					Expect(k8sClient.Create(ctx, &src)).Should(Succeed())

					// Create ConfigMap object
					Expect(k8sClient.Create(ctx, &cm)).Should(Succeed())

					fetchedSrc := &marketplacev1alpha2.Source{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, types.NamespacedName{Name: src.Name, Namespace: ""}, fetchedSrc)
						return err == nil && fetchedSrc.Status.State == marketplacev1alpha2.SyncStateSuccess
					}, timeout, interval).Should(BeTrue())

					By("Updating Source")
					// Update Source repo with updated app
					srcAddr = fmt.Sprintf("localhost:%d", 8088)
					fetchedSrc.Spec.URL = "http://" + srcAddr

					// Create local server to host updated Source
					mux := http.NewServeMux()
					mux.Handle("/", http.FileServer(http.Dir("testdata/marketplace-updated-source/")))
					updatedSourceServer := newServerWithCancel(mux, srcAddr)
					go updatedSourceServer.Run()

					Expect(k8sClient.Update(ctx, fetchedSrc)).Should(Succeed())
					time.Sleep(time.Second * 1)

					appList := &marketplacev1alpha2.ApplicationList{}
					Eventually(func() bool {
						err := k8sClient.List(ctx, appList)
						return err == nil && len(appList.Items) > 0
					}, timeout, interval).Should(BeTrue())

					updatedSrcAppNum := 0

					for _, app := range appList.Items {
						if app.Labels["marketplace.criticalstack.com/source.name"] == src.Name {
							for _, v := range app.Versions {
								if v.Version != "1.0.0" {
									updatedSrcAppNum += 1
									break
								}
							}
						}
					}

					Expect(updatedSrcAppNum).Should(Equal(2))

					// Cleanup - Tear down local source server
					err := updatedSourceServer.Cancel(3 * time.Second)
					Expect(err).Should(BeNil())

				})
			})

		})
	})
})

type serverWithCancel struct {
	server *http.Server
	done   chan (error)
}

func newServerWithCancel(h http.Handler, addr string) serverWithCancel {
	return serverWithCancel{
		done: make(chan error),
		server: &http.Server{
			Addr:    addr,
			Handler: h,
		},
	}
}

func (s serverWithCancel) Run() {
	defer close(s.done)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.done <- err
	}
}

func (s serverWithCancel) Cancel(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-s.done:
		return err
	}
}
