# ADR-0097: YAML Ownership Boundary Documentation

Status: Accepted
Date: 2026-05-01
Decision Origin: Design

## Context

Operator-facing config docs explained strict YAML loading and listed config
files, but they did not consistently distinguish normal deployment knobs from
algorithm calibration exposed in YAML for inspectability.

## Decision

No durable decision change.

This change documents the existing boundary: deployment/runtime settings,
operator policy settings, reference tables, and algorithm calibration are
different kinds of YAML-owned surfaces. It updates operator and support routing
docs and adds comments to high-risk YAML files without changing values, schema,
loader behavior, defaults, runtime behavior, or algorithms.

## Alternatives considered

- Leave the boundary implicit in package READMEs and older ADRs.
- Move algorithm calibration out of YAML.
- Add loader enforcement for ownership classes.

## Consequences

- Operators get a clearer warning before retuning path reliability, call
  correction, mode inference, or solar override calibration.
- Support routing can point users to the ownership class before suggesting YAML
  edits.
- Runtime behavior and existing config directories remain unchanged.

## Links

- `data/config/README.md`
- `README.md`
- `docs/OPERATOR_GUIDE.md`
- `spot/README.md`
- `pathreliability/README.md`
- `customgpt/`
