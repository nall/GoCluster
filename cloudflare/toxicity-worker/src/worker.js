const MODEL = "@cf/meta/llama-guard-3-8b";
const MAX_COMMENT_BYTES = 1024;

export default {
  async fetch(request, env) {
    if (request.method !== "POST") {
      return json({ error: "method_not_allowed" }, 405);
    }
    const expected = env.CLASSIFIER_TOKEN;
    if (!expected || request.headers.get("authorization") !== `Bearer ${expected}`) {
      return json({ error: "unauthorized" }, 401);
    }
    let body;
    try {
      body = await request.json();
    } catch {
      return json({ error: "invalid_json" }, 400);
    }
    const comment = typeof body?.comment === "string" ? body.comment.trim() : "";
    if (!comment) {
      return json({ status: "safe", categories: [], model: MODEL });
    }
    if (new TextEncoder().encode(comment).length > MAX_COMMENT_BYTES) {
      return json({ error: "comment_too_large" }, 413);
    }
    const response = await env.AI.run(MODEL, {
      messages: [{ role: "user", content: comment }],
      temperature: 0,
      max_tokens: 64
    });
    const text = modelText(response);
    const parsed = parseLlamaGuard(text);
    return json({ ...parsed, model: MODEL });
  }
};

export function parseLlamaGuard(text) {
  const lines = String(text || "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  const first = (lines[0] || "").toLowerCase();
  if (first === "safe") {
    return { status: "safe", categories: [] };
  }
  if (first === "unsafe") {
    const categories = lines.slice(1).flatMap((line) => line.split(/[,\s]+/)).filter(Boolean);
    return { status: "toxic", categories };
  }
  throw new Error("unexpected_llama_guard_response");
}

function modelText(response) {
  if (typeof response === "string") {
    return response;
  }
  if (typeof response?.response === "string") {
    return response.response;
  }
  if (typeof response?.result?.response === "string") {
    return response.result.response;
  }
  return "";
}

function json(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" }
  });
}
