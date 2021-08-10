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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"antrea.io/resource-auditing/pkg/types"
)

// tag flags
var tagAuthor, tagEmail string

// rollback flags
var rollbackTag, rollbackSHA string

// shared flags
var serverAddr string

var commandName = "auditctl"

var rootCmd = &cobra.Command{
	Use:  commandName,
	Long: commandName + " is the command line tool for managing the auditing resource repository",
}

var tagCmd = &cobra.Command{
	Use:   "tag create tag_name commit_sha [-a author] [-e email]\n   or: tag delete tag_name",
	Short: "tags commits in the repository",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("too few args")
		}
		if args[0] == "create" {
			if len(args) != 3 {
				return fmt.Errorf("unexpected number of args for tag create")
			}
		} else if args[0] == "delete" {
			if len(args) != 2 {
				return fmt.Errorf("unexpected number of args for tag delete")
			}
		} else {
			return fmt.Errorf("unsupported keyword (not create or delete)")
		}
		return nil
	},
	Run: runTag,
	Example: `	Tag a commit in the auditing repository
	$ auditctl tag create new-tag 6dd1f926c346f06fc2c57d356ed648a2b518e74c
	Tag with author/email signature
	$ auditctl tag create new-tag-2 6dd1f926c346f06fc2c57d356ed648a2b518e74c -a current-user -e user@audit.io
	Delete a tag
	$ auditctl tag delete new-tag`,
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback -t tag_name | -s commit_sha",
	Short: "rollback to the specified commit by tag name or SHA",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("unexpected number of args for rollback")
		}
		if (rollbackTag != "") == (rollbackSHA != "") {
			return fmt.Errorf("must specify exactly one of -t or -s")
		}
		return nil
	},
	Run: runRollback,
	Example: `	Rollback by tag name
	$ auditctl rollback -t new-tag
	Rollback by commit hash
	$ auditctl rollback -s 6dd1f926c346f06fc2c57d356ed648a2b518e74c`,
}

func runTag(cmd *cobra.Command, args []string) {
	var request types.TagRequest
	if args[0] == "create" {
		request = types.TagRequest{
			Type:   types.TagCreate,
			Tag:    args[1],
			Sha:    args[2],
			Author: tagAuthor,
			Email:  tagEmail,
		}
	} else {
		request = types.TagRequest{
			Type: types.TagDelete,
			Tag:  args[1],
		}
	}
	j, err := json.Marshal(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	url := "http://" + serverAddr + "/tag"
	// #nosec G107: need user-provided URL for server
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(j))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
		fmt.Println("Error encountered while processing tag request")
		return
	}
	fmt.Println(string(body))
}

func runRollback(cmd *cobra.Command, args []string) {
	request := types.RollbackRequest{
		Tag: rollbackTag,
		Sha: rollbackSHA,
	}
	j, err := json.Marshal(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	url := "http://" + serverAddr + "/rollback"
	// #nosec G107: need user-provided URL for server
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(j))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
		fmt.Println("Error encountered while processing rollback request")
		return
	}
	fmt.Println(string(body))
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "server-addr", "S", "", "address and port of the webhook server")
	tagCmd.Flags().StringVarP(&tagAuthor, "author", "a", "no-author", "tag author")
	tagCmd.Flags().StringVarP(&tagEmail, "email", "e", "default@audit.io", "tag email")
	rootCmd.AddCommand(tagCmd)
	rollbackCmd.Flags().StringVarP(&rollbackTag, "tag", "t", "", "name of tag to rollback to")
	rollbackCmd.Flags().StringVarP(&rollbackSHA, "sha", "s", "", "commit hash to rollback to")
	rootCmd.AddCommand(rollbackCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
