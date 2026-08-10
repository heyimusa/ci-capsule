package capsule

import (
	"strings"
	"testing"
)

func TestParseWorkflowRejectsDuplicateKeys(t *testing.T) {
	_, err := ParseWorkflow([]byte("jobs:\n  test: {}\n  test: {}\n"))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err = %v, want duplicate-key error", err)
	}
}

func TestParseWorkflowRejectsAnchorsAliasesAndMultiDocument(t *testing.T) {
	for _, source := range []string{
		"base: &base {runs-on: ubuntu-latest}\njobs: {test: *base}\n",
		"jobs: {}\n---\njobs: {}\n",
		"jobs: {}\n---\ninvalid: [\n",
	} {
		if _, err := ParseWorkflow([]byte(source)); err == nil {
			t.Fatalf("ParseWorkflow accepted unsafe input: %q", source)
		}
	}
}

func TestAnalyzeWorkflowReportsLiteralCommandAndSourceLine(t *testing.T) {
	workflow, err := ParseWorkflow([]byte("jobs:\n  test:\n    steps:\n      - name: Run Tests\n        run: go test ./...\n"))
	if err != nil {
		t.Fatal(err)
	}
	report := AnalyzeWorkflow(workflow, "test", "Run Tests", nil)
	if report.Replay.State != "candidate" || report.Replay.Command != "go test ./..." || report.Replay.Line != 5 {
		t.Fatalf("unexpected replay: %#v", report.Replay)
	}
}

func TestAnalyzeWorkflowRefusesEmptyCommand(t *testing.T) {
	workflow, err := ParseWorkflow([]byte("jobs:\n  test:\n    steps:\n      - name: Test\n        run: '   '\n"))
	if err != nil {
		t.Fatal(err)
	}
	if report := AnalyzeWorkflow(workflow, "test", "Test", nil); report.Replay.State != "unavailable" {
		t.Fatalf("replay = %#v", report.Replay)
	}
}

func TestAnalyzeWorkflowRefusesSecretDependentStepCandidate(t *testing.T) {
	workflow, err := ParseWorkflow([]byte("jobs:\n  test:\n    steps:\n      - name: Test\n        run: go test ./...\n        env:\n          TOKEN: ${{ secrets.TOKEN }}\n"))
	if err != nil {
		t.Fatal(err)
	}
	report := AnalyzeWorkflow(workflow, "test", "Test", nil)
	if report.Replay.State != "unavailable" {
		t.Fatalf("replay = %#v", report.Replay)
	}
}

func TestAnalyzeWorkflowRefusesInheritedExecutionContext(t *testing.T) {
	for _, source := range []string{
		"env:\n  TOKEN: ${{ secrets.TOKEN }}\njobs:\n  test:\n    steps:\n      - name: Test\n        run: go test ./...\n",
		"jobs:\n  test:\n    env:\n      MODE: fast\n    steps:\n      - name: Test\n        run: go test ./...\n",
		"defaults:\n  run:\n    shell: bash\njobs:\n  test:\n    steps:\n      - name: Test\n        run: go test ./...\n",
	} {
		workflow, err := ParseWorkflow([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if report := AnalyzeWorkflow(workflow, "test", "Test", nil); report.Replay.State != "unavailable" {
			t.Fatalf("replay = %#v for %q", report.Replay, source)
		}
	}
}

func TestAnalyzeWorkflowClassifiesMultilineAndSecretEnvironment(t *testing.T) {
	workflow, err := ParseWorkflow([]byte("jobs:\n  test:\n    steps:\n      - name: Test\n        run: |\n          make test\n        env:\n          DRIVER: ${{ matrix.driver }}\n          TOKEN: ${{ secrets.TOKEN }}\n"))
	if err != nil {
		t.Fatal(err)
	}
	report := AnalyzeWorkflow(workflow, "test", "Test", map[string]string{"driver": "docker"})
	if report.Replay.State != "evidence_only" || report.Replay.Script != "make test" {
		t.Fatalf("replay = %#v", report.Replay)
	}
	if report.Environment["DRIVER"].State != "resolved_matrix" || report.Environment["TOKEN"].State != "secret_unavailable" {
		t.Fatalf("env = %#v", report.Environment)
	}
}
