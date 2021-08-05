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

package main

import (
	"flag"

	"k8s.io/klog/v2"

	"antrea.io/resource-auditing/pkg/gitops"
	"antrea.io/resource-auditing/pkg/webhook"
)

func processArgs() {
	flag.StringVar(&portFlag, "p", "8080", "specifies port that audit webhook listens on")
	flag.StringVar(&dirFlag, "d", "", "directory where resource repository is created, defaults to current working directory")
	flag.Parse()
}

var (
	portFlag string
	dirFlag  string
)

func main() {
	klog.InitFlags(nil)
	processArgs()
	k8s, err := gitops.NewKubernetes()
	if err != nil {
		klog.ErrorS(err, "unable to create kube client")
		return
	}
	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeDisk, dirFlag)
	if err != nil {
		klog.ErrorS(err, "unable to set up resource repository")
		return
	}
	if err := webhook.ReceiveEvents(portFlag, cr); err != nil {
		klog.ErrorS(err, "an error occurred while running the audit webhook service")
		return
	}
}
