/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	// "k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ksmv1 "github.com/jtyr/crsm-operator/api/v1"
)

var _ = Describe("CustomResourceStateMetrics Controller", func() {
	Context("when reconciling a resource", func() {
		const resourceName = "test-resource"
		const resourceNamespace = "default"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		customresourcestatemetrics := &ksmv1.CustomResourceStateMetrics{}

		BeforeEach(func() {
			By("Creating the custom resource for the Kind CustomResourceStateMetrics")
			err := k8sClient.Get(ctx, typeNamespacedName, customresourcestatemetrics)
			if err != nil && errors.IsNotFound(err) {
				resource := &ksmv1.CustomResourceStateMetrics{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: ksmv1.CustomResourceStateMetricsSpec{
						ConfigMap: ksmv1.CustomResourceStateMetricsConfigMap{
							Name: "kube-state-metrics-customresourcestate-config",
						},
						Resources: []runtime.RawExtension{
							{
								Raw: []byte(`{"foo": "bar"}`),
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// Cleanup logic after each test, like removing the resource instance.
			resource := &ksmv1.CustomResourceStateMetrics{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CustomResourceStateMetrics")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CustomResourceStateMetricsReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
				// Queue:  workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[types.NamespacedName]()),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			verifyResourceIsReady := func(g Gomega) {
				crsm := &ksmv1.CustomResourceStateMetrics{}
				err := k8sClient.Get(ctx, typeNamespacedName, crsm)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(crsm.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue), "Resource is not yet ready")
			}
			Eventually(verifyResourceIsReady).Should(Succeed())
		})
	})
})

func TestFindBlock(t *testing.T) {
	g := NewWithT(t)

	tests := map[string]struct {
		name     string
		expected string
	}{
		"foo-found": {
			name:     "foo",
			expected: "true;1;3",
		},
		"bar-found": {
			name:     "bar",
			expected: "true;4;6",
		},
		"baz-found": {
			name:     "baz",
			expected: "true;7;9",
		},
		"asd-not-found": {
			name:     "asd",
			expected: "false;-1;-1",
		},
	}

	r := CustomResourceStateMetricsReconciler{}

	lines := []string{
		"aaa: bbb",
		fmt.Sprintf(beginMarkerFormat, "foo"),
		"foo: bar",
		fmt.Sprintf(endMarkerFormat, "foo"),
		fmt.Sprintf(beginMarkerFormat, "bar"),
		"bar: baz",
		fmt.Sprintf(endMarkerFormat, "bar"),
		fmt.Sprintf(beginMarkerFormat, "baz"),
		"baz: foo",
		fmt.Sprintf(endMarkerFormat, "baz"),
	}

	for name, test := range tests {
		found, begin, end := r.findBlock(test.name, lines)
		result := fmt.Sprintf("%t;%d;%d", found, begin, end)

		g.Expect(result).To(Equal(test.expected), "Test [%s]:", name)
	}
}

func TestJoinLines(t *testing.T) {
	g := NewWithT(t)

	tests := map[string]struct {
		begin    int
		end      int
		expected string
	}{
		"beginning": {
			begin:    0,
			end:      3,
			expected: "0\n1\n2\n3\n",
		},
		"middle": {
			begin:    4,
			end:      7,
			expected: "4\n5\n6\n7\n",
		},
		"end": {
			begin:    7,
			end:      9,
			expected: "7\n8\n9",
		},
		"unknown-end": {
			begin:    7,
			end:      -1,
			expected: "7\n8\n9",
		},
		"out-of-bound-begin": {
			begin:    -100,
			end:      3,
			expected: "0\n1\n2\n3\n",
		},
		"out-of-bound-end": {
			begin:    7,
			end:      1000,
			expected: "7\n8\n9",
		},
	}

	r := CustomResourceStateMetricsReconciler{}

	lines := []string{
		"0",
		"1",
		"2",
		"3",
		"4",
		"5",
		"6",
		"7",
		"8",
		"9",
	}

	for name, test := range tests {
		result := r.joinLines(lines, test.begin, test.end)

		g.Expect(result).To(Equal(test.expected), "Test [%s]:", name)
	}
}
