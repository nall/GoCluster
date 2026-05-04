# ADR-0090: Local Telnet Login Call Validation

- Status: Superseded
- Date: 2026-04-30
- Decision Origin: Design

## Context
Local telnet login previously used the shared callsign syntax predicate. That
predicate rejected empty and all-letter short inputs, but it still allowed
numeric-only tokens such as `8300` and `9600` because it required a digit but
did not require a letter. Those strings are not valid amateur callsigns and
should not create persisted local user records or registered telnet sessions.

The cluster already uses CTY and FCC ULS checks in ingest paths. Local human
telnet login should use the same authority sources where they are available,
without making reference-data outages an operator lockout risk.

## Decision
Local telnet login now applies a telnet-specific gate before user state is
loaded or the session is registered:

- the normalized login token must pass the existing callsign syntax predicate;
- the token must contain at least one ASCII letter and one digit;
- skimmer marker suffixes such as `-#` are not accepted for local human login;
- when CTY is loaded, the login call must resolve through portable CTY lookup;
- US calls resolved to ADIF 291 must pass FCC ULS validation when ULS is
  available;
- CTY or FCC ULS unavailability fails open and is logged through a rate-limited
  path;
- local TEST calls with CTY-valid ADIF 291 prefixes bypass FCC ULS validation
  but still require CTY validity.

Local TEST calls use the same shape already used by manual DX posting: no slash
segments, a base ending in `TEST`, and an optional numeric SSID.

## Alternatives considered
1. Strengthen only the regex
   - Simpler, but would still allow syntactically plausible garbage with unknown
     CTY prefixes to create local user state.
2. Require CTY and FCC ULS fail-closed
   - Stronger admission, but a CTY refresh failure or missing FCC database could
     lock out legitimate operators.
3. Add YAML knobs for login validation policy
   - More flexible, but unnecessary for the current operator requirement and
     larger than the approved scope.

## Consequences
### Benefits
- Numeric-only and word-like login tokens are rejected before local user records
  or sessions are created.
- CTY/FCC validation is applied consistently to local human telnet login while
  preserving operator access during reference-data outages.
- TEST calls remain usable for local testing without requiring fake FCC license
  rows.

### Risks
- CTY-loaded deployments can reject syntactically valid but CTY-unknown calls.
- A stale FCC ULS database can reject a newly licensed US operator until the
  next refresh.

### Operational impact
- The visible rejection remains the configured invalid-login message.
- Detailed reject/skip reasons are rate-limited server logs.
- No new YAML settings are introduced.

## Links
- Related issues/PRs/commits: none
- Related tests: `telnet/handshake_transcript_test.go`
- Related docs: `telnet/README.md`
- Related TSRs: none
- Supersedes / superseded by: superseded by ADR-0110
