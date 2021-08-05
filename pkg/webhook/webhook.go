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

package webhook

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"antrea.io/resource-auditing/pkg/gitops"
	"antrea.io/resource-auditing/pkg/types"

	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"
)

func events(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	klog.V(3).Infof("Audit received: %s", string(body))
	if err := cr.HandleEventList(body); err != nil {
		if err.Error() == "rollback in progress" {
			klog.ErrorS(err, "audit received during rollback")
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			klog.ErrorS(err, "unable to process audit event list")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
}

func tag(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	if r.Method != "POST" {
		klog.Errorf("tag does not accept non-POST request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tagRequest := types.TagRequest{}
	if err := json.Unmarshal(body, &tagRequest); err != nil {
		klog.ErrorS(err, "unable to marshal request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if tagRequest.Type == types.TagCreate {
		signature := object.Signature{
			Name:  tagRequest.Author,
			Email: tagRequest.Email,
			When:  time.Now(),
		}
		sha, err := cr.TagCommit(tagRequest.Sha, tagRequest.Tag, &signature)
		if err != nil {
			klog.ErrorS(err, "failed to tag commit")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Commit " + sha + " tagged"))
	} else if tagRequest.Type == types.TagDelete {
		tag, err := cr.RemoveTag(tagRequest.Tag)
		if err != nil {
			klog.ErrorS(err, "failed to delete tag")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Tag " + tag + " deleted"))
	} else {
		klog.ErrorS(err, "unknown tag request type found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func rollback(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	if r.Method != "POST" {
		klog.Errorf("rollback does not accept non-POST request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rollbackRequest := types.RollbackRequest{}
	if err := json.Unmarshal(body, &rollbackRequest); err != nil {
		klog.ErrorS(err, "unable to marshal request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var commit *object.Commit
	if rollbackRequest.Tag != "" {
		commit, err = cr.TagToCommit(rollbackRequest.Tag)
	} else if rollbackRequest.Sha != "" {
		commit, err = cr.HashToCommit(rollbackRequest.Sha)
	}
	if err != nil {
		klog.ErrorS(err, "unable to convert user input into commit object")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sha, err := cr.RollbackRepo(commit)
	if err != nil {
		klog.ErrorS(err, "failed to rollback repo")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Rollback to commit " + sha + " successful"))
}

func ReceiveEvents(port string, cr *gitops.CustomRepo) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		events(w, r, cr)
	})
	http.HandleFunc("/rollback", func(w http.ResponseWriter, r *http.Request) {
		rollback(w, r, cr)
	})
	http.HandleFunc("/tag", func(w http.ResponseWriter, r *http.Request) {
		tag(w, r, cr)
	})
	klog.V(2).Infof("Audit webhook server started, listening on port %s", port)
	if err := http.ListenAndServe(":"+string(port), nil); err != nil {
		klog.ErrorS(err, "Audit webhook service died")
		return err
	}
	return nil
}
