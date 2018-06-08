package resource

import (
	"log"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Check (business logic)
func Check(request CheckRequest, manager Github) (CheckResponse, error) {
	var response CheckResponse

	pulls, err := manager.ListOpenPullRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get last commits: %s", err)
	}
	var disableSkipCI bool
	if request.Source.DisableCISkip != "" {
		disableSkipCI, err = strconv.ParseBool(request.Source.DisableCISkip)
		if err != nil {
			return nil, fmt.Errorf("failed to parse disable_ci_skip: %s", err)
		}
	}

	var newPullsToReturn []*PullRequest

Loop:
	for _, p := range pulls {
		// [ci skip]/[skip ci] in Pull request title
		if !disableSkipCI && ContainsSkipCI(p.Title) {
			continue
		}
		// [ci skip]/[skip ci] in Commit message
		if !disableSkipCI && ContainsSkipCI(p.Tip.Message) {
			continue
		}

		// Fetch files once if paths/ignore_paths are specified.
		var files []string

		if len(request.Source.Paths) > 0 || len(request.Source.IgnorePaths) > 0 {
			files, err = manager.ListModifiedFiles(p.Number)
			if err != nil {
				return nil, fmt.Errorf("failed to list modified files: %s", err)
			}
		}

		// Skip version if no files match the specified paths.
		if len(request.Source.Paths) > 0 {
			var wanted []string
			for _, pattern := range request.Source.Paths {
				w, err := FilterPath(files, pattern)
				if err != nil {
					return nil, fmt.Errorf("path match failed: %s", err)
				}
				wanted = append(wanted, w...)
			}
			if len(wanted) == 0 {
				continue Loop
			}
		}

		// Skip version if all files are ignored.
		if len(request.Source.IgnorePaths) > 0 {
			wanted := files
			for _, pattern := range request.Source.IgnorePaths {
				wanted, err = FilterIgnorePath(wanted, pattern)
				if err != nil {
					return nil, fmt.Errorf("ignore path match failed: %s", err)
				}
			}
			if len(wanted) == 0 {
				continue Loop
			}
		}

		// TODO: determine above/below the fold
		if AboveTheFold(GetVersionStringFromPullRequest(p), request.Version.AlreadySeen) {
			newPullsToReturn = append(newPullsToReturn, p)
		}
		// alreadySeenPullsToReturn = append(alreadySeenPullsToReturn, p)
	}

	var versionsJustSeen = GenerateVersion(newPullsToReturn)

	// Add "above-the-fold" with new alreadySeen version strings
	for _, p := range newPullsToReturn {
		response = append(response, NewVersion(p, versionsJustSeen))
	}
	// Sort the commits by date
	sort.Sort(response)

	// If there are no new but an old version = return the old
	if len(response) == 0 && request.Version.PR != "" {
		response = append(response, request.Version)
	}
	// If there are new versions and no previous = return just the latest
	if len(response) != 0 && request.Version.AlreadySeen == "" {
		response = CheckResponse{response[len(response)-1]}
	}
	return response, nil
}

type alreadySeenVersion struct {
	PR            string
	committedDate time.Time
}

// GetVersionStringFromPullRequest returns string-serialized representation of latest commit in a PR
func GetVersionStringFromPullRequest(pull *PullRequest) string {
	return strconv.Itoa(pull.Number) + ":" + strconv.FormatInt(pull.Tip.CommittedDate.Time.UnixNano(), 10)
}

// GenerateVersion returns a string-formatted array of PR#:CommittedDate
func GenerateVersion(pulls []*PullRequest) string {
	var pairs []string
	for _, p := range pulls {
		pairs = append(pairs, GetVersionStringFromPullRequest(p))
	}
	return strings.Join(pairs, ",")
}

func ExtractVersion(alreadySeenPair string) alreadySeenVersion {
	log.Printf("extracting %v", alreadySeenPair)
	var pairs = strings.Split(alreadySeenPair, ":")
	var committedDate, _ = time.Parse(time.UnixDate, pairs[1])
	return alreadySeenVersion{PR: pairs[0], committedDate: committedDate}
}

func AboveTheFold(pullRequestVersion string, alreadySeen string) bool {
	log.Printf("Check the fold? %v : %v", pullRequestVersion, alreadySeen)
	if !strings.Contains(alreadySeen, ":") {
		log.Printf("No pairs, must be above the fold %v", pullRequestVersion)
		return true
	}
	var pairs = strings.Split(alreadySeen, ",")
	var isAboveTheFold = false
	var isFoundInPairs = false
	var pullRequest = ExtractVersion(pullRequestVersion)
	for _, pair := range pairs {
		log.Printf("Checking pair %v", pair)
		var thisPairVersion = ExtractVersion(pair)
		if thisPairVersion.PR == pullRequest.PR {
			isFoundInPairs = true
			if thisPairVersion.committedDate.Before(pullRequest.committedDate) {
				log.Printf("Pull request is newer commit %v", pullRequest.PR)
				isAboveTheFold = true
			}
		}
	}
	if !isFoundInPairs {
		log.Printf("Pull request is not in pairs, so Above the fold %v", pullRequest.PR)
		isAboveTheFold = true
	}
	return isAboveTheFold
}

// ContainsSkipCI returns true if a string contains [ci skip] or [skip ci].
func ContainsSkipCI(s string) bool {
	re := regexp.MustCompile("(?i)\\[(ci skip|skip ci)\\]")
	return re.MatchString(s)
}

// FilterIgnorePath ...
func FilterIgnorePath(files []string, pattern string) ([]string, error) {
	var out []string
	for _, file := range files {
		match, err := filepath.Match(pattern, file)
		if err != nil {
			return nil, err
		}
		if !match {
			out = append(out, file)
		}
	}
	return out, nil
}

// FilterPath ...
func FilterPath(files []string, pattern string) ([]string, error) {
	var out []string
	for _, file := range files {
		match, err := filepath.Match(pattern, file)
		if err != nil {
			return nil, err
		}
		if match {
			out = append(out, file)
		}
	}
	return out, nil
}

// CheckRequest ...
type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

// CheckResponse ...
type CheckResponse []Version

func (r CheckResponse) Len() int {
	return len(r)
}

func (r CheckResponse) Less(i, j int) bool {
	return r[j].CommittedDate.After(r[i].CommittedDate)
}

func (r CheckResponse) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
