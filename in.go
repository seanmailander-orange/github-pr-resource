package resource

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

// Get (business logic)
func Get(request GetRequest, github Github, git Git, outputDir string) (*GetResponse, error) {
	pull, err := github.GetPullRequest(request.Version.PR, request.Version.Commit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pull request: %s", err)
	}

	// Clone the repository and fetch the PR
	if err := git.Init(); err != nil {
		return nil, err
	}
	if err := git.Pull(pull.Repository.URL); err != nil {
		return nil, err
	}
	if err := git.Fetch(pull.Repository.URL, pull.Number); err != nil {
		return nil, err
	}

	// Create a branch from the base ref and merge PR into it
	baseSHA, err := git.RevParse(pull.BaseRefName)
	if err != nil {
		return nil, err
	}
	if err := git.Checkout(baseSHA); err != nil {
		return nil, err
	}
	if err := git.Merge(pull.Tip.OID); err != nil {
		return nil, err
	}

	// Create the metadata
	var metadata Metadata
	metadata.Add("pr", strconv.Itoa(pull.Number))
	metadata.Add("url", pull.URL)
	metadata.Add("head_sha", pull.Tip.OID)
	metadata.Add("base_sha", baseSHA)
	metadata.Add("message", pull.Tip.Message)
	metadata.Add("author", pull.Tip.Author.User.Login)

	// Write version and metadata for reuse in PUT
	path := filepath.Join(outputDir, ".git", "resource")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %s", err)
	}
	b, err := json.Marshal(request.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(path, "version.json"), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write version: %s", err)
	}
	b, err = json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(path, "metadata.json"), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %s", err)
	}

	return &GetResponse{
		Version:  request.Version,
		Metadata: metadata,
	}, nil
}

// GetParameters ...
type GetParameters struct{}

// GetRequest ...
type GetRequest struct {
	Source  Source        `json:"source"`
	Version Version       `json:"version"`
	Params  GetParameters `json:"params"`
}

// GetResponse ...
type GetResponse struct {
	Version  Version  `json:"version"`
	Metadata Metadata `json:"metadata,omitempty"`
}
