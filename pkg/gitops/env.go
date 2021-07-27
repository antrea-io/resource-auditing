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
	"os"
)

const (
	svcAcctNameEnvKey   = "SERVICEACCOUNT_NAME"
	svcAcctDefault      = "audit-account"
	podNamespaceEnvKey  = "POD_NAMESPACE"
	podNamespaceDefault = "default"
)

func GetAuditServiceAccount() string {
	svcAcctName := os.Getenv(svcAcctNameEnvKey)
	if svcAcctName == "" {
		svcAcctName = svcAcctDefault
	}
	return svcAcctName
}

func GetAuditPodNamespace() string {
	podNamespace := os.Getenv(podNamespaceEnvKey)
	if podNamespace == "" {
		podNamespace = podNamespaceDefault
	}
	return podNamespace
}
