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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleEventList(t *testing.T) {
	fakeClient := NewClient(np1.DeepCopy(), anp1.DeepCopy())
	k8s := &K8sClient{
		Client: fakeClient,
	}

	jsonstring, err := ioutil.ReadFile("../../test/files/correct-audit-log.txt")
	assert.NoError(t, err, "unable to read mock audit log")

	cr, err := SetupRepo(k8s, StorageModeInMemory, dir)
	assert.NoError(t, err, "could not set up repo")

	err = cr.HandleEventList(jsonstring)
	assert.NoError(t, err, "could not handle correct audit event list")

	cr.RollbackMode = true
	err = cr.HandleEventList(jsonstring)
	cr.RollbackMode = false
	assert.EqualError(t, err, "audit skipped - rollback in progress")

	for i := 1; i < 4; i++ {
		filename := fmt.Sprintf("%s%d%s", "../../test/files/incorrect-audit-log-", i, ".txt")
		jsonstring, err := ioutil.ReadFile(filename)
		assert.NoError(t, err, "unable to read audit log")
		err = cr.HandleEventList(jsonstring)
		assert.Error(t, err, "should have returned error on bad audit log")
	}
}
