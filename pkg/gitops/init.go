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
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"k8s.io/klog/v2"

	billy "github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"
	memory "github.com/go-git/go-git/v5/storage/memory"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StorageModeType string

const (
	StorageModeDisk     StorageModeType = "Disk"
	StorageModeInMemory StorageModeType = "InMemory"
)

var gvkDirMap = map[schema.GroupVersionKind]string{
	{Group: "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicyList",
	}: "k8s-policies",
	{Group: "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicyList",
	}: "antrea-policies",
	{Group: "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ClusterNetworkPolicyList",
	}: "antrea-cluster-policies",
	{Group: "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "TierList",
	}: "antrea-tiers",
}

func getAllResourceListTypes() []schema.GroupVersionKind {
	return []schema.GroupVersionKind{
		{
			Group:   "networking.k8s.io",
			Version: "v1",
			Kind:    "NetworkPolicyList",
		},
		{
			Group:   "crd.antrea.io",
			Version: "v1alpha1",
			Kind:    "NetworkPolicyList",
		},
		{
			Group:   "crd.antrea.io",
			Version: "v1alpha1",
			Kind:    "ClusterNetworkPolicyList",
		},
		{
			Group:   "crd.antrea.io",
			Version: "v1alpha1",
			Kind:    "TierList",
		},
	}
}

type CustomRepo struct {
	Repo           *git.Repository
	K8s            *K8sClient
	RollbackMode   bool
	StorageMode    StorageModeType
	ServiceAccount string
	Dir            string
	Fs             billy.Filesystem
	Mutex          sync.Mutex
}

func SetupRepo(k8s *K8sClient, mode StorageModeType, dir string) (*CustomRepo, error) {
	if mode != StorageModeDisk && mode != StorageModeInMemory {
		return nil, fmt.Errorf("mode must be memory(mem) or disk(disk), '%s' is not valid", mode)
	}
	storer := memory.NewStorage()
	fs := memfs.New()
	svcAcct := "system:serviceaccount:" + GetAuditPodNamespace() + ":" + GetAuditServiceAccount()
	cr := CustomRepo{
		K8s:            k8s,
		RollbackMode:   false,
		StorageMode:    mode,
		ServiceAccount: svcAcct,
		Dir:            dir,
		Fs:             fs,
	}
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	r, err := cr.createRepo(storer)
	cr.Repo = r
	if err == git.ErrRepositoryAlreadyExists {
		klog.V(2).InfoS("resource repository already exists - skipping initialization")
		return &cr, nil
	} else if err != nil {
		klog.ErrorS(err, "unable to create resource repository")
		return nil, err
	}
	if err := cr.addAllResources(); err != nil {
		klog.ErrorS(err, "unable to add resource yamls to repository")
		return nil, err
	}
	if err := cr.AddAndCommit("audit-init", "system@audit.antrea.io", "Initial commit of existing policies"); err != nil {
		klog.ErrorS(err, "unable to add and commit existing resources to repository")
		return nil, err
	}
	klog.V(2).Infof("Repository successfully initialized at %s", cr.Dir)
	return &cr, nil
}

func (cr *CustomRepo) createRepo(storer *memory.Storage) (*git.Repository, error) {
	if cr.StorageMode == StorageModeInMemory {
		r, err := git.Init(storer, cr.Fs)
		if err == git.ErrRepositoryAlreadyExists {
			klog.V(2).InfoS("resource repository already exists - skipping initialization")
			return nil, err
		} else if err != nil {
			klog.ErrorS(err, "unable to initialize git repo")
			return nil, err
		}
		return r, nil
	}
	if cr.Dir == "" {
		path, err := os.Getwd()
		if err != nil {
			klog.ErrorS(err, "unable to retrieve the current working directory")
			return nil, err
		}
		if path != "/" {
			cr.Dir = path
		}
	}
	cr.Dir += "/resource-auditing-repo"
	r, err := git.PlainInit(cr.Dir, false)
	if err == git.ErrRepositoryAlreadyExists {
		klog.V(2).InfoS("resource repository already exists - skipping initialization")
		r, err := git.PlainOpen(cr.Dir)
		if err != nil {
			klog.ErrorS(err, "unable to retrieve existing repository")
			return nil, err
		}
		return r, git.ErrRepositoryAlreadyExists
	} else if err != nil {
		klog.ErrorS(err, "unable to initialize git repo")
		return nil, err
	}
	return r, nil
}

func (cr *CustomRepo) addAllResources() error {
	for _, resourceListType := range getAllResourceListTypes() {
		if err := cr.createResourceDir(resourceListType); err != nil {
			klog.ErrorS(err, "unable to create resource directory",
				"gvk", resourceListType.String())
			return err
		}
		if err := cr.addResource(resourceListType); err != nil {
			klog.ErrorS(err, "unable to add resource",
				"gvk", resourceListType.String())
			return err
		}
	}
	return nil
}

func (cr *CustomRepo) addResource(resourceList schema.GroupVersionKind) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(resourceList)
	resources, err := cr.K8s.ListResource(list)
	if err != nil {
		klog.ErrorS(err, "could not list resource",
			"APIVersion", list.GetAPIVersion(),
			"Kind", list.GetKind())
		return err
	}
	var namespaces []string
	for i, np := range resources.Items {
		clearFields(&resources.Items[i])
		name := np.GetName()
		namespace := np.GetNamespace()
		if !stringInSlice(namespace, namespaces) {
			namespaces = append(namespaces, namespace)
			if cr.StorageMode == StorageModeDisk {
				namespaceDir := computePath(cr.Dir, gvkDirMap[resourceList], namespace, "")
				os.Mkdir(namespaceDir, 0700)
			} else {
				namespaceDir := computePath("", gvkDirMap[resourceList], namespace, "")
				cr.Fs.MkdirAll(namespaceDir, 0700)
			}
		}
		path := computePath(cr.Dir, gvkDirMap[resourceList], namespace, name+".yaml")
		y, err := yaml.Marshal(&resources.Items[i])
		if err != nil {
			klog.ErrorS(err, "unable to marshal resource config")
			return err
		}
		if err := cr.writeFileToPath(path, y); err != nil {
			klog.ErrorS(err, "unable to write yaml to path", "path", path)
			return err
		}
		klog.V(2).InfoS("Added resource", "path", path)
	}
	return nil
}

func (cr *CustomRepo) createResourceDir(resourceList schema.GroupVersionKind) error {
	var err error
	if cr.StorageMode == StorageModeDisk {
		resourceDir := computePath(cr.Dir, gvkDirMap[resourceList], "", "")
		err = os.Mkdir(resourceDir, 0700)
	} else {
		resourceDir := computePath("", gvkDirMap[resourceList], "", "")
		err = cr.Fs.MkdirAll(resourceDir, 0700)
	}
	if err != nil {
		klog.ErrorS(err, "unable to create resource directory")
		return err
	}
	return nil
}

func (cr *CustomRepo) writeFileToPath(path string, yaml []byte) error {
	if cr.StorageMode == StorageModeDisk {
		err := ioutil.WriteFile(path, yaml, 0600)
		if err != nil {
			klog.ErrorS(err, "unable to write resource config to file")
			return err
		}
	} else {
		newFile, err := cr.Fs.Create(path)
		if err != nil {
			klog.ErrorS(err, "unable to write resource config to file")
			return err
		}
		newFile.Write(yaml)
		newFile.Close()
	}
	return nil
}
