package capsule

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type GitHubCollector struct {
	BaseURL, Token string
	Client         *http.Client
}
type runResponse struct {
	HTMLURL string `json:"html_url"`
	HeadSHA string `json:"head_sha"`
	Path    string `json:"path"`
}
type jobResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
	Steps      []struct {
		Name       string `json:"name"`
		Conclusion string `json:"conclusion"`
	} `json:"steps"`
}

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func validRepository(repository string) bool {
	if !repositoryPattern.MatchString(repository) {
		return false
	}
	parts := strings.Split(repository, "/")
	return parts[0] != "." && parts[0] != ".." && parts[1] != "." && parts[1] != ".."
}

func validLogRedirect(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" {
		return false
	}
	return parsed.Port() == "" && strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".blob.core.windows.net")
}

// Collect makes only GET requests to the supplied GitHub API base URL. It does not rerun jobs or download artifacts.
func (collector GitHubCollector) Collect(repository string, runID int64) (Input, error) {
	if !validRepository(repository) {
		return Input{}, fmt.Errorf("repository must be a valid owner/repo identifier")
	}
	base := strings.TrimRight(collector.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	client := collector.Client
	if client == nil {
		client = http.DefaultClient
	}
	apiClient := *client
	apiClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	get := func(path string, output any) error {
		request, err := http.NewRequest(http.MethodGet, base+path, nil)
		if err != nil {
			return err
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if collector.Token != "" {
			request.Header.Set("Authorization", "Bearer "+collector.Token)
		}
		response, err := apiClient.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode >= 300 && response.StatusCode <= 399 {
			return fmt.Errorf("GET %s: unexpected redirect", path)
		}
		if response.StatusCode < 200 || response.StatusCode > 299 {
			return fmt.Errorf("GET %s: %s", path, response.Status)
		}
		return json.NewDecoder(response.Body).Decode(output)
	}
	var run runResponse
	if err := get(fmt.Sprintf("/repos/%s/actions/runs/%d", repository, runID), &run); err != nil {
		return Input{}, err
	}
	var allJobs []jobResponse
	for page := 1; ; page++ {
		var jobs struct {
			Jobs []jobResponse `json:"jobs"`
		}
		if err := get(fmt.Sprintf("/repos/%s/actions/runs/%d/jobs?per_page=100&page=%d", repository, runID, page), &jobs); err != nil {
			return Input{}, err
		}
		allJobs = append(allJobs, jobs.Jobs...)
		if len(jobs.Jobs) == 0 {
			break
		}
	}
	var failed *jobResponse
	var failedStep string
	for index := range allJobs {
		if allJobs[index].Conclusion == "failure" {
			failed = &allJobs[index]
			for _, step := range failed.Steps {
				if step.Conclusion == "failure" {
					failedStep = step.Name
					break
				}
			}
			break
		}
	}
	if failed == nil || failedStep == "" {
		return Input{}, fmt.Errorf("run has no failed job and failed step")
	}
	var contents struct {
		Content string `json:"content"`
	}
	if err := get(fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repository, url.PathEscape(run.Path), url.QueryEscape(run.HeadSHA)), &contents); err != nil {
		return Input{}, err
	}
	workflow, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(contents.Content, "\n", ""))
	if err != nil {
		return Input{}, fmt.Errorf("decode workflow: %w", err)
	}
	var allArtifacts []struct {
		Name    string `json:"name"`
		Size    int64  `json:"size_in_bytes"`
		Digest  string `json:"digest"`
		Expired bool   `json:"expired"`
	}
	for page := 1; ; page++ {
		var artifactsPayload struct {
			Artifacts []struct {
				Name    string `json:"name"`
				Size    int64  `json:"size_in_bytes"`
				Digest  string `json:"digest"`
				Expired bool   `json:"expired"`
			} `json:"artifacts"`
		}
		if err := get(fmt.Sprintf("/repos/%s/actions/runs/%d/artifacts?per_page=100&page=%d", repository, runID, page), &artifactsPayload); err != nil {
			return Input{}, err
		}
		allArtifacts = append(allArtifacts, artifactsPayload.Artifacts...)
		if len(artifactsPayload.Artifacts) == 0 {
			break
		}
	}
	artifacts := make([]Artifact, 0, len(allArtifacts))
	for _, artifact := range allArtifacts {
		artifacts = append(artifacts, Artifact{Name: artifact.Name, SizeBytes: artifact.Size, Digest: artifact.Digest, Expired: artifact.Expired})
	}
	logBytes, err := collector.getJobLog(client, base+fmt.Sprintf("/repos/%s/actions/jobs/%d/logs", repository, failed.ID))
	if err != nil {
		return Input{}, err
	}
	log, err := NormalizeJobLog(logBytes)
	if err != nil {
		return Input{}, err
	}
	return Input{Source: Source{Repository: repository, RunURL: run.HTMLURL, SHA: run.HeadSHA, WorkflowPath: run.Path}, Workflow: workflow, FailedJob: failed.Name, FailedStep: failedStep, Log: log, Artifacts: artifacts}, nil
}

func (collector GitHubCollector) getJobLog(client *http.Client, endpoint string) ([]byte, error) {
	noRedirect := *client
	noRedirect.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if collector.Token != "" {
		request.Header.Set("Authorization", "Bearer "+collector.Token)
	}
	response, err := noRedirect.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode <= 399 {
		location := response.Header.Get("Location")
		if location == "" || !validLogRedirect(location) {
			return nil, fmt.Errorf("GET job logs: redirect target is not an allowed absolute HTTPS URL")
		}
		anonymous := &http.Client{Transport: client.Transport, Timeout: client.Timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
		redirected, err := anonymous.Get(location)
		if err != nil {
			return nil, err
		}
		defer redirected.Body.Close()
		if redirected.StatusCode < 200 || redirected.StatusCode > 299 {
			return nil, fmt.Errorf("GET redirected job logs: %s", redirected.Status)
		}
		return readBounded(redirected.Body)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("GET job logs: %s", response.Status)
	}
	return readBounded(response.Body)
}
