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

package gitmanager

import (
	"sync"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
)

type StorageModeType string

const (
	StorageModeDisk     StorageModeType = "Disk"
	StorageModeInMemory StorageModeType = "InMemory"
)

type CustomRepo struct {
	Repo           *git.Repository
	K8s            *KubeClients
	RollbackMode   bool
	StorageMode    StorageModeType
	ServiceAccount string
	Dir            string
	Fs             billy.Filesystem
	Mutex          sync.Mutex
}

func SetupRepo(k *KubeClients, mode StorageModeType, dir string) (*CustomRepo, error) {
	cr := CustomRepo{
		K8s:            k,
		RollbackMode:   false,
		StorageMode:    mode,
		ServiceAccount: "",
		Dir:            dir,
		Fs:             nil,
	}
	return &cr, nil
}
