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
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

var dirMap = map[string]string{
	"networkpoliciesnetworking.k8s.io":    "k8s-policies",
	"networkpoliciescrd.antrea.io":        "antrea-policies",
	"clusternetworkpoliciescrd.antrea.io": "antrea-cluster-policies",
	"tierscrd.antrea.io":                  "antrea-tiers",
}

var resourceMap = map[string]string{
	"networkpoliciesnetworking.k8s.io":    "K8s network policy ",
	"networkpoliciescrd.antrea.io":        "Antrea network policy ",
	"clusternetworkpoliciescrd.antrea.io": "Antrea cluster network policy ",
	"tierscrd.antrea.io":                  "Antrea tier ",
}

func computePath(dir string, resource string, namespace string, file string) string {
	return filepath.Join(dir, resource, namespace, file)
}

func getAbsRepoPath(dir string, event auditv1.Event) string {
	resource := dirMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]
	namespace := event.ObjectRef.Namespace
	return computePath(dir, resource, namespace, "")
}

func getRelRepoPath(event auditv1.Event) string {
	resource := dirMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]
	namespace := event.ObjectRef.Namespace
	path := computePath("", resource, namespace, "")
	return path
}

func getFileName(event auditv1.Event) string {
	return "/" + event.ObjectRef.Name + ".yaml"
}

func clearFields(resource *unstructured.Unstructured) {
	resource.SetUID("")
	resource.SetGeneration(0)
	resource.SetManagedFields(nil)
	resource.SetCreationTimestamp(metav1.Time{})
	resource.SetResourceVersion("")
	annotations := resource.GetAnnotations()
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
	resource.SetAnnotations(annotations)
	delete(resource.Object, "status")
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
