# GoCluster Custom GPT Knowledge Base

This folder is a thin routing layer for a custom GPT that supports GoCluster
operators, telnet users, and Go developers. It should point to existing
authoritative documentation instead of restating the same knowledge in another
maintained copy.

## Audience

- Operators running a GoCluster node.
- Telnet users connecting to a GoCluster node.
- Go developers who want to understand, debug, or extend the cluster.

## Source Of Truth

GoCluster behavior is owned by the existing repository docs and source tree.
Use this folder to find the right source, not as a replacement for that source.

- Start with [source-map.md](source-map.md) for topic-to-document routing.
- Use [operator-guide-index.md](operator-guide-index.md) for operator support.
- Use [developer-guide-index.md](developer-guide-index.md) for contributor
  support.
- Use [external-authorities.md](external-authorities.md) for current official
  Go, GitHub, Linux/systemd, and PowerShell references.
- Use [gpt-instructions.md](gpt-instructions.md) as the custom GPT instruction
  seed.

When a repo doc and this folder disagree, prefer the repo doc. When a question
depends on effective YAML, runtime config, current code, or upstream tooling,
say so and route to the authoritative source.
