package capsule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectBundleSanitizesLegacyBundleOnRender(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bundle.json"), []byte(`{"source":{"run_url":"https://x/ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ123456","sha":"s"},"artifacts":{"policy":"metadata_only"},"log_path":"l","analysis":{"replay":{"state":"candidate","command":"echo ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ123456"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := InspectBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(report, "ghp_") {
		t.Fatalf("inspect leaked credential-shaped data: %q", report)
	}
}

func TestCreateBundleWritesSanitizedEvidenceAndInspection(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bundle")
	input := Input{
		Source:    Source{Repository: "owner/repo", RunURL: "https://github.com/owner/repo/actions/runs/7", SHA: "ae9eed66a000e0d5ce54ed8ca6a08c373c0aa270", WorkflowPath: ".github/workflows/test.yml"},
		Workflow:  []byte("jobs:\n  test:\n    steps:\n      - name: Test\n        run: go test ./...\n"),
		FailedJob: "test", FailedStep: "Test", Log: "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9.signature01234567890123456789\nFAILED",
		Artifacts: []Artifact{{Name: "report", SizeBytes: 42, Digest: "sha256:abc", Expired: false}},
	}
	bundle, err := CreateBundle(dir, input, nil)
	if err != nil {
		t.Fatal(err)
	}
	logBytes, err := os.ReadFile(filepath.Join(dir, "logs", "failed-job.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logBytes), "eyJhbGciOiJIUzI1NiJ9") {
		t.Fatal("token leaked into persisted log")
	}
	if bundle.Analysis.Replay.State != "candidate" || bundle.Artifacts.Policy != "metadata_only" {
		t.Fatalf("bundle = %#v", bundle)
	}
	inspection, err := InspectBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(inspection, "REPLAY_CANDIDATE go test ./...") || !strings.Contains(inspection, "ARTIFACTS 1 (metadata_only)") {
		t.Fatalf("inspection = %q", inspection)
	}
}

func TestCreateBundleSanitizesPersistedWorkflowEvidence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bundle")
	_, err := CreateBundle(dir, Input{
		Workflow:  []byte("jobs:\n  test:\n    steps:\n      - name: Test\n        run: go test ./...\n        env:\n          TOKEN: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890\n"),
		FailedJob: "test", FailedStep: "Test",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(filepath.Join(dir, "workflow.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(workflow), "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890") {
		t.Fatal("token leaked into persisted workflow")
	}
}

func TestCreateBundleSanitizesAnalysisAndArtifactMetadata(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bundle")
	_, err := CreateBundle(dir, Input{
		Source:    Source{Repository: "owner/ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"},
		Workflow:  []byte("jobs:\n  test:\n    steps:\n      - name: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890\n        run: echo ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890\n        env:\n          NAME: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890\n"),
		FailedJob: "test", FailedStep: "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890", Artifacts: []Artifact{{Name: "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(filepath.Join(dir, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bundle), "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890") {
		t.Fatal("token leaked into bundle JSON")
	}
	inspection, err := InspectBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(inspection, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890") {
		t.Fatal("token leaked through inspect")
	}
}

func TestCreateBundleRejectsExistingOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := CreateBundle(dir, Input{Workflow: []byte("jobs: {}\n")}, nil)
	if err == nil {
		t.Fatal("CreateBundle accepted pre-existing output directory")
	}
}

func TestCreateBundleRejectsSymlinkOutputDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	_, err := CreateBundle(link, Input{Workflow: []byte("jobs: {}\n")}, nil)
	if err == nil {
		t.Fatal("CreateBundle accepted symlink output directory")
	}
}

func TestCreateBundleRejectsSymlinkedParentComponent(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "parent")
	if err := os.Symlink(target, parent); err != nil {
		t.Fatal(err)
	}
	_, err := CreateBundle(filepath.Join(parent, "bundle"), Input{Workflow: []byte("jobs: {}\n")}, nil)
	if err == nil {
		t.Fatal("CreateBundle accepted symlinked parent component")
	}
}

func TestCreateBundleRejectsUnsafeWorkflow(t *testing.T) {
	_, err := CreateBundle(t.TempDir(), Input{Workflow: []byte("jobs:\n  a: {}\n  a: {}\n")}, nil)
	if err == nil {
		t.Fatal("CreateBundle accepted duplicate keys")
	}
}
