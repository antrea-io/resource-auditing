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
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"antrea.io/antrea/pkg/apis/crd/v1alpha1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClient struct {
	client.Client
}

func NewKubernetes() (*K8sClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig, hasIt := os.LookupEnv("KUBECONFIG")
		if !hasIt {
			kubeconfig = clientcmd.RecommendedHomeFile
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("unable to build config from flags, check KUBECONFIG file: %w", err)
		}
	}
	scheme := runtime.NewScheme()
	RegisterTypes(scheme)
	client, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate new generic client: %w", err)
	}
	return &K8sClient{client}, nil
}

func RegisterTypes(scheme *runtime.Scheme) {
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy"},
		&networking.NetworkPolicy{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicyList"},
		&networking.NetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "ListOptions"},
		&metav1.ListOptions{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy"},
		&v1alpha1.NetworkPolicy{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicyList"},
		&v1alpha1.NetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ClusterNetworkPolicy"},
		&v1alpha1.ClusterNetworkPolicy{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ClusterNetworkPolicyList"},
		&v1alpha1.ClusterNetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "Tier"},
		&v1alpha1.Tier{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "TierList"},
		&v1alpha1.TierList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ListOptions"},
		&metav1.ListOptions{})
}

func (k *K8sClient) GetResource(resource *unstructured.Unstructured, namespace string, name string) (*unstructured.Unstructured, error) {
	err := k.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, resource)
	if err != nil {
		return nil, fmt.Errorf("unable to get resource namespace: %s, name: %s: %w", namespace, name, err)
	}
	return resource, nil
}

func (k *K8sClient) ListResource(resourceList *unstructured.UnstructuredList) (*unstructured.UnstructuredList, error) {
	err := k.List(context.TODO(), resourceList)
	if err != nil {
		return nil, fmt.Errorf("unable to list resource: APIVersion: %s, Kind: %s: %w", resourceList.GetAPIVersion(), resourceList.GetKind(), err)
	}
	return resourceList, nil
}

func (k *K8sClient) CreateOrUpdateResource(resource *unstructured.Unstructured) error {
	if err := k.Create(context.TODO(), resource); err == nil {
		klog.V(2).InfoS("created resource", "resourceName", resource.GetName())
		return nil
	} else if errors.IsAlreadyExists(err) {
		klog.V(2).InfoS("resource already exists, trying update instead", "resourceName", resource.GetName())
		oldResource := &unstructured.Unstructured{}
		oldResource.SetGroupVersionKind(resource.GroupVersionKind())
		_ = k.Get(context.TODO(), client.ObjectKey{
			Namespace: resource.GetNamespace(),
			Name:      resource.GetName(),
		}, oldResource)
		resource.SetResourceVersion(oldResource.GetResourceVersion())
		if err := k.Update(context.TODO(), resource); err != nil {
			return fmt.Errorf("unable to update resource: resourceName: %s: %w", resource.GetName(), err)
		}
		klog.V(2).InfoS("updated resource", "resourceName", resource.GetName())
		return nil
	} else {
		return fmt.Errorf("error while creating resource: resourceName: %s: %w", resource.GetName(), err)
	}
}

func (k *K8sClient) DeleteResource(resource *unstructured.Unstructured) error {
	err := k.Delete(context.TODO(), resource)
	if err != nil {
		return fmt.Errorf("unable to delete resource %s: %w", resource.GetName(), err)
	}
	klog.V(2).InfoS("deleted k8s network policy", "resourceName", resource.GetName())
	return nil
}
