package gitops

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilterCommits(t *testing.T) {
	start := time.Now()
	time.Sleep(time.Millisecond * 500) // delay to emulate a time range

	empty := ""
	author := "kubernetes-admin"
	zerotime := time.Time{}
	resource := "k8s-policies"
	namespace := "default"
	name := "allow-client1.yaml"
	fakeClient := NewClient(Np1.inputResource, Anp1.inputResource)
	k8s := &K8sClient{
		Client: fakeClient,
	}

	jsonStr, err := ioutil.ReadFile("../../test/files/correct-audit-log.txt")
	assert.NoError(t, err, "could not read mock audit log")

	cr, err := SetupRepo(k8s, StorageModeInMemory, empty)
	assert.NoError(t, err, "could not set up mock repo")

	err = cr.HandleEventList(jsonStr)
	assert.NoError(t, err, "could not handle mock eventlist")
	until := time.Now()

	// query by author and time range
	commits, err := cr.FilterCommits(author, start, until, empty, empty, empty)
	assert.NoError(t, err, "could not filter commits with time range")
	for _, c := range commits {
		assert.Equal(t, "kubernetes-admin", c.Author.Name, "incorrect commit author in author and time query")
		assert.NotEqual(t, "", c.Message, "commit message is empty")
	}

	// query by namespace
	commits, err = cr.FilterCommits(empty, zerotime, zerotime, empty, namespace, empty)
	assert.NoError(t, err, "could not filter commits by namespace")
	assert.Equal(t, 3, len(commits), "could not get the correct amount of commits")
	for _, c := range commits {
		assert.Equal(t, "kubernetes-admin", c.Author.Name, "incorrect commit author in namespace query")
	}

	// query by resource, namespace, and name
	commits, err = cr.FilterCommits(empty, zerotime, zerotime, resource, namespace, name)
	assert.NoError(t, err, "could not filter by resource, namespace, and name")
	assert.Equal(t, 3, len(commits), "could not get the correct amount of commits")
	for _, c := range commits {
		assert.Equal(t, "kubernetes-admin", c.Author.Name, "incorrect commit author in resource, namespace, and name query")
	}
}
