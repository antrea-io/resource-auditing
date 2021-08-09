package gitops

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func (cr *CustomRepo) FilterCommits(author string, since time.Time, until time.Time, resource string, namespace string, name string) ([]object.Commit, error) {
	var logopts git.LogOptions
	var filteredCommits []object.Commit

	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	ref, err := cr.Repo.Head()
	if err != nil {
		return filteredCommits, fmt.Errorf("could not get ref head from repository: %w", err)
	}

	logopts.From = ref.Hash()
	if !since.IsZero() {
		logopts.Since = &since
	}
	if !until.IsZero() {
		logopts.Until = &until
	}

	filterByAuthor := func(commit *object.Commit) bool {
		return commit.Author.Name == author
	}
	filters := []customFilterFn{}
	if author != "" {
		filters = append(filters, filterByAuthor)
	}
	setPathFilter(resource, namespace, name, &logopts)
	filteredCommits, err = cr.filter(&logopts, filters...)
	return filteredCommits, err
}

type customFilterFn func(*object.Commit) bool

func (cr *CustomRepo) filter(logopts *git.LogOptions, customFilters ...customFilterFn) ([]object.Commit, error) {
	var filteredCommits []object.Commit
	cIter, err := cr.Repo.Log(logopts)
	if err != nil {
		return filteredCommits, fmt.Errorf("could not get logs from repo: %w", err)
	}

	err = cIter.ForEach(func(c *object.Commit) error {
		passed := true
		for _, filterFn := range customFilters {
			if !filterFn(c) {
				passed = false
				break
			}
		}
		if passed {
			filteredCommits = append(filteredCommits, *c)
		}
		return nil
	})
	return filteredCommits, err
}

func setPathFilter(resource string, namespace string, name string, logopts *git.LogOptions) {
	if resource == "" {
		resource = "*"
	}
	if namespace == "" {
		namespace = "*"
	}
	if name == "" {
		name = "*"
	}
	pattern := filepath.Join(resource, namespace, name)
	logopts.PathFilter = func(path string) bool {
		b, _ := filepath.Match(pattern, path)
		return b
	}
}
