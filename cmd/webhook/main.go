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

	"antrea.io/resource-auditing/pkg/webhook"
	"antrea.io/resource-auditing/pkg/webhook/gitmanager"
)

var (
	portFlag string
	dirFlag  string
)

func processArgs() {
	flag.StringVar(&portFlag, "p", "8080", "specifies port that audit webhook listens on")
	flag.StringVar(&dirFlag, "d", "", "specifies directory where resource repository is created, defaults to current working directory")
	flag.Parse()
}

func main() {
	klog.InitFlags(nil)
	processArgs()
	k8s, err := gitmanager.NewKubernetes()
	if err != nil {
		klog.ErrorS(err, "unable to create new kube clients")
		return
	}
	cr, err := gitmanager.SetupRepo(k8s, gitmanager.StorageModeDisk, dirFlag)
	if err != nil {
		klog.ErrorS(err, "unable to set up resource repository")
		return
	}
	if err := webhook.ReceiveEvents(dirFlag, portFlag, cr); err != nil {
		klog.ErrorS(err, "an error occurred while running the audit webhook service")
		return
	}
}
