package gitops

import (
	"fmt"
	"io/ioutil"
	"testing"

	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	directory   = ""
	selectorA   = metav1.LabelSelector{MatchLabels: map[string]string{"foo1": "bar1"}}
	selectorB   = metav1.LabelSelector{MatchLabels: map[string]string{"foo2": "bar2"}}
	selectorC   = metav1.LabelSelector{MatchLabels: map[string]string{"foo3": "bar3"}}
	p10         = float64(10)
	int80       = intstr.FromInt(80)
	int81       = intstr.FromInt(81)
	allowAction = crdv1alpha1.RuleActionAllow
	np1         = &networkingv1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
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

func TestHandleEventList(t *testing.T) {
	fakeClient := NewClient(np1, anp1)
	k8s := &K8sClient{
		Client: fakeClient,
	}

	jsonstring, err := ioutil.ReadFile("../../test/files/correct-audit-log.txt")
	if err != nil {
		t.Errorf("Error (TestHandleEventList): could not read correct-audit-log file")
	}
	cr, err := SetupRepo(k8s, StorageModeInMemory, directory)
	if err != nil {
		t.Errorf("Error (TestHandleEventList): could not set up repo")
	}

	err = cr.HandleEventList(jsonstring)
	if err != nil {
		t.Errorf("Error (TestHandleEventList): could not handle correct audit event list")
	}

	cr.RollbackMode = true
	if cr.HandleEventList(jsonstring) == nil {
		t.Errorf("Error (TestHandleEventList): could not return correct error with roll back")
	}
	cr.RollbackMode = false

	for i := 1; i < 4; i++ {
		filename := fmt.Sprintf("%s%d%s", "../../test/files/incorrect-audit-log-", i, ".txt")
		jsonstring, err := ioutil.ReadFile(filename)
		if err != nil {
			cr.HandleEventList(jsonstring)
			if err == nil {
				t.Errorf("Error (TestHandleEventList): could not return error on erroneous json file")
			}
		}
	}
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
