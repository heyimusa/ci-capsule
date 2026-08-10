package capsule

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const BundleVersion = "0.1.0-dev"

type Source struct{ Repository, RunURL, SHA, WorkflowPath string }
type Artifact struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_in_bytes"`
	Digest    string `json:"digest,omitempty"`
	Expired   bool   `json:"expired"`
}
type ArtifactInventory struct {
	Policy string     `json:"policy"`
	Items  []Artifact `json:"items"`
}
type Input struct {
	Source                     Source
	Workflow                   []byte
	FailedJob, FailedStep, Log string
	Matrix                     map[string]string
	Artifacts                  []Artifact
}
type Bundle struct {
	Version    string            `json:"version"`
	CreatedAt  time.Time         `json:"created_at"`
	Source     Source            `json:"source"`
	FailedJob  string            `json:"failed_job"`
	FailedStep string            `json:"failed_step"`
	Analysis   Analysis          `json:"analysis"`
	Artifacts  ArtifactInventory `json:"artifacts"`
	LogPath    string            `json:"log_path"`
}

// CreateBundle creates an offline, metadata-only evidence bundle. It never executes workflow commands.
func CreateBundle(directory string, input Input, masks []string) (Bundle, error) {
	workflow, err := ParseWorkflow(input.Workflow)
	if err != nil {
		return Bundle{}, err
	}
	output, err := openBundleDirectory(directory)
	if err != nil {
		return Bundle{}, err
	}
	defer output.close()
	if err := output.mkdir("logs"); err != nil {
		return Bundle{}, err
	}
	if err := output.writeLog([]byte(SanitizeText(input.Log, masks))); err != nil {
		return Bundle{}, err
	}
	sanitizedWorkflow := SanitizeText(string(input.Workflow), masks)
	if err := output.writeFile("workflow.yml", []byte(sanitizedWorkflow)); err != nil {
		return Bundle{}, err
	}
	bundle := Bundle{Version: BundleVersion, CreatedAt: time.Now().UTC(), Source: input.Source, FailedJob: input.FailedJob, FailedStep: input.FailedStep, Analysis: AnalyzeWorkflow(workflow, input.FailedJob, input.FailedStep, input.Matrix), Artifacts: ArtifactInventory{Policy: "metadata_only", Items: input.Artifacts}, LogPath: "logs/failed-job.txt"}
	encoded, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return Bundle{}, err
	}
	encoded = []byte(SanitizeText(string(encoded), masks))
	if err := json.Unmarshal(encoded, &bundle); err != nil {
		return Bundle{}, fmt.Errorf("reparse sanitized bundle: %w", err)
	}
	if err := output.writeFile("bundle.json", append(encoded, '\n')); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

func InspectBundle(directory string) (string, error) {
	encoded, err := os.ReadFile(filepath.Join(directory, "bundle.json"))
	if err != nil {
		return "", err
	}
	var bundle Bundle
	if err := json.Unmarshal(encoded, &bundle); err != nil {
		return "", fmt.Errorf("parse bundle: %w", err)
	}
	lines := []string{fmt.Sprintf("RUN %s", bundle.Source.RunURL), fmt.Sprintf("SHA %s", bundle.Source.SHA), fmt.Sprintf("ARTIFACTS %d (%s)", len(bundle.Artifacts.Items), bundle.Artifacts.Policy), fmt.Sprintf("EVIDENCE %s", bundle.LogPath)}
	switch bundle.Analysis.Replay.State {
	case "candidate":
		lines = append(lines, "REPLAY_CANDIDATE "+bundle.Analysis.Replay.Command)
	case "evidence_only":
		lines = append(lines, "SCRIPT_EVIDENCE "+bundle.Analysis.Replay.Script)
	default:
		lines = append(lines, "REPLAY_UNAVAILABLE "+bundle.Analysis.Replay.Reason)
	}
	return SanitizeText(strings.Join(lines, "\n")+"\n", nil), nil
}
