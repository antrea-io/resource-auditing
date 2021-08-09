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
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

func TestTagging(t *testing.T) {
	fakeClient := NewClient()
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
	// Attempt to add tag to nonexistent commit
	_, err = cr.TagCommit("bad-hash", "test-tag", testSig)
	assert.Error(t, err, "should have returned error on bad commit hash")

	// Create new tags successfully
	_, err = cr.TagCommit(h.Hash().String(), "test-tag", testSig)
	assert.NoError(t, err, "unable to create 1st new tag")
	_, err = cr.TagCommit(h.Hash().String(), "test-tag-2", testSig)
	assert.NoError(t, err, "unable to create 2nd new tag")

	_, err = cr.Repo.Tag("test-tag")
	assert.NoError(t, err, "could not retrieve 1st created tag")
	_, err = cr.Repo.Tag("test-tag-2")
	assert.NoError(t, err, "could not retrieve 2nd created tag")

	// Attempt to add tag with the same name
	_, err = cr.TagCommit(h.Hash().String(), "test-tag", testSig)
	assert.EqualError(t, err, "unable to create tag: tag already exists")

	tags, _ := cr.Repo.Tags()
	tagCount := 0
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagCount += 1
		return nil
	})
	assert.NoError(t, err, "could not iterate through repo tags")
	assert.Equal(t, 2, tagCount, "unexpected number of tags, should have 2 tags")

	// Delete tag
	_, err = cr.RemoveTag("test-tag-2")
	assert.NoError(t, err, "unable to delete 2nd tag")
	tags, _ = cr.Repo.Tags()
	tagCount = 0
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagCount += 1
		return nil
	})
	assert.NoError(t, err, "could not iterate through repo tags")
	assert.Equal(t, 1, tagCount, "unexpected number of tags, should have 1 tag")
}
