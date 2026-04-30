# ADR-0089: SET SOLAR Help Routing

- Status: Accepted
- Date: 2026-04-30
- Decision Origin: Design

## Context
`SET SOLAR` was implemented and had command-specific help, but it was not listed
in the top-level `HELP` command order for either supported dialect. The support
knowledge base also lacked a direct route for solar-summary questions.

## Decision
Add `SET SOLAR` to the top-level `HELP` order for both `go` and `cc` dialects,
keep the README generated HELP block synchronized, and route custom GPT solar
summary questions to existing operator and command docs.

No durable command semantics changed.

## Alternatives considered
1. Leave `SET SOLAR` discoverable only through `HELP SET SOLAR`.
2. Add only custom GPT routing and leave runtime `HELP` unchanged.
3. Duplicate solar-summary behavior inside the custom GPT files.

## Consequences
### Benefits
- Top-level `HELP` now exposes an implemented operator command.
- README and runtime HELP stay aligned through the existing sync test.
- The custom GPT can route solar-summary questions without owning duplicate
  behavior text.

### Risks
- Future command additions still need the same runtime HELP, README, and support
  routing review.

### Operational impact
- Operators see one additional line in top-level `HELP`.
- `SET SOLAR` syntax and runtime behavior are unchanged.

## Links
- Related issues/PRs/commits: local change
- Related tests: `go test ./commands -run TestReadmeDefaultGoHelpBlockMatchesProcessor -count=1`
- Related docs: `README.md`, `customgpt/source-map.md`, `customgpt/common-questions.md`, `customgpt/operator-guide-index.md`
- Related TSRs: none
- Supersedes / superseded by: none
