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
	"os"
	"path/filepath"
	"sync"

	"github.com/ghodss/yaml"
	billy "github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	memory "github.com/go-git/go-git/v5/storage/memory"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
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

func GetResources() []string {
	var resources []string
	for _, resource := range gvkDirMap {
		resources = append(resources, resource)
	}
	return resources
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
	ServiceAccount string
	Fs             billy.Filesystem
	Mutex          sync.Mutex
}

func SetupRepo(k8s *K8sClient, mode StorageModeType, dir string) (*CustomRepo, error) {
	storer, fs, err := setupStorage(dir, mode)
	if err != nil {
		return nil, fmt.Errorf("unable to set up filesystem/storer backend for repo")
	}
	svcAcct := "system:serviceaccount:" + GetAuditPodNamespace() + ":" + GetAuditServiceAccount()
	cr := CustomRepo{
		K8s:            k8s,
		RollbackMode:   false,
		ServiceAccount: svcAcct,
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
		return nil, fmt.Errorf("unable to create resource repository: %w", err)
	}
	if err := cr.addAllResources(); err != nil {
		return nil, fmt.Errorf("unable to add resource yamls to repository: %w", err)
	}
	if err := cr.AddAndCommit("audit-init", "system@audit.antrea.io", "Initial commit of existing policies"); err != nil {
		return nil, fmt.Errorf("unable to add/commit existing reosurces to repository: %w", err)
	}
	klog.V(2).Infof("repository successfully initialized at %s", dir)
	return &cr, nil
}

func setupStorage(dir string, mode StorageModeType) (storage.Storer, billy.Filesystem, error) {
	var storer storage.Storer
	var worktreeFs, storerFs billy.Filesystem
	if mode == StorageModeDisk {
		if dir == "" {
			dir, _ = os.Getwd()
		}
		dir = filepath.Join(dir, "resource-auditing-repo")
		worktreeFs = osfs.New(dir)
		storerFs = osfs.New(filepath.Join(dir, ".git"))
		storer = filesystem.NewStorage(storerFs, cache.NewObjectLRUDefault())
	} else if mode == StorageModeInMemory {
		worktreeFs = memfs.New()
		storer = memory.NewStorage()
	} else {
		return nil, nil, fmt.Errorf("mode must be memory(mem) or disk(disk), '%s' is not valid", mode)
	}
	return storer, worktreeFs, nil
}

func (cr *CustomRepo) createRepo(storer storage.Storer) (*git.Repository, error) {
	r, err := git.Init(storer, cr.Fs)
	if err == git.ErrRepositoryAlreadyExists {
		r, _ := git.Open(storer, cr.Fs)
		return r, err
	} else if err != nil {
		return nil, fmt.Errorf("unable to initialize git repo: %w ", err)
	}
	return r, nil
}

func (cr *CustomRepo) addAllResources() error {
	for _, resourceListType := range getAllResourceListTypes() {
		if err := cr.createResourceDir(resourceListType); err != nil {
			return fmt.Errorf("unable to create directory for resource type %s: %w", resourceListType.String(), err)
		}
		if err := cr.addResource(resourceListType); err != nil {
			return fmt.Errorf("unable to add resources for type %s: %w", resourceListType.String(), err)
		}
	}
	return nil
}

func (cr *CustomRepo) addResource(resourceList schema.GroupVersionKind) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(resourceList)
	resources, err := cr.K8s.ListResource(list)
	if err != nil {
		return fmt.Errorf("could not list resource APIVersion: %s Kind: %s: %w", list.GetAPIVersion(), list.GetKind(), err)
	}
	var namespaces []string
	for i, np := range resources.Items {
		clearFields(&resources.Items[i])
		name := np.GetName()
		namespace := np.GetNamespace()
		if !stringInSlice(namespace, namespaces) {
			namespaces = append(namespaces, namespace)
			namespaceDir := computePath("", gvkDirMap[resourceList], namespace, "")
			cr.Fs.MkdirAll(namespaceDir, 0700)
		}
		path := computePath("", gvkDirMap[resourceList], namespace, name+".yaml")
		y, err := yaml.Marshal(&resources.Items[i])
		if err != nil {
			return fmt.Errorf("could not marshal resource config: %w", err)
		}
		if err := cr.writeFileToPath(path, y); err != nil {
			return fmt.Errorf("could not write yaml to path %s: %w", path, err)
		}
		klog.V(2).InfoS("added resource", "path", path)
	}
	return nil
}

func (cr *CustomRepo) createResourceDir(resourceList schema.GroupVersionKind) error {
	resourceDir := computePath("", gvkDirMap[resourceList], "", "")
	err := cr.Fs.MkdirAll(resourceDir, 0700)
	if err != nil {
		return fmt.Errorf("unable to create resource directory: %w", err)
	}
	return nil
}

func (cr *CustomRepo) writeFileToPath(path string, yaml []byte) error {
	newFile, err := cr.Fs.Create(path)
	if err != nil {
		return fmt.Errorf("unable to write resource config to file: %w", err)
	}
	defer newFile.Close()
	newFile.Write(yaml)
	return nil
}
