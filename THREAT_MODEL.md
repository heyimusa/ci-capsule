# Threat model

## Scope

`ci-capsule` is an owner-operated diagnostic CLI for a GitHub Actions run the operator is authorized to inspect. It collects a minimal, local evidence bundle and performs static workflow analysis. It is not a CI runner, scanner, remote service, or upload client.

## Assets to protect

- GitHub API tokens supplied through `GITHUB_TOKEN` or `--token`.
- Secret values and sensitive data that might appear in CI logs.
- Signed log-storage URLs returned by GitHub.
- Local filesystem contents outside the output directory.
- Artifact, cache, and private-run contents.

## Trust boundaries

### Operator-supplied values

Repository, run ID, paths, local workflow YAML, log files, and output directories are supplied by the operator. The tool treats workflow content as data and never executes its `run:` content.

### GitHub API

`pull` makes REST `GET` requests only. A token is attached only to GitHub API requests. When GitHub redirects a log request to signed Azure Blob Storage, the collector accepts only an absolute HTTPS `*.blob.core.windows.net` URL, follows it anonymously, and never forwards the API bearer token to the storage host.

### Workflow YAML

Workflow YAML can be complex and ambiguous. The parser rejects duplicate mapping keys, anchors, aliases, merge keys, and multiple documents. Unsupported expression semantics are reported as unavailable, not evaluated.

### Log data

Logs are untrusted, may be attacker-controlled, and may contain sensitive content. Before persistence, the tool applies explicit masks supplied to its API and bounded pattern redaction for common credential shapes. This is mitigation, not proof of absence.

## Deliberate non-goals

`ci-capsule` does not:

- execute shell, workflow commands, Docker, or action code;
- use a kubeconfig, cloud credential, SSH credential, or runner registration;
- rerun, cancel, edit, dispatch, or otherwise mutate a GitHub Actions workflow;
- download artifact content or cache content;
- upload, publish, or transmit bundles to a third party;
- claim exact GitHub-hosted runner reproduction;
- resolve arbitrary GitHub expressions, reusable workflows, composite actions, or secrets.

## Redaction limitations

Pattern matching cannot detect every credential format. A bundle can retain unknown token formats, short tokens, encoded/encrypted data, high-entropy strings, or sensitive non-credential content. Operators must inspect and control access to bundles, run `ci-capsule audit`, and treat a successful audit as a check for supported indicators—not a security guarantee.

## Output safety

Bundles are created with owner-only file modes where supported by the host. Artifact inventory stores metadata only, and omits signed download URLs. `audit` reports only a file path and indicator class; it does not print the matched value.

## Reporting a vulnerability

Use GitHub's private security-advisory reporting for this repository. Do not
include a real credential in a report; provide a synthetic reproduction instead.
