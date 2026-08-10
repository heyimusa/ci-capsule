# ci-capsule

**Package a failed GitHub Actions run into local, sanitized evidence—then show a deterministic replay candidate only when the workflow source proves one.**

`ci-capsule` is a read-only, owner-operated **Linux** CLI. It is not a GitHub runner emulator and it never executes a recovered command.

## Why

A failed CI run usually leaves useful evidence scattered across logs, job metadata, a workflow revision, and artifacts. `ci-capsule` creates a compact local bundle that answers:

- Which job and step failed?
- Which workflow revision and SHA produced it?
- Is there an exact one-line static command associated with that step?
- What does the workflow show when a safe replay candidate cannot be made?
- Which artifacts existed, without downloading their contents?

## Install (development)

```bash
go install github.com/heyimusa/ci-capsule/cmd/ci-capsule@latest
```

For a checkout:

```bash
go build -o ./bin/ci-capsule ./cmd/ci-capsule
```

## Quick start

Collect a public or authorized failed run. `GITHUB_TOKEN` is optional for public data, but improves API rate limits; it is never written into the bundle.

```bash
ci-capsule pull \
  --repo nektos/act \
  --run 31340388304 \
  --out ./act-failure

ci-capsule inspect --bundle ./act-failure
ci-capsule audit --path ./act-failure
```

Example result:

```text
RUN https://github.com/nektos/act/actions/runs/31340388304
SHA ae9eed66a000e0d5ce54ed8ca6a08c373c0aa270
ARTIFACTS 5 (metadata_only)
EVIDENCE logs/failed-job.txt
REPLAY_CANDIDATE go run gotest.tools/gotestsum@latest ...
```

For already-exported local evidence, avoid network access entirely:

```bash
ci-capsule create \
  --workflow .github/workflows/checks.yml \
  --log failed-job.log \
  --job test-linux \
  --step 'Run Tests' \
  --sha <commit-sha> \
  --out ./offline-capsule
```

## What is stored

- workflow source at the failed run SHA;
- run identity, repository, SHA, failed job and failed step;
- potentially sensitive workflow and log evidence, persisted only after bounded pattern sanitization;
- workflow analysis and source line for supported static commands;
- artifact **metadata only**: name, size, expiry, and digest where GitHub supplies it.

Artifact content, cache content, and tokens are not collected. Redaction is best-effort: secret-shaped or platform-masked values are removed, but unknown secret formats and sensitive non-credential text may remain.

## Analysis states

| State | Meaning |
|---|---|
| `REPLAY_CANDIDATE` | A uniquely named step resolves to a one-line literal `run:` command. It is evidence, not an instruction to execute. |
| `SCRIPT_EVIDENCE` | A multiline script is preserved for review without evaluation or execution. |
| `REPLAY_UNAVAILABLE` | Static source does not support a trustworthy command recovery. |

## Security model and limitations

Read [THREAT_MODEL.md](THREAT_MODEL.md) before using the tool for private CI logs.

Important limitations:

- Sanitization reduces exposure but **does not prove a bundle is secret-free**.
- Unknown secret formats, high-entropy values, and sensitive non-credential data may survive pattern redaction.
- GitHub-hosted runner behavior, secrets, OIDC, protected environments, service containers, caches, reusable workflows, and arbitrary GitHub expression semantics are not reproduced.
- The tool rejects duplicate YAML keys, anchors/aliases, merge keys, and multiple documents rather than guessing an interpretation.

## Commands

```text
ci-capsule pull    --repo owner/repo --run RUN_ID --out DIR
ci-capsule create  --workflow FILE --log FILE --job KEY --step NAME --out DIR
ci-capsule inspect --bundle DIR
ci-capsule audit   --path FILE_OR_DIR
```

`pull` uses only GitHub REST `GET` requests. It does not rerun a job, edit a workflow, download artifacts, or upload evidence.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/ci-capsule
```

## License

MIT. See [LICENSE](LICENSE).
