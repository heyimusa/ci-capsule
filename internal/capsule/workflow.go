package capsule

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type Workflow struct{ Root *yaml.Node }
type Replay struct {
	State, Command, Script, Reason string
	Line                           int
}
type EnvironmentValue struct{ State, Value, Axis string }
type Analysis struct {
	Replay      Replay
	Environment map[string]EnvironmentValue
}

// ParseWorkflow accepts exactly one strictly unambiguous YAML document.
func ParseWorkflow(source []byte) (*Workflow, error) {
	decoder := yaml.NewDecoder(strings.NewReader(string(source)))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("parse workflow: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple YAML documents are unsupported")
		}
		return nil, fmt.Errorf("parse trailing workflow content: %w", err)
	}
	if err := validateNode(&document); err != nil {
		return nil, err
	}
	return &Workflow{Root: &document}, nil
}

func validateNode(node *yaml.Node) error {
	if node.Anchor != "" || node.Alias != nil || node.Kind == yaml.AliasNode {
		return fmt.Errorf("anchors and aliases are unsupported at line %d", node.Line)
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Value == "<<" {
				return fmt.Errorf("merge keys are unsupported at line %d", key.Line)
			}
			if seen[key.Value] {
				return fmt.Errorf("duplicate key %q at line %d", key.Value, key.Line)
			}
			seen[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if err := validateNode(child); err != nil {
			return err
		}
	}
	return nil
}

func mapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func jobNode(root *yaml.Node, name string) *yaml.Node {
	if root == nil || len(root.Content) != 1 {
		return nil
	}
	jobs := mapValue(root.Content[0], "jobs")
	return mapValue(jobs, name)
}

func scalar(node *yaml.Node) string {
	if node != nil && node.Kind == yaml.ScalarNode {
		return node.Value
	}
	return ""
}

func hasUnsupportedExecutionContext(root, job, step *yaml.Node) bool {
	for _, node := range []*yaml.Node{root, job, step} {
		if node == nil {
			continue
		}
		if mapValue(node, "env") != nil {
			return true
		}
		if node != root {
			for _, key := range []string{"if", "working-directory", "shell"} {
				if mapValue(node, key) != nil {
					return true
				}
			}
		}
		defaults := mapValue(node, "defaults")
		if defaults != nil && mapValue(defaults, "run") != nil {
			return true
		}
	}
	return false
}

func AnalyzeWorkflow(workflow *Workflow, jobName, stepName string, matrix map[string]string) Analysis {
	result := Analysis{Replay: Replay{State: "unavailable", Reason: "failed step is missing or ambiguous in static workflow YAML"}, Environment: map[string]EnvironmentValue{}}
	job := jobNode(workflow.Root, jobName)
	steps := mapValue(job, "steps")
	if steps == nil || steps.Kind != yaml.SequenceNode {
		return result
	}
	var matched *yaml.Node
	for _, step := range steps.Content {
		if scalar(mapValue(step, "name")) == stepName {
			if matched != nil {
				return result
			}
			matched = step
		}
	}
	if matched == nil {
		return result
	}
	run := mapValue(matched, "run")
	if run == nil || run.Kind != yaml.ScalarNode {
		return result
	}
	if strings.TrimSpace(run.Value) == "" {
		result.Replay.Reason = "workflow run command is empty"
		return result
	}
	if strings.Contains(run.Value, "${{") {
		result.Replay.Reason = "workflow run command contains an unsupported expression"
		return result
	}
	if strings.Contains(run.Value, "\n") {
		result.Replay = Replay{State: "evidence_only", Script: strings.TrimSpace(run.Value), Line: run.Line}
	} else {
		if hasUnsupportedExecutionContext(workflow.Root.Content[0], job, matched) {
			result.Replay.Reason = "step execution context contains unsupported fields or expressions"
			return result
		}
		result.Replay = Replay{State: "candidate", Command: run.Value, Line: run.Line}
	}
	forEnv := mapValue(matched, "env")
	if forEnv == nil || forEnv.Kind != yaml.MappingNode {
		return result
	}
	for i := 0; i+1 < len(forEnv.Content); i += 2 {
		key, value := forEnv.Content[i].Value, scalar(forEnv.Content[i+1])
		switch {
		case strings.HasPrefix(value, "${{ matrix.") && strings.HasSuffix(value, " }}"):
			axis := strings.TrimSuffix(strings.TrimPrefix(value, "${{ matrix."), " }}")
			if found, ok := matrix[axis]; ok {
				result.Environment[key] = EnvironmentValue{State: "resolved_matrix", Value: found}
			} else {
				result.Environment[key] = EnvironmentValue{State: "matrix_value_unavailable", Axis: axis}
			}
		case strings.Contains(value, "${{ secrets."):
			result.Environment[key] = EnvironmentValue{State: "secret_unavailable"}
		case strings.Contains(value, "${{"):
			result.Environment[key] = EnvironmentValue{State: "expression_unavailable"}
		default:
			result.Environment[key] = EnvironmentValue{State: "literal", Value: value}
		}
	}
	return result
}
