# Contributing to ci-capsule

Thanks for improving `ci-capsule`.

## Scope and safety

Keep the project an owner-operated, read-only evidence collector and static
workflow analyzer. Contributions must not execute recovered commands, rerun or
mutate GitHub Actions workflows, collect artifact/cache content, or upload
bundles without a separately reviewed design and threat model.

Use only synthetic data in tests, issues, documentation, and commits. Never
commit a real credential, signed log URL, private CI log, internal hostname,
proprietary workflow, or private bundle.

## Development checks

Run the same baseline checks as CI:

```sh
go test ./...
go vet ./...
go build ./cmd/ci-capsule
```

Changes to collection, sanitization, or workflow analysis need focused tests.
For a release-level behavior change, add a sanitized acceptance fixture that
states its expected output and non-claims.

## Pull requests

Describe the input boundary, output change, error/exit behavior, and any new
limitations. Update [THREAT_MODEL.md](THREAT_MODEL.md) and README language when
a public security or privacy claim changes. Do not describe a successful
sanitization or audit as proof that a bundle is secret-free.

## Reporting problems

Use issue forms for incorrect analysis, unsupported inputs, or redaction/false
assurance concerns. For a security vulnerability, follow
[SECURITY.md](SECURITY.md) instead of opening a public issue.
