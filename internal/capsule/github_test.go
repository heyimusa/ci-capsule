package capsule

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const fixtureWorkflowContent = "am9iczoKICB0ZXN0OgogICAgc3RlcHM6CiAgICAgIC0gbmFtZTogVGVzdAogICAgICAgIHJ1bjogZ28gdGVzdCAuLy4uCg=="

func githubFixtureHandler(logHandler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/actions/runs/7":
			_, _ = w.Write([]byte(`{"html_url":"https://github.com/o/r/actions/runs/7","head_sha":"abc","path":".github/workflows/test.yml"}`))
		case "/repos/o/r/actions/runs/7/jobs":
			if r.URL.Query().Get("page") == "1" {
				_, _ = w.Write([]byte(`{"jobs":[{"id":9,"name":"test","conclusion":"failure","steps":[{"name":"Test","conclusion":"failure"}]}]}`))
			} else {
				_, _ = w.Write([]byte(`{"jobs":[]}`))
			}
		case "/repos/o/r/actions/runs/7/artifacts":
			if r.URL.Query().Get("page") == "1" {
				_, _ = w.Write([]byte(`{"artifacts":[{"name":"report","size_in_bytes":3,"expired":false,"digest":"sha256:abc"}]}`))
			} else {
				_, _ = w.Write([]byte(`{"artifacts":[]}`))
			}
		case "/repos/o/r/contents/.github/workflows/test.yml":
			_, _ = w.Write([]byte(`{"content":"` + fixtureWorkflowContent + `"}`))
		case "/repos/o/r/actions/jobs/9/logs":
			logHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}

func TestGitHubCollectorRejectsAuthenticatedAPIRedirect(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example/", http.StatusFound)
	}))
	defer api.Close()
	_, err := (GitHubCollector{BaseURL: api.URL, Token: "token", Client: api.Client()}).Collect("o/r", 7)
	if err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("err = %v", err)
	}
}

func TestGitHubCollectorFollowsPaginationForFailedJobAndArtifacts(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch r.URL.Path {
		case "/repos/o/r/actions/runs/7":
			_, _ = w.Write([]byte(`{"html_url":"run","head_sha":"abc","path":".github/workflows/test.yml"}`))
		case "/repos/o/r/actions/runs/7/jobs":
			if page == "1" {
				_, _ = w.Write([]byte(`{"jobs":[{"id":1,"name":"ok","conclusion":"success","steps":[]}]}`))
			} else if page == "2" {
				_, _ = w.Write([]byte(`{"jobs":[{"id":9,"name":"test","conclusion":"failure","steps":[{"name":"Test","conclusion":"failure"}]}]}`))
			} else {
				_, _ = w.Write([]byte(`{"jobs":[]}`))
			}
		case "/repos/o/r/actions/runs/7/artifacts":
			if page == "1" {
				_, _ = w.Write([]byte(`{"artifacts":[{"name":"first","size_in_bytes":1,"expired":false}]}`))
			} else if page == "2" {
				_, _ = w.Write([]byte(`{"artifacts":[{"name":"second","size_in_bytes":2,"expired":false}]}`))
			} else {
				_, _ = w.Write([]byte(`{"artifacts":[]}`))
			}
		case "/repos/o/r/contents/.github/workflows/test.yml":
			_, _ = w.Write([]byte(`{"content":"` + fixtureWorkflowContent + `"}`))
		case "/repos/o/r/actions/jobs/9/logs":
			_, _ = w.Write([]byte("FAILED"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	input, err := (GitHubCollector{BaseURL: api.URL, Client: api.Client()}).Collect("o/r", 7)
	if err != nil {
		t.Fatal(err)
	}
	if input.FailedJob != "test" || len(input.Artifacts) != 2 || input.Artifacts[1].Name != "second" {
		t.Fatalf("input = %#v", input)
	}
}

func TestGitHubCollectorCreatesInputFromReadOnlyEndpoints(t *testing.T) {
	server := httptest.NewServer(githubFixtureHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("missing API auth")
		}
		_, _ = w.Write([]byte("FAILED"))
	}))
	defer server.Close()
	input, err := (GitHubCollector{BaseURL: server.URL, Token: "token", Client: server.Client()}).Collect("o/r", 7)
	if err != nil {
		t.Fatal(err)
	}
	if input.FailedJob != "test" || input.FailedStep != "Test" || input.Log != "FAILED" {
		t.Fatalf("input = %#v", input)
	}
	if input.Artifacts[0].Name != "report" || !strings.Contains(string(input.Workflow), "go test") {
		t.Fatalf("input = %#v", input)
	}
}

func TestValidRepositoryRejectsDotSegments(t *testing.T) {
	for _, repository := range []string{"./repo", "owner/.", "owner/.."} {
		if validRepository(repository) {
			t.Fatalf("accepted dot segment %q", repository)
		}
	}
}

func TestValidLogRedirectRequiresDefaultHTTPSAzureBlobEndpoint(t *testing.T) {
	for _, rawURL := range []string{
		"https://account.blob.core.windows.net/log",
	} {
		if !validLogRedirect(rawURL) {
			t.Fatalf("rejected valid redirect %q", rawURL)
		}
	}
	for _, rawURL := range []string{
		"http://account.blob.core.windows.net/log",
		"https://account.blob.core.windows.net:444/log",
		"https://user:pass@account.blob.core.windows.net/log",
		"https://account.blob.core.windows.net.evil.example/log",
		"/relative",
	} {
		if validLogRedirect(rawURL) {
			t.Fatalf("accepted unsafe redirect %q", rawURL)
		}
	}
}

func TestGitHubCollectorRejectsInvalidRepositoryBeforeRequest(t *testing.T) {
	collector := GitHubCollector{}
	for _, repository := range []string{"owner", "owner/repo/extra", "owner//repo", "owner/repo?x=1", "../repo"} {
		if _, err := collector.Collect(repository, 7); err == nil {
			t.Fatalf("accepted invalid repository %q", repository)
		}
	}
}

func TestGitHubCollectorRejectsNonHTTPSLogRedirect(t *testing.T) {
	api := httptest.NewServer(githubFixtureHandler(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1/metadata", http.StatusFound)
	}))
	defer api.Close()
	_, err := (GitHubCollector{BaseURL: api.URL, Client: api.Client()}).Collect("o/r", 7)
	if err == nil || !strings.Contains(err.Error(), "allowed absolute HTTPS") {
		t.Fatalf("err = %v", err)
	}
}

func TestGitHubCollectorFollowsSignedLogRedirectWithoutForwardingBearerToken(t *testing.T) {
	var storageAuthorization string
	storage := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storageAuthorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("FAILED"))
	}))
	defer storage.Close()
	api := httptest.NewServer(githubFixtureHandler(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://account.blob.core.windows.net/log", http.StatusFound)
	}))
	defer api.Close()
	transport := api.Client().Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test-only TLS server
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if strings.HasPrefix(address, "account.blob.core.windows.net:") {
			return (&net.Dialer{}).DialContext(ctx, network, storage.Listener.Addr().String())
		}
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	client := &http.Client{Transport: transport}
	_, err := (GitHubCollector{BaseURL: api.URL, Token: "token", Client: client}).Collect("o/r", 7)
	if err != nil {
		t.Fatal(err)
	}
	if storageAuthorization != "" {
		t.Fatalf("API bearer token leaked to redirected log storage: %q", storageAuthorization)
	}
}
