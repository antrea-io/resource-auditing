package gitops

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

func (cr *CustomRepo) AddAndCommit(username string, email string, message string) error {
	w, err := cr.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to get git worktree from repository: %w", err)
	}
	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("unable to add git change to worktree: %w", err)
	}
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  username,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("unable to commit git change to worktree: %w", err)
	}
	return nil
}

func (cr *CustomRepo) modifyFile(event auditv1.Event) error {
	resource := unstructured.Unstructured{}
	if err := json.Unmarshal(event.ResponseObject.Raw, &resource); err != nil {
		return fmt.Errorf("unable to unmarshal ResponseObject resource config: %w", err)
	}
	clearFields(&resource)
	y, err := yaml.Marshal(&resource)
	if err != nil {
		return fmt.Errorf("unable to marshal new resource config: %w", err)
	}
	path := getAbsRepoPath(cr.Dir, event)
	if cr.StorageMode == StorageModeDisk {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, 0700)
		}
		path += getFileName(event)
		if err := ioutil.WriteFile(path, y, 0600); err != nil {
			return fmt.Errorf("unable to write/update file in repository: %w", err)
		}
	} else {
		path += getFileName(event)
		newfile, err := cr.Fs.Create(path)
		if err != nil {
			return fmt.Errorf("unable to create file at: %s: %w", path, err)
		}
		newfile.Write(y)
		newfile.Close()
	}
	return nil
}

func (cr *CustomRepo) deleteFile(event auditv1.Event) error {
	w, err := cr.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to get git worktree from repository: %w", err)
	}
	path := getRelRepoPath(event) + getFileName(event)
	_, err = w.Remove(path)
	if err != nil {
		return fmt.Errorf("unable to remove file at: %s: %w", path, err)
	}
	return nil
}
