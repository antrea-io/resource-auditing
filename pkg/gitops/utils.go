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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func computePath(dir string, resource string, namespace string, file string) string {
	return filepath.Join(dir, resource, namespace, file)
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
