# ADR-0076: GitHub Release Package

- Status: Accepted
- Date: 2026-04-24
- Decision Origin: Design

## Context

Operators need a simple deployment artifact that does not require manually
copying individual files from a development checkout. The live server expects a
binary plus a `data/config` directory, while the local `data` tree also contains
runtime state, Pebble stores, logs, generated downloads, user files, and optional
secret-bearing tool config. Zipping the live working directory would risk
publishing state that should remain local.

The checked-in `data/config` tree is itself published by normal GitHub pushes.
Therefore it cannot contain private operational settings such as real peer
callsigns, peer hostnames or IP addresses, passwords, or service tokens.

The repo also needs to keep the existing development shape: `go build .` from
the repo root remains the live binary build path, and advanced users can keep
cloning, editing, and building the source tree directly.

## Decision

Publish a Windows amd64 GitHub Release zip containing a top-level
`gocluster-windows-amd64/` directory with:

1. `gocluster.exe` built from the repo root.
2. A curated `data/` tree copied from source-controlled runtime inputs,
   including public example config.
3. A deliberately small operator documentation bundle: a zip-root `README.md`,
   `docs/OPERATOR_GUIDE.md`, and the config-local docs already included under
   `data/config`.
4. No live Pebble stores, logs, user state, generated local caches, or optional
   secret-bearing files such as `data/config/openai.yaml`.

The package is built by an explicit allowlist script instead of copying the
developer's live `data` directory. The first supported release target is Windows
amd64 only.

The repository's tracked `data/config` is public example config. Private
operational config lives in a complete ignored directory such as
`data/config.local/` and is selected with `DXC_CONFIG_PATH`. The release script
also validates the staged public config for example callsigns, `.example.invalid`
peer hosts, blank peering passwords, and placeholder/disabled IPinfo settings
before creating the zip.

The zip-root `README.md` is rendered during packaging from the staged
`data/config` directory for runtime-configured values such as `telnet.port`.
This prevents the release instructions from drifting away from the packaged
YAML.

## Alternatives considered

1. Zip the whole repo or whole `data` directory.
2. Check a deployable `dist/` folder into the repository.
3. Publish only a GitHub Actions artifact instead of a GitHub Release asset.

## Consequences

### Benefits

- Operators can deploy by extracting one zip and running `gocluster.exe`.
- Release contents are reproducible from committed inputs.
- Runtime state and secrets are excluded by construction.
- Source developers keep the current repo layout and build workflow.
- Normal GitHub commits and release packages use the same public config policy,
  so the release path is not the only privacy boundary.
- Operators get enough local documentation to configure and launch the package
  without bundling the full repository docs tree.

### Risks

- The allowlist can omit a future required runtime input if new data
  dependencies are added without updating release packaging.
- Windows amd64 only is intentionally narrow; other platforms need a later
  decision and validation pass.
- Manual workflow dispatch depends on using an existing release tag.
- The bundled docs are intentionally incomplete; operators need GitHub for
  detailed architecture, package-local internals, ADRs, and troubleshooting
  history.

### Operational impact

- Binary and packaged config should be deployed and rolled back together.
- The release package does not include learned runtime Pebble data; stores and
  downloads are created or refreshed by runtime behavior according to YAML.
- Operators should copy the packaged public config to a private config
  directory, edit it, and run with `DXC_CONFIG_PATH` for real deployments.
- The zip-root `README.md` is the package start point. It is rendered from
  packaged YAML where it names runtime-configured values, and it points to the
  small bundled operator guide and to GitHub for the full documentation set.
- Already-published private values must be treated as exposed outside this ADR;
  rotate passwords and service tokens as needed, and handle any Git history
  rewrite as a separate repository operation.

## Links

- Related issues/PRs/commits:
- Related tests: `go test ./config`, `scripts/build-release-package.ps1 -Version test-release -OutputDir .tmp\release-validation`, `go test ./...`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run ./... --config=.golangci.yaml`
- Related docs: `README.md`, `.github/workflows/release.yml`, `scripts/build-release-package.ps1`
- Related TSRs:
- Supersedes / superseded by:
