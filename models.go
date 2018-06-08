package resource

import (
	"errors"
	"time"

	"github.com/shurcooL/githubv4"
)

// Source represents the configuration for the resource.
type Source struct {
	Repository    string   `json:"repository"`
	AccessToken   string   `json:"access_token"`
	V3Endpoint    string   `json:"v3_endpoint"`
	V4Endpoint    string   `json:"v4_endpoint"`
	Paths         []string `json:"path"`
	IgnorePaths   []string `json:"ignore_path"`
	DisableCISkip string   `json:"disable_ci_skip"`
}

// Validate the source configuration.
func (s *Source) Validate() error {
	if s.AccessToken == "" {
		return errors.New("access_token must be set")
	}
	if s.Repository == "" {
		return errors.New("repository must be set")
	}
	if s.V3Endpoint != "" && s.V4Endpoint == "" {
		return errors.New("v4_endpoint must be set together with v3_endpoint")
	}
	if s.V4Endpoint != "" && s.V3Endpoint == "" {
		return errors.New("v3_endpoint must be set together with v4_endpoint")
	}
	return nil
}

// Metadata output from get/put steps.
type Metadata []*MetadataField

// Add a MetadataField to the Metadata.
func (m *Metadata) Add(name, value string) {
	*m = append(*m, &MetadataField{Name: name, Value: value})
}

// MetadataField ...
type MetadataField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AlreadySeenVersion Interesting values compressed into data blob stored in Version.AlreadySeen
type AlreadySeenVersion struct {
	PR            string
	committedDate time.Time
}

// Version communicated with Concourse. ID is the Github Global ID.
type Version struct {
	PR            string    `json:"pr"`
	Commit        string    `json:"commit"`
	CommittedDate time.Time `json:"committed,omitempty"`
	AlreadySeen   string    `json:"alreadyseen"`
}

// NewVersion constructs a new Version.
func NewVersion(p *PullRequest, alreadySeen string) Version {
	return Version{
		PR:            p.ID,
		Commit:        p.Tip.ID,
		CommittedDate: p.Tip.CommittedDate.Time,
		AlreadySeen:   alreadySeen,
	}
}

// PullRequest represents a pull request and includes the tip (commit).
type PullRequest struct {
	PullRequestObject
	Tip CommitObject
}

// PullRequestObject represents the GraphQL commit node.
// https://developer.github.com/v4/object/commit/
type PullRequestObject struct {
	ID          string
	Number      int
	Title       string
	URL         string
	BaseRefName string
	HeadRefName string
	Repository  struct {
		URL string
	}
}

// CommitObject represents the GraphQL commit node.
// https://developer.github.com/v4/object/commit/
type CommitObject struct {
	ID            string
	OID           string
	CommittedDate githubv4.DateTime
	Message       string
	Author        struct {
		User struct {
			Login string
		}
	}
}
