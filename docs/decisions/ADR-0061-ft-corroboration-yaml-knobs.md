# ADR-0061 - YAML-Driven FT Corroboration Timing and Threshold Knobs

Status: Accepted
Date: 2026-04-08
Decision Makers: Cluster maintainers
Technical Area: config, main output pipeline, FT confidence, docs
Decision Origin: Operator tuning request
Troubleshooting Record(s): -
Tags: ft8, ft4, ft2, confidence, config, yaml

## Context

- ADR-0060 established the live FT corroboration model: bounded arrival-burst clustering keyed by DX call, exact FT mode, and canonical dial frequency.
- ADR-0060 intentionally left FT burst timing code-bound while live tuning settled.
- Operators now need to tune FT `P`/`V` thresholds and per-mode hold durations without rebuilding the binary.
- The runtime already treats FT corroboration as separate from resolver mutation, even though both live under `call_correction` in the YAML surface.

## Decision

- Add explicit FT corroboration YAML knobs under `call_correction`:
  - `p_min_unique_spotters`
  - `v_min_unique_spotters`
  - `ft8_quiet_gap_seconds`
  - `ft8_hard_cap_seconds`
  - `ft4_quiet_gap_seconds`
  - `ft4_hard_cap_seconds`
  - `ft2_quiet_gap_seconds`
  - `ft2_hard_cap_seconds`
- Preserve ADR-0060 behavior when those keys are omitted by normalizing to the existing defaults:
  - `p_min_unique_spotters=2`
  - `v_min_unique_spotters=3`
  - `FT8 6s/12s`
  - `FT4 5s/10s`
  - `FT2 3s/6s`
- Validate at load time and refuse startup when FT corroboration settings are structurally unsafe:
  - `p_min_unique_spotters >= 2`
  - `v_min_unique_spotters > p_min_unique_spotters`
  - every quiet gap is `> 0`
  - every hard cap is `>=` its corresponding quiet gap
- Feed the normalized FT settings into the FT burst controller at output-pipeline construction time.
- Keep FT corroboration independent of `call_correction.enabled`; that switch still controls resolver mutation, not FT burst tagging.

## Alternatives Considered

1. Leave FT thresholds and hold durations code-bound.
   - Rejected because it forces rebuild/redeploy for live tuning of an operator-facing policy.
2. Add a separate top-level `ft_corroboration` config section.
   - Rejected for now because the existing operator workflow already expects confidence-related knobs under `call_correction`.
3. Support live reload for FT burst knobs.
   - Rejected in this revision because it would add runtime mutation and observability complexity to the single-owner pipeline.

## Consequences

### Benefits
- Operators can tune FT `P`/`V` behavior and hold duration without code changes.
- Invalid FT corroboration settings now fail fast at config load instead of degrading silently at runtime.
- Startup diagnostics can print the active FT corroboration policy alongside the rest of the `call_correction` summary.

### Risks
- Bad but structurally valid values can still reduce FT promotion quality or over-merge bursts.
- Placing FT knobs under `call_correction` can imply they are gated by `call_correction.enabled` unless the docs stay explicit.

### Operational impact
- Default behavior is unchanged when the new keys are omitted.
- Startup now fails fast on invalid FT corroboration bounds instead of silently normalizing them.

## Validation

- Added/updated tests:
  - `config/call_correction_stabilizer_test.go`
  - `ft_confidence_runtime_test.go`
- Commands:
  - `go test ./config . -run "TestLoadCallCorrection|TestOutputPipelineFT|TestFTConfidenceController|TestBuildFTConfidence"`
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `go test -race ./...`

## Rollout and Reversal

- Rollout plan:
  - Deploy with no YAML changes first; defaults preserve the existing FT policy.
  - Tune YAML only after observing live FT burst stats and output glyph mix.
- Backward compatibility impact:
  - Existing configs continue to load and keep the previous FT defaults.
- Reversal plan:
  - Remove the new YAML fields and restore code-bound defaults if the operator surface proves too error-prone.

## References

- Related ADR(s):
  - `docs/decisions/ADR-0060-source-aware-ft-burst-clustering.md`
- Docs:
  - `README.md`
  - `data/config/pipeline.yaml`
