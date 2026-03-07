# docs/skill-contract.md

This document defines how skills interact with the repository workflow.

## Principle
Skills may accelerate work, but they are subordinate to repository policy. They do not replace approval gates, current-state understanding, dependency analysis, review discipline, or documentation requirements.

## Mandatory precedence
When a skill conflicts with repository policy:
1. repository policy wins
2. use only the compatible parts of the skill
3. state the deviation explicitly

Example output:
- `Skill check: skipped initial-review - reason: skill requires change hooks, but Initial Review Mode forbids unsolicited change suggestions`

## Approval-gate rule
No change-oriented skill may:
- edit files
- produce diffs
- apply fixes
- run full validation suites
before the exact approval token `Approved vN` has been received for a Non-trivial task.

This applies even if a skill says:
- "implement after approval"
- "apply selected comments"
- "fix CI now"
- or similar generic language

For this repository, the only valid approval form is the exact Scope Ledger token:
- `Approved vN`

## Explanation-only review rule
For explanation-only review requests:
- do not propose changes
- do not provide refactor hooks
- do not provide implementation suggestions unless the user explicitly asks for them

If a review skill includes a mandatory "change hooks" or similar section, suppress that section and state why.

## Environment and tool prerequisites
A skill may be used only if its prerequisites are available in the current environment.

Skip the skill if it requires any unavailable dependency, including:
- privileged sandbox escalation
- external MCP servers
- network tool surfaces not present
- repo connectors not configured
- browser, screenshot, or CI control planes not available in the session

Use this output shape:
- `Skill check: skipped <skill> - reason: unavailable prerequisite <name>`

## Installed vs repo-managed skills
When both exist:
- prefer the repo-managed skill copy for this repository
- keep repo-managed copies aligned with repository workflow
- update repo-managed copies when installed copies are too generic or conflict with the repo's approval model

## Recommended behavior for common skill categories
### Review skills
Allowed:
- code reading
- call-chain analysis
- invariant extraction
- risk identification

Not allowed for explanation-only requests:
- proposed changes
- refactor hooks
- patch plans
- implementation suggestions

### Change-execution skills
Allowed before approval:
- context gathering
- ledger drafting
- requirements discovery
- dependency mapping
- validation planning

Not allowed before approval:
- file changes
- diff generation
- full validation runs

### Tool-heavy skills
Allowed only if tools are available and stable in the session.
If not, skip the skill and continue with direct workflow.

## Repo-managed skill maintenance guidance
Repo-managed skill copies should explicitly say:
- repository policy overrides the skill
- exact `Approved vN` is required for Non-trivial changes
- explanation-only requests must not include change suggestions
- unavailable tools/prerequisites require skipping the skill rather than forcing tool assumptions
