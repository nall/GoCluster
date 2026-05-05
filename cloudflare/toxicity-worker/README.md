# Toxicity Worker

This Worker is the optional AI boundary for GoCluster human spot comments.

## Contract

- Endpoint: `POST /classify`
- Auth: `Authorization: Bearer <CLASSIFIER_TOKEN>`
- Request body: `{ "comment": "cleaned free-text comment" }`
- Response body:
  - safe: `{ "status": "safe", "categories": [], "model": "<model id>" }`
  - toxic: `{ "status": "toxic", "categories": ["S..."], "model": "<model id>" }`

The Worker must receive only the cleaned comment text. Do not add callsigns,
mode, band, source, IP, session identifiers, raw spot lines, or archive records
to the request.

## Runtime

`wrangler.toml` binds Cloudflare Workers AI as `AI`. The handler calls
`@cf/meta/llama-guard-3-8b` and parses the Llama Guard `safe` / `unsafe`
result. Configure the bearer token as a Worker secret named
`CLASSIFIER_TOKEN`; configure GoCluster to read its own bearer token value from
the environment variable named by `data/config/toxicity.yaml`.

## Test

```text
npm test
```

The tests mock `env.AI.run` and assert that only the complete cleaned comment is
sent to the model.
