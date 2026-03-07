# ADR-0051 - Mode Inference Provenance and Band Default Policy

Status: Superseded
Date: 2026-03-06
Decision Makers: Maintainers
Technical Area: spot, rbn, pskreporter, commands, docs
Decision Origin: Design
Troubleshooting Record(s): none
Tags: mode-inference, ingest-contract, provenance, defaults

Superseded by: ADR-0052

## Context
- The runtime mode inference path previously allowed RBN/RBN-Digital spots with reports but no explicit mode to reach the shared output pipeline.
- The final fallback used a coarse within-band allocation heuristic. On low bands, especially 160m, that heuristic can manufacture false certainty.
- The DX+frequency cache reused any final mode value, including fallback guesses, which let defaulted labels contaminate later ambiguous spots.
- Operators still want a final emitted mode label rather than an internal unknown state, but that label must be distinguished from trusted evidence.

## Decision
- Treat RBN and RBN-Digital as explicit-mode skimmer feeds. If a skimmer spot arrives without an explicit mode token, drop it at ingest. This aligns the RBN contract with PSKReporter’s existing explicit-mode requirement.
- Keep the existing bounded recent-evidence DX+integer-kHz cache as the reuse mechanism for step 2. The contract is “recent trusted mode evidence for the same DX+frequency bucket,” not exact same-spot identity.
- Keep the learned/seeded digital-frequency map for trusted `FT4`/`FT8`/`JS8` inference.
- Replace the old within-band allocation fallback with a final band-policy default table driven by `data/config/mode_allocations.yaml`.
- Separate `Spot.Mode` from `Spot.ModeProvenance`.
- Allow only trusted evidence classes (`source_explicit`, `comment_explicit`, `recent_evidence`, `digital_frequency`) to seed or refresh reusable inference state.
- Mark band-policy defaults as `band_default` provenance and do not feed them back into reusable caches.

## Alternatives Considered
1. Keep the old allocation-based fallback and continue caching final outputs
   - Pros:
   - Minimal code churn.
   - Preserves current behavior for all ambiguous spots.
   - Cons:
   - Continues to overstate certainty on low bands.
   - Lets fallback guesses contaminate later lookups.
2. Add a stricter cross-source “same spot” matcher with time/frequency identity
   - Pros:
   - More precise semantics for cross-source reuse.
   - Stronger claim that reused evidence matches one transmission.
   - Cons:
   - More state, more matching logic, and higher review surface.
   - Did not provide enough value compared with the existing bounded DX+frequency cache.
3. Keep internal `UNKNOWN` and map to phone/CW only in display layers
   - Pros:
   - Most semantically pure distinction between truth and presentation.
   - Prevents default labels from affecting all downstream consumers.
   - Cons:
   - Requires wider changes across filters, formatting, and downstream mode consumers.
   - Does not satisfy the current product requirement for a final emitted mode state.

## Consequences
- Positive outcomes:
  - Shared ingest contracts are consistent across skimmer sources.
  - Band-policy defaults are explicit and operator-readable.
  - Default labels no longer seed later inference.
- Negative outcomes / risks:
  - Some RBN/RBN-Digital spots that previously passed will now be dropped.
  - Ambiguous human spots will default by band policy, which remains a product decision rather than proof.
- Operational impact:
  - Mode-cache observability now distinguishes defaulted outputs from trusted evidence reuse.
  - Operators can tune the final default table in `data/config/mode_allocations.yaml`.
- Follow-up work required:
  - Keep README/config docs aligned with the new provenance semantics.

## Validation
- Tests/benchmarks/analysis that justify this decision.
  - `go test ./spot ./rbn ./peer ./commands .`
  - Added regression coverage for:
    - mode-less RBN spot rejection,
    - minimal-parser human pass-through,
    - band-policy defaults,
    - cache non-seeding for defaults,
    - cache provenance on recent evidence reuse.
- What evidence would invalidate this decision later?
  - If operators need exact cross-source same-spot identity instead of the current DX+bucket cache semantics.
  - If downstream consumers need a true `UNKNOWN` state instead of a final emitted default label.

## Rollout and Reversal
- Rollout plan:
  - Deploy with updated docs and config defaults.
  - Verify that mode-less RBN drops are expected and that default counts rise only on human/no-mode traffic.
- Backward compatibility impact:
  - Behavior changes for mode-less RBN/RBN-Digital spots and for ambiguous low-band human spots.
- Reversal plan:
  - Revert to the prior fallback path or restore fallback cache seeding if operator feedback shows unacceptable regressions.

## References
- Issue(s):
  - none
- PR(s):
  - none
- Commit(s):
  - pending
- Related ADR(s):
  - none
- Troubleshooting Record(s):
  - none
- Docs:
  - `README.md`
  - `data/config/README.md`
  - `data/config/mode_allocations.yaml`
