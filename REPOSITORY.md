# Repository metadata

## Description

Sanitized, local evidence bundles and fail-closed replay-plan analysis for failed GitHub Actions runs.

## Suggested topics

`github-actions`, `ci`, `devops`, `debugging`, `go`, `cli`, `developer-tools`, `security`, `offline-first`

## Initial release checklist

- [ ] Review all source and documentation for real credentials or private run references.
- [ ] Run `go test ./...`, `go vet ./...`, `go build ./cmd/ci-capsule`, and the hardened Docker smoke command in the README/CI evidence.
- [ ] Confirm a clean `ci-capsule audit --path <sample-bundle>` result.
- [ ] Create the GitHub repository only with explicit owner approval.
- [ ] Push only after local review approval.
- [ ] Wait for GitHub Actions to pass.
- [ ] Create a draft release only after a version/tag decision and explicit approval.
