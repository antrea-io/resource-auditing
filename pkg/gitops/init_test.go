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
	"testing"

	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type test_resource struct {
	inputResource runtime.Object
	expPath       string
	expYaml       string
}

var (
	dir         = ""
	selectorA   = metav1.LabelSelector{MatchLabels: map[string]string{"foo1": "bar1"}}
	selectorB   = metav1.LabelSelector{MatchLabels: map[string]string{"foo2": "bar2"}}
	selectorC   = metav1.LabelSelector{MatchLabels: map[string]string{"foo3": "bar3"}}
	p10         = float64(10)
	int80       = intstr.FromInt(80)
	int81       = intstr.FromInt(81)
	allowAction = crdv1alpha1.RuleActionAllow
	Np1         = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
			TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
			ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
				Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
			},
		},
		expPath: "/k8s-policies/nsA/npA.yaml",
		expYaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: npA
  namespace: nsA
spec:
  ingress:
  - {}
  podSelector: {}
  policyTypes:
  - Ingress
`,
	}
	Np2 = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
			TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
			ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npB", UID: "uidB"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
				Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
			},
		},
		expPath: "/k8s-policies/nsA/npB.yaml",
		expYaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: npB
  namespace: nsA
spec:
  egress:
  - {}
  podSelector: {}
  policyTypes:
  - Egress
`,
	}
	Np3 = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
			TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
			ObjectMeta: metav1.ObjectMeta{Namespace: "nsB", Name: "npC", UID: "uidC"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: selectorA,
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Port: &int80,
							},
						},
						From: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector:       &selectorB,
								NamespaceSelector: &selectorC,
							},
						},
					},
				},
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Port: &int81,
							},
						},
						To: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector:       &selectorB,
								NamespaceSelector: &selectorC,
							},
						},
					},
				},
			},
		},
		expPath: "/k8s-policies/nsB/npC.yaml",
		expYaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: npC
  namespace: nsB
spec:
  egress:
  - ports:
    - port: 81
    to:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
    ports:
    - port: 80
  podSelector:
    matchLabels:
      foo1: bar1
`,
	}
	Anp1 = test_resource{
		inputResource: &crdv1alpha1.NetworkPolicy{
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
		},
		expPath: "/antrea-policies/nsA/anpA.yaml",
		expYaml: `apiVersion: crd.antrea.io/v1alpha1
kind: NetworkPolicy
metadata:
  name: anpA
  namespace: nsA
spec:
  appliedTo:
  - podSelector:
      matchLabels:
        foo1: bar1
  egress:
  - action: Allow
    enableLogging: false
    from: null
    name: ""
    ports:
    - port: 81
    to:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
  ingress:
  - action: Allow
    enableLogging: false
    from:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
    name: ""
    ports:
    - port: 80
    to: null
  priority: 10
`,
	}
	Acnp1 = test_resource{
		inputResource: &crdv1alpha1.ClusterNetworkPolicy{
			TypeMeta:   metav1.TypeMeta{Kind: "ClusterNetworkPolicy", APIVersion: "crd.antrea.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{Name: "cnpA", UID: "uidD"},
			Spec: crdv1alpha1.ClusterNetworkPolicySpec{
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
		},
		expPath: "/antrea-cluster-policies/cnpA.yaml",
		expYaml: `apiVersion: crd.antrea.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  name: cnpA
spec:
  appliedTo:
  - podSelector:
      matchLabels:
        foo1: bar1
  egress:
  - action: Allow
    enableLogging: false
    from: null
    name: ""
    ports:
    - port: 81
    to:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
  ingress:
  - action: Allow
    enableLogging: false
    from:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
    name: ""
    ports:
    - port: 80
    to: null
  priority: 10
`,
	}
	Tier1 = test_resource{
		inputResource: &crdv1alpha1.Tier{
			TypeMeta: metav1.TypeMeta{Kind: "Tier", APIVersion: "crd.antrea.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "TierA",
			},
			Spec: crdv1alpha1.TierSpec{
				Priority:    10,
				Description: "This is a test tier",
			},
		},
		expPath: "/antrea-tiers/TierA.yaml",
		expYaml: `apiVersion: crd.antrea.io/v1alpha1
kind: Tier
metadata:
  name: TierA
spec:
  description: This is a test tier
  priority: 10
`,
	}
)

func TestSetupRepo(t *testing.T) {
	tests := []struct {
		name           string
		inputResources []test_resource
	}{
		{
			name:           "empty-test",
			inputResources: []test_resource{},
		},
		{
			name:           "basic-test",
			inputResources: []test_resource{Np1, Np2, Np3, Anp1, Acnp1},
		},
		{
			name:           "empty-K8s-test",
			inputResources: []test_resource{Anp1, Acnp1},
		},
		{
			name:           "empty-CRDs-test",
			inputResources: []test_resource{Np1, Np2},
		},
		{
			name:           "tiers-test",
			inputResources: []test_resource{Np1, Np2, Anp1, Tier1},
		},
	}
	for _, test := range tests {
		var expectedPaths = []string{}
		var expectedYamls = []string{}
		var resources = []runtime.Object{}
		for _, resource := range test.inputResources {
			resources = append(resources, resource.inputResource)
			expectedPaths = append(expectedPaths, resource.expPath)
			expectedYamls = append(expectedYamls, resource.expYaml)
		}
		fakeClient := NewClient(resources...)
		k8s := &K8sClient{
			Client: fakeClient,
		}
		runSetupTest(t, k8s, expectedPaths, expectedYamls)
	}
}

func runSetupTest(t *testing.T, k8s *K8sClient, expPaths []string, expYamls []string) {
	cr, err := SetupRepo(k8s, StorageModeInMemory, dir)
	assert.NoError(t, err, "unable to set up repo")
	for i, path := range expPaths {
		file, err := cr.Fs.Open(path)
		assert.NoError(t, err, "unable to open file")
		fstat, _ := cr.Fs.Stat(path)
		var buffer = make([]byte, fstat.Size())
		file.Read(buffer)
		assert.Equal(t, string(buffer), expYamls[i], "read file does not match expected YAML")
	}
}

func TestRepoDuplicate(t *testing.T) {
	fakeClient := NewClient(Np1.inputResource, Anp1.inputResource)
	k8s := &K8sClient{
		Client: fakeClient,
	}
	_, err := SetupRepo(k8s, StorageModeInMemory, dir)
	assert.NoError(t, err, "unable to set up repo (1st time)")
	_, err = SetupRepo(k8s, StorageModeInMemory, dir)
	assert.NoError(t, err, "unable to handle duplicate repo initialization")
}

func NewClient(objects ...runtime.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	RegisterTypes(scheme)
	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithRuntimeObjects(objects...)
	clientBuilder.WithScheme(scheme)
	client := clientBuilder.Build()
	return client
}
