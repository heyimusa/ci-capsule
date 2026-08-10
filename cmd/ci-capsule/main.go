package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/heyimusa/ci-capsule/internal/capsule"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}

func run(args []string) error {
	if len(args) == 1 && args[0] == "--version" {
		fmt.Println(version)
		return nil
	}
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "create":
		return create(args[1:])
	case "pull":
		return pull(args[1:])
	case "inspect":
		return inspect(args[1:])
	case "audit":
		return audit(args[1:])
	default:
		return usage()
	}
}

func usage() error { return errors.New("usage: ci-capsule <pull|create|inspect|audit> [flags]") }

func create(args []string) error {
	flags := flag.NewFlagSet("create", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	workflowPath := flags.String("workflow", "", "local workflow YAML path")
	logPath := flags.String("log", "", "local failed-job log path")
	job := flags.String("job", "", "static workflow job key")
	step := flags.String("step", "", "failed step name")
	out := flags.String("out", "", "output bundle directory")
	repository := flags.String("repo", "", "repository identity for metadata")
	runURL := flags.String("run-url", "", "run URL for metadata")
	sha := flags.String("sha", "", "commit SHA for metadata")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *workflowPath == "" || *logPath == "" || *job == "" || *step == "" || *out == "" {
		return errors.New("create requires --workflow, --log, --job, --step, and --out")
	}
	workflow, err := os.ReadFile(filepath.Clean(*workflowPath))
	if err != nil {
		return err
	}
	log, err := capsule.ReadLocalLog(filepath.Clean(*logPath))
	if err != nil {
		return err
	}
	_, err = capsule.CreateBundle(filepath.Clean(*out), capsule.Input{Source: capsule.Source{Repository: *repository, RunURL: *runURL, SHA: *sha, WorkflowPath: *workflowPath}, Workflow: workflow, FailedJob: *job, FailedStep: *step, Log: string(log)}, nil)
	return err
}

func pull(args []string) error {
	flags := flag.NewFlagSet("pull", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	repository := flags.String("repo", "", "GitHub repository owner/repo")
	runID := flags.Int64("run", 0, "GitHub Actions run ID")
	out := flags.String("out", "", "output bundle directory")
	token := flags.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token; defaults to GITHUB_TOKEN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *repository == "" || *runID <= 0 || *out == "" {
		return errors.New("pull requires --repo, --run, and --out")
	}
	input, err := (capsule.GitHubCollector{Token: *token}).Collect(*repository, *runID)
	if err != nil {
		return err
	}
	_, err = capsule.CreateBundle(filepath.Clean(*out), input, nil)
	return err
}
func inspect(args []string) error {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	bundle := flags.String("bundle", "", "bundle directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *bundle == "" {
		return errors.New("inspect requires --bundle")
	}
	output, err := capsule.InspectBundle(filepath.Clean(*bundle))
	if err != nil {
		return err
	}
	_, err = fmt.Print(output)
	return err
}

func audit(args []string) error {
	flags := flag.NewFlagSet("audit", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	path := flags.String("path", "", "file or directory to scan for supported credential patterns")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("audit requires --path")
	}
	findings, err := capsule.AuditPath(filepath.Clean(*path))
	if err != nil {
		return err
	}
	for _, finding := range findings {
		fmt.Println(finding)
	}
	if len(findings) > 0 {
		return errors.New("supported credential patterns found")
	}
	fmt.Println("AUDIT_OK no supported credential patterns found")
	return nil
}
