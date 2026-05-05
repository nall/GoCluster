import test from "node:test";
import assert from "node:assert/strict";
import worker, { parseLlamaGuard } from "../src/worker.js";

test("parseLlamaGuard maps safe output", () => {
  assert.deepEqual(parseLlamaGuard("safe\n"), { status: "safe", categories: [] });
});

test("parseLlamaGuard maps unsafe categories", () => {
  assert.deepEqual(parseLlamaGuard("unsafe\nS10 S11"), {
    status: "toxic",
    categories: ["S10", "S11"]
  });
});

test("worker requires bearer auth", async () => {
  const response = await worker.fetch(new Request("https://example.test/classify", { method: "POST" }), {
    CLASSIFIER_TOKEN: "secret",
    AI: { run: async () => "safe" }
  });
  assert.equal(response.status, 401);
});

test("worker sends only comment to AI", async () => {
  let seen;
  const response = await worker.fetch(
    new Request("https://example.test/classify", {
      method: "POST",
      headers: { authorization: "Bearer secret" },
      body: JSON.stringify({ comment: "POTA insulto" })
    }),
    {
      CLASSIFIER_TOKEN: "secret",
      AI: {
        run: async (_model, input) => {
          seen = input;
          return { response: "unsafe\nS10" };
        }
      }
    }
  );
  assert.equal(response.status, 200);
  assert.deepEqual(seen.messages, [{ role: "user", content: "POTA insulto" }]);
  assert.deepEqual(await response.json(), {
    status: "toxic",
    categories: ["S10"],
    model: "@cf/meta/llama-guard-3-8b"
  });
});
