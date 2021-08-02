// Copyright 2021 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitops

import (
	"io/ioutil"
	"testing"
	"time"

	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	np1 = &networkingv1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
		},
	}
	np2 = &networkingv1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npB", UID: "uidB"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
	anp1 = &crdv1alpha1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "crd.antrea.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "anpA", UID: "uidC"},
		Spec: crdv1alpha1.NetworkPolicySpec{
			AppliedTo: []crdv1alpha1.NetworkPolicyPeer{
				{PodSelector: &selectorA},
			},
			Priority: p10,
			Ingress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int80,
						},
					},
					From: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
			Egress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int81,
						},
					},
					To: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
		},
	}
)

func TestRollback(t *testing.T) {
	fakeClient := NewClient(np1.DeepCopy(), anp1.DeepCopy())
	k8s := &K8sClient{
		Client: fakeClient,
	}
	cr, err := SetupRepo(k8s, StorageModeInMemory, dir)
	assert.NoError(t, err, "unable to set up repo")
	h, err := cr.Repo.Head()
	assert.NoError(t, err, "unable to get repo head ref")
	testSig := &object.Signature{
		Name:  "test",
		Email: "test@antrea.audit.io",
		When:  time.Now(),
	}
	err = cr.TagCommit(h.Hash().String(), "test-tag", testSig)
	assert.NoError(t, err, "unable to create new tag")

	// Create, update, and delete a resource
	r := unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	r.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(np2)
	assert.NoError(t, err, "unable to convert typed to unstructured object")
	err = k8s.CreateOrUpdateResource(&r)
	assert.NoError(t, err, "unable to create new resource")

	updatedNP := np1
	updatedNP.ObjectMeta.SetClusterName("new-cluster-name")
	r = unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	r.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(updatedNP)
	assert.NoError(t, err, "unable to convert typed to unstructured object")
	err = k8s.CreateOrUpdateResource(&r)
	assert.NoError(t, err, "unable to update new resource")

	r = unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy",
	})
	r.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(anp1)
	assert.NoError(t, err, "unable to convert typed to unstructured object")
	err = k8s.DeleteResource(&r)
	assert.NoError(t, err, "unable to delete resource")

	jsonStr, err := ioutil.ReadFile("../../test/files/rollback-log.txt")
	assert.NoError(t, err, "could not read rollback-log file")
	err = cr.HandleEventList(jsonStr)
	assert.NoError(t, err, "could not process audit events from file")

	// Attempt rollback
	commit, err := cr.TagToCommit("test-tag")
	assert.NoError(t, err, "could not retrieve commit from tag")
	err = cr.RollbackRepo(commit)
	assert.NoError(t, err, "rollback failed")

	// Check latest commit
	newH, err := cr.Repo.Head()
	assert.NoError(t, err, "unable to get repo head ref")
	rollbackCommit, err := cr.Repo.CommitObject(newH.Hash())
	assert.NoError(t, err, "unable to get rollback commit object")
	assert.Equal(t, "Rollback to commit "+h.Hash().String(), rollbackCommit.Message,
		"rollback commit not found, head commit message mismatch")

	// Check cluster state
	res := &unstructured.Unstructured{}
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	np, err := k8s.GetResource(res, "nsA", "npA")
	assert.NoError(t, err, "unable to get policy after rollback")
	assert.Equal(t, "", np.GetClusterName(),
		"Error (TestRollback): updated field should be empty after rollback")

	res = &unstructured.Unstructured{}
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy",
	})
	_, err = k8s.GetResource(res, "nsA", "anpA")
	assert.NoError(t, err, "unable to get antrea policy after rollback")
}
