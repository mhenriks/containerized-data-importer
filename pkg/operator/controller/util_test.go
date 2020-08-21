/*
Copyright 2020 The CDI Authors.

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
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdiuploadv1 "kubevirt.io/containerized-data-importer/pkg/apis/upload/v1beta1"
)

var _ = Describe("mergeLabelsAndAnnotations", func() {
	It("Should properly merge labels and annotations, if no dest labels/anns", func() {
		source := createPod("source", map[string]string{"l1": "test"}, map[string]string{"a1": "ann"})
		dest := createPod("dest", nil, nil)
		mergeLabelsAndAnnotations(&source.ObjectMeta, &dest.ObjectMeta)
		Expect(dest.GetObjectMeta).ToNot(BeNil())
		Expect(dest.GetLabels()["l1"]).To(Equal("test"))
		Expect(dest.GetAnnotations()["a1"]).To(Equal("ann"))
	})

	It("Should properly merge labels and annotations, if no dest labels", func() {
		source := createPod("source", map[string]string{"l1": "test"}, map[string]string{"a1": "ann"})
		dest := createPod("dest", nil, map[string]string{"a1": "ann2"})
		mergeLabelsAndAnnotations(&source.ObjectMeta, &dest.ObjectMeta)
		Expect(dest.GetObjectMeta).ToNot(BeNil())
		Expect(dest.GetLabels()["l1"]).To(Equal("test"))
		// Check that dest is now equal to source
		Expect(dest.GetAnnotations()["a1"]).To(Equal("ann"))
	})

	It("Should properly merge labels and annotations, if no dest labels, and different ann", func() {
		source := createPod("source", map[string]string{"l1": "test"}, map[string]string{"a1": "ann"})
		dest := createPod("dest", nil, map[string]string{"a2": "ann2"})
		mergeLabelsAndAnnotations(&source.ObjectMeta, &dest.ObjectMeta)
		Expect(dest.GetObjectMeta).ToNot(BeNil())
		Expect(dest.GetLabels()["l1"]).To(Equal("test"))
		Expect(dest.GetAnnotations()["a1"]).To(Equal("ann"))
		Expect(dest.GetAnnotations()["a2"]).To(Equal("ann2"))
	})

	It("Should properly merge labels and annotations, if no dest ann", func() {
		source := createPod("source", map[string]string{"l1": "test"}, map[string]string{"a1": "ann"})
		dest := createPod("dest", map[string]string{"l1": "test2"}, nil)
		mergeLabelsAndAnnotations(&source.ObjectMeta, &dest.ObjectMeta)
		Expect(dest.GetObjectMeta).ToNot(BeNil())
		// Check that dest is now equal to source
		Expect(dest.GetLabels()["l1"]).To(Equal("test"))
		Expect(dest.GetAnnotations()["a1"]).To(Equal("ann"))
	})

	It("Should properly merge labels and annotations, if no dest ann, and different label", func() {
		source := createPod("source", map[string]string{"l1": "test"}, map[string]string{"a1": "ann"})
		dest := createPod("dest", map[string]string{"l2": "test2"}, nil)
		mergeLabelsAndAnnotations(&source.ObjectMeta, &dest.ObjectMeta)
		Expect(dest.GetObjectMeta).ToNot(BeNil())
		Expect(dest.GetLabels()["l1"]).To(Equal("test"))
		Expect(dest.GetLabels()["l2"]).To(Equal("test2"))
		Expect(dest.GetAnnotations()["a1"]).To(Equal("ann"))
	})

	It("will not merge CRD correctly", func() {
		obj1 := &extv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "obj",
			},
			Spec: extv1.CustomResourceDefinitionSpec{
				Group: "foo",
			},
		}

		err := setLastAppliedConfiguration(obj1)
		Expect(err).ToNot(HaveOccurred())

		obj2 := obj1.DeepCopy()
		obj2.Spec.PreserveUnknownFields = true

		obj3 := obj1.DeepCopy()
		// not necessary but let's be explicit
		obj3.Spec.PreserveUnknownFields = false
		err = setLastAppliedConfiguration(obj3)
		Expect(err).ToNot(HaveOccurred())

		obj4, err := mergeObject(obj3, obj2)
		Expect(err).ToNot(HaveOccurred())

		crd := obj4.(*extv1.CustomResourceDefinition)
		Expect(crd.Spec.PreserveUnknownFields).To(BeFalse())
	})

	It("will merge CRD correctly", func() {
		obj1 := &extv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "obj",
			},
			Spec: extv1.CustomResourceDefinitionSpec{
				Group:                 "foo",
				PreserveUnknownFields: true,
			},
		}

		err := setLastAppliedConfiguration(obj1)
		Expect(err).ToNot(HaveOccurred())

		obj2 := obj1.DeepCopy()

		obj3 := obj1.DeepCopy()
		// not necessary but let's be explicit
		obj3.Spec.PreserveUnknownFields = false
		err = setLastAppliedConfiguration(obj3)
		Expect(err).ToNot(HaveOccurred())

		obj4, err := mergeObject(obj3, obj2)
		Expect(err).ToNot(HaveOccurred())

		crd := obj4.(*extv1.CustomResourceDefinition)
		Expect(crd.Spec.PreserveUnknownFields).To(BeFalse())
	})
})

var _ = Describe("StripStatusFromObject", func() {
	It("Should not alter object without status", func() {
		in := &cdiuploadv1.UploadTokenRequestList{}
		out, err := stripStatusFromObject(in.DeepCopyObject())
		Expect(err).ToNot(HaveOccurred())
		Expect(reflect.DeepEqual(out, in)).To(BeTrue())
	})

	It("Should strip object status", func() {
		in := &cdiuploadv1.UploadTokenRequest{
			Status: cdiuploadv1.UploadTokenRequestStatus{
				Token: "thisisatoken",
			},
		}
		expected := &cdiuploadv1.UploadTokenRequest{
			Status: cdiuploadv1.UploadTokenRequestStatus{},
		}
		out, err := stripStatusFromObject(in.DeepCopyObject())
		Expect(err).ToNot(HaveOccurred())
		Expect(reflect.DeepEqual(out, in)).To(BeFalse())
		Expect(reflect.DeepEqual(out, expected)).To(BeTrue())
	})

})

func createPod(name string, labels, annotations map[string]string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if len(labels) > 0 {
		pod.ObjectMeta.Labels = labels
	}
	if len(annotations) > 0 {
		pod.ObjectMeta.Annotations = annotations
	}
	return pod
}
