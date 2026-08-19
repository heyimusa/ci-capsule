# Security policy

## Supported versions

Security fixes are applied to the latest released version of `ci-capsule`.

## Reporting a vulnerability

Please **do not** open a public issue for a suspected vulnerability.

Use GitHub's private security-advisory reporting for this repository. Include a
concise description, affected version or commit, impact, and a synthetic
reproduction when possible. Do not include real credentials, signed log URLs,
private CI logs, internal hostnames, proprietary workflow content, or bundle
contents that may contain sensitive information.

Reports receive an acknowledgement within seven days when contact details are
provided.

## Security boundary

`ci-capsule` is a Linux, owner-operated, read-only CLI. It collects local
sanitized evidence and performs static workflow analysis. It never executes a
recovered command, runs a workflow, downloads artifact content, uploads a
bundle, or forwards a GitHub API token to redirected log-storage hosts.

A missed redaction, potentially misleading security claim, or incorrect
analysis result may be important without necessarily being a vulnerability. A
public report must use a fully synthetic, minimized reproduction only. If a
report involves a real bundle, log, workflow, URL, hostname, credential, or any
uncertainty about whether redaction succeeded, use the private advisory channel
instead.
