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
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (cr *CustomRepo) TagToCommit(tag string) (*object.Commit, error) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	ref, err := cr.Repo.Tag(tag)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve tag reference: %w", err)
	}
	obj, err := cr.Repo.TagObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("could not retrieve tag object: %w", err)
	}
	commit, err := obj.Commit()
	if err != nil {
		return nil, fmt.Errorf("could not get commit from tag object: %w", err)
	}
	return commit, nil
}

func (cr *CustomRepo) HashToCommit(commitSha string) (*object.Commit, error) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	hash := plumbing.NewHash(commitSha)
	commit, err := cr.Repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("could not get commit from hash: %w", err)
	}
	return commit, nil
}

func (cr *CustomRepo) RollbackRepo(targetCommit *object.Commit) error {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()

	klog.V(2).InfoS("rollback initiated, ignoring all non-rollback generated audits",
		"targetCommit", targetCommit.Hash.String())
	cr.RollbackMode = true

	// Get patch between head and target commit
	w, err := cr.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to get git worktree from repository: %w", err)
	}
	h, err := cr.Repo.Head()
	if err != nil {
		return fmt.Errorf("unable to get repo head: %w", err)
	}
	headCommit, err := cr.Repo.CommitObject(h.Hash())
	if err != nil {
		return fmt.Errorf("unable to get head commit: %w", err)
	}
	patch, err := headCommit.Patch(targetCommit)
	if err != nil {
		return fmt.Errorf("unable to get patch between commits: %w", err)
	}

	// Must do cluster delete requests before resetting in order to be able to read metadata from files
	if err := cr.doDeletePatch(patch); err != nil {
		return fmt.Errorf("could not patch cluster to old commit state (delete phase): %w", err)
	}

	// Update repo using resets
	err = resetWorktree(w, targetCommit.Hash, git.HardReset)
	if err != nil {
		return fmt.Errorf("unable to hard reset repo: %w", err)
	}
	err = resetWorktree(w, h.Hash(), git.SoftReset)
	if err != nil {
		return fmt.Errorf("unable to hard reset repo: %w", err)
	}

	// Must similarly do cluster update/create requests after resetting
	if err := cr.doCreateUpdatePatch(patch); err != nil {
		return fmt.Errorf("could not patch cluster to old commit state (create/update phase): %w", err)
	}

	// Finally commit changes to repo after cluster updates
	username := "audit-manager"
	email := "system@audit.antrea.io"
	message := "Rollback to commit " + targetCommit.Hash.String()
	if err := cr.AddAndCommit(username, email, message); err != nil {
		return fmt.Errorf("error while committing rollback: %w", err)
	}
	cr.RollbackMode = false
	klog.V(2).InfoS("rollback successful", "targetCommit", targetCommit.Hash.String())
	return nil
}

func resetWorktree(w *git.Worktree, hash plumbing.Hash, mode git.ResetMode) error {
	options := &git.ResetOptions{
		Commit: hash,
		Mode:   mode,
	}
	if err := w.Reset(options); err != nil {
		return fmt.Errorf("unable to reset worktree: %w", err)
	}
	return nil
}

func (cr *CustomRepo) doDeletePatch(patch *object.Patch) error {
	for _, filePatch := range patch.FilePatches() {
		fromFile, toFile := filePatch.Files()
		if toFile == nil {
			path := fromFile.Path()
			resource, err := cr.getResourceByPath(path)
			if err != nil {
				return fmt.Errorf("unable to read resource at path %s: %w", path, err)
			}
			if err := cr.K8s.DeleteResource(resource); err != nil {
				return fmt.Errorf("unable to delete resource %s: %w", resource.GetName(), err)
			}
			klog.V(2).InfoS("(rollback) deleted file", "path", path)
		}
	}
	return nil
}

func (cr *CustomRepo) doCreateUpdatePatch(patch *object.Patch) error {
	for _, filePatch := range patch.FilePatches() {
		_, toFile := filePatch.Files()
		if toFile != nil {
			path := toFile.Path()
			resource, err := cr.getResourceByPath(path)
			if err != nil {
				return fmt.Errorf("unable to read resource at path %s: %w", path, err)
			}
			if err := cr.K8s.CreateOrUpdateResource(resource); err != nil {
				return fmt.Errorf("unable to create/update resource %s: %w", resource.GetName(), err)
			}
			klog.V(2).InfoS("(rollback) created/updated file", "path", path)
		}
	}
	return nil
}

func (cr *CustomRepo) getResourceByPath(path string) (*unstructured.Unstructured, error) {
	resource := &unstructured.Unstructured{}
	gvk := schema.GroupVersionKind{}
	if err := cr.readResource(resource, path); err != nil {
		return nil, fmt.Errorf("unable to read resource: %w", err)
	}
	apiVersion := resource.GetAPIVersion()
	kind := resource.GetKind()
	if apiVersion == "networking.k8s.io/v1" {
		gvk.Group = "networking.k8s.io"
		gvk.Version = "v1"
	} else if apiVersion == "crd.antrea.io/v1alpha1" {
		gvk.Group = "crd.antrea.io"
		gvk.Version = "v1alpha1"
	} else {
		return nil, fmt.Errorf("unknown apiVersion found: %s", apiVersion)
	}
	gvk.Kind = kind
	resource.SetGroupVersionKind(gvk)
	return resource, nil
}

func (cr *CustomRepo) readResource(resource *unstructured.Unstructured, path string) error {
	var y []byte
	klog.Info(path)
	fstat, _ := cr.Fs.Stat(path)
	y = make([]byte, fstat.Size())
	f, err := cr.Fs.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	f.Read(y)
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return fmt.Errorf("error converting from YAML to JSON: %w", err)
	}
	if err := json.Unmarshal(j, &resource.Object); err != nil {
		return fmt.Errorf("error while unmarshalling from file: %w", err)
	}
	return nil
}
