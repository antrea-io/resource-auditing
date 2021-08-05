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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"k8s.io/klog/v2"
)

func (cr *CustomRepo) TagCommit(commitSha string, tag string, tagger *object.Signature) (string, error) {
	hash := plumbing.NewHash(commitSha)
	_, err := cr.Repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("unable to get commit object: %w", err)
	}
	if err = setTag(cr.Repo, hash, tag, tagger); err != nil {
		return "", fmt.Errorf("unable to create tag: %w", err)
	}
	klog.V(2).InfoS("Tag created", "tagName", tag, "commit", commitSha)
	return commitSha, nil
}

func (cr *CustomRepo) RemoveTag(tag string) (string, error) {
	if err := cr.Repo.DeleteTag(tag); err != nil {
		return "", fmt.Errorf("unable to delete tag: %w", err)
	}
	klog.V(2).InfoS("Tag deleted", "tagName", tag)
	return tag, nil
}

func setTag(r *git.Repository, commitSha plumbing.Hash, tag string, tagger *object.Signature) error {
	_, err := r.CreateTag(tag, commitSha, &git.CreateTagOptions{
		Tagger:  tagger,
		Message: tag,
	})
	if err != nil {
		if err.Error() == "tag already exists" {
			return err
		} else {
			return fmt.Errorf("error creating tag: %w", err)
		}
	}
	return nil
}
