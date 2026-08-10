# v0.1.0 acceptance fixture

This fixture is deliberately synthetic. It contains no real token, customer URL, internal hostname, or production identifier.

## Positive control

Build and run the source in a hardened container:

```sh
docker build -t local/ci-capsule:v0.1.0 .
mkdir -p /tmp/ci-capsule-acceptance
chmod 777 /tmp/ci-capsule-acceptance
docker run --rm --read-only --network none --cap-drop ALL \
  --security-opt no-new-privileges --user 65532:65532 \
  -v "$PWD/acceptance/v0.1.0/workflow.yml:/input/workflow.yml:ro" \
  -v "$PWD/acceptance/v0.1.0/failed.log:/input/failed.log:ro" \
  -v /tmp/ci-capsule-acceptance:/output \
  local/ci-capsule:v0.1.0 create --workflow /input/workflow.yml \
  --log /input/failed.log --job test --step Test --out /output/bundle
```

The resulting `bundle.json` must have `analysis.replay.state` equal to `candidate`, and the placeholder `ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ123456` must not be present in the output.

## Negative control

Replace `workflow.yml` with a duplicate YAML key. The command must exit non-zero and no `bundle.json` should be created.

## Non-claims

A matching local fixture proves only this source path and fixture behavior. It does not prove a GitHub workflow ran, a candidate command will reproduce a GitHub-hosted runner, an artifact is benign, or a deployment/rollback works.
