const REPO_OWNER = "N2WQ";
const REPO_NAME = "GoCluster";
const BRANCH = "main";

const RAW_BASE = `https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/${BRANCH}`;

const ENTRYPOINT_PATH = "customgpt/source-map.md";
const TROUBLESHOOTING_PATH = "customgpt/troubleshooting-index.md";
const EXTERNAL_AUTHORITIES_PATH = "customgpt/external-authorities.md";

const MAX_FILE_CHARS = 140000;
const MAX_BUNDLE_FILES = 12;
const MAX_RELATED_PATHS = 80;
const AUTH_SECRET_BINDING = "GOCLUSTER_DOCS_ACTION_TOKEN";
const AUTH_MODE = "bearer";

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    if (url.pathname === "/privacy") {
      return privacyPolicyResponse();
    }

    if (request.method !== "GET") {
      return jsonResponse(
        {
          error: "method_not_allowed",
          message: "Only GET is supported"
        },
        405
      );
    }

    if (!isAuthenticated(request, env)) {
      return unauthorizedResponse();
    }

    try {
      if (url.pathname === "/version") {
        return jsonResponse({
          service: "gocluster-docs-action",
          status: "ok",
          repo: `${REPO_OWNER}/${REPO_NAME}`,
          branch: BRANCH,
          backend: "raw-github-file-fetch",
          entrypoint: ENTRYPOINT_PATH,
          auth: AUTH_MODE,
          retrieved_at: new Date().toISOString(),
          message: "Worker is reachable with authenticated access"
        });
      }

      if (url.pathname === "/entrypoint" || url.pathname === "/source-map") {
        return await fetchRepoFileResponse(ENTRYPOINT_PATH, "");
      }

      if (url.pathname === "/troubleshooting-index") {
        return await fetchRepoFileResponse(TROUBLESHOOTING_PATH, "");
      }

      if (url.pathname === "/external-authorities") {
        return await fetchRepoFileResponse(EXTERNAL_AUTHORITIES_PATH, "");
      }

      if (url.pathname === "/doc" || url.pathname === "/file") {
        const requested = url.searchParams.get("path") || url.searchParams.get("url");
        const base = url.searchParams.get("base") || "";

        if (!requested) {
          return jsonResponse(
            {
              error: "missing_path",
              message: "Provide a repo-relative path using ?path= or a matching raw GitHub URL using ?url="
            },
            400
          );
        }

        return await fetchRepoFileResponse(requested, base);
      }

      if (url.pathname === "/bundle") {
        const rawPaths = collectBundlePaths(url);

        if (rawPaths.length === 0) {
          return jsonResponse(
            {
              error: "missing_paths",
              message: "Provide paths using ?paths=a,b,c or repeated ?path=a&path=b parameters"
            },
            400
          );
        }

        const files = [];
        const requestedPaths = dedupeStrings(rawPaths).slice(0, MAX_BUNDLE_FILES);

        for (const rawPath of requestedPaths) {
          const file = await fetchRepoFilePayload(rawPath, "");
          if (file.error) {
            return jsonResponse(
              {
                error: "bundle_file_failed",
                message: "One requested bundle path could not be retrieved",
                failed_path: rawPath,
                file_error: file
              },
              file.status || 500
            );
          }
          files.push(file);
        }

        return jsonResponse({
          repo: `${REPO_OWNER}/${REPO_NAME}`,
          branch: BRANCH,
          auth: AUTH_MODE,
          retrieved_at: new Date().toISOString(),
          requested_paths: requestedPaths,
          file_count: files.length,
          files
        });
      }

      return jsonResponse(
        {
          error: "not_found",
          message: "Use /version, /entrypoint, /source-map, /troubleshooting-index, /external-authorities, /doc?path=, /file?path=, /bundle, or /privacy"
        },
        404
      );
    } catch (err) {
      return jsonResponse(
        {
          error: "worker_exception",
          message: err && err.message ? err.message : String(err)
        },
        500
      );
    }
  }
};

async function fetchRepoFileResponse(requestedPathOrUrl, basePath) {
  const payload = await fetchRepoFilePayload(requestedPathOrUrl, basePath);

  if (payload.error) {
    return jsonResponse(payload, payload.status || 500);
  }

  return jsonResponse(payload);
}

async function fetchRepoFilePayload(requestedPathOrUrl, basePath) {
  const path = resolveRepoPath(requestedPathOrUrl, basePath);

  if (!path) {
    return {
      status: 400,
      error: "invalid_path",
      message: "Path could not be resolved to a safe GoCluster repo-relative path",
      requested: requestedPathOrUrl,
      base: basePath || ""
    };
  }

  if (!isSafeRepoPath(path)) {
    return {
      status: 403,
      error: "path_not_allowed",
      message: "Requested path is not safe for retrieval",
      path
    };
  }

  const sourceUrl = toRawUrl(path);

  const response = await fetch(sourceUrl, {
    headers: {
      "user-agent": "gocluster-docs-action/4.3-bearer"
    }
  });

  if (!response.ok) {
    return {
      status: response.status === 404 ? 404 : 502,
      error: response.status === 404 ? "path_not_found" : "github_fetch_failed",
      message: response.status === 404
        ? "Requested path was not found in the GoCluster repository"
        : "Could not retrieve the requested file from raw GitHub",
      path,
      source_url: sourceUrl,
      upstream_status: response.status
    };
  }

  const rawContent = await response.text();
  const trimmed = trimContent(rawContent, MAX_FILE_CHARS);
  const header = extractHeader(rawContent);
  const related = extractRelatedPaths(rawContent, path).slice(0, MAX_RELATED_PATHS);
  const structured = structuredRoutesForPath(path, rawContent);

  return {
    repo: `${REPO_OWNER}/${REPO_NAME}`,
    branch: BRANCH,
    path,
    source_url: sourceUrl,
    auth: AUTH_MODE,
    retrieved_at: new Date().toISOString(),
    kind: classifyPath(path),
    header,
    related_paths: related,
    truncated: trimmed.truncated,
    content: trimmed.content,
    ...structured
  };
}

function collectBundlePaths(url) {
  const paths = [];

  for (const path of url.searchParams.getAll("path")) {
    if (path && path.trim()) {
      paths.push(path.trim());
    }
  }

  const csv = url.searchParams.get("paths") || "";
  if (csv.trim()) {
    for (const part of csv.split(",")) {
      if (part.trim()) {
        paths.push(part.trim());
      }
    }
  }

  return paths;
}

function resolveRepoPath(value, basePath) {
  let raw = String(value || "").trim();

  if (!raw) {
    return "";
  }

  raw = raw.replace(/^<+/, "").replace(/>+$/, "").trim();
  raw = raw.split("#")[0].trim();

  const rawPrefix = `https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/${BRANCH}/`;
  if (raw.startsWith(rawPrefix)) {
    return normalizePath(decodeURIComponent(raw.slice(rawPrefix.length)));
  }

  const blobPrefix = `https://github.com/${REPO_OWNER}/${REPO_NAME}/blob/${BRANCH}/`;
  if (raw.startsWith(blobPrefix)) {
    return normalizePath(decodeURIComponent(raw.slice(blobPrefix.length)));
  }

  if (/^https?:\/\//i.test(raw)) {
    return "";
  }

  if (raw.startsWith("./") || raw.startsWith("../")) {
    const baseDir = directoryOf(basePath);
    return normalizePath(joinPath(baseDir, raw));
  }

  return normalizePath(raw);
}

function extractRelatedPaths(content, basePath) {
  const related = [];
  const seen = new Set();

  function add(path, text, context) {
    const normalized = resolveRepoPath(path, basePath);

    if (!normalized || !isSafeRepoPath(normalized)) {
      return;
    }

    if (seen.has(normalized)) {
      return;
    }

    seen.add(normalized);

    related.push({
      path: normalized,
      source_url: toRawUrl(normalized),
      text: text || normalized,
      context: context || ""
    });
  }

  const markdownLinkPattern = /\[([^\]]*)\]\(([^)]+)\)/g;
  let match;

  while ((match = markdownLinkPattern.exec(content || "")) !== null) {
    const text = match[1] || "";
    const href = match[2] || "";
    add(href, text, contextAround(content, match.index, 220));
  }

  const rawUrlPattern = /https:\/\/raw\.githubusercontent\.com\/N2WQ\/GoCluster\/main\/[^\s)\]>'"]+/g;

  while ((match = rawUrlPattern.exec(content || "")) !== null) {
    add(match[0], match[0], contextAround(content, match.index, 220));
  }

  const pathPattern = /\b[\w.-]+(?:\/[\w.-]+)+\.(?:go|md|yaml|yml|json)\b|\b(?:README\.md|AGENTS\.md|VALIDATION\.md)\b/g;

  while ((match = pathPattern.exec(content || "")) !== null) {
    if (match[0].startsWith("raw.githubusercontent.com/") || match[0].startsWith("github.com/")) {
      continue;
    }
    add(match[0], match[0], contextAround(content, match.index, 220));
  }

  return related;
}

function structuredRoutesForPath(path, content) {
  if (path === ENTRYPOINT_PATH) {
    return {
      routes: extractSourceMapRoutes(content)
    };
  }

  if (path === TROUBLESHOOTING_PATH) {
    return {
      symptom_routes: extractSymptomRoutes(content)
    };
  }

  return {};
}

function extractSourceMapRoutes(content) {
  const routes = [];

  for (const cells of markdownTableRows(content)) {
    if (cells.length < 3) {
      continue;
    }

    const topic = plainCellText(cells[0]);
    if (!topic || topic.toLowerCase() === "topic") {
      continue;
    }

    const primaryLinks = extractLinksFromCell(cells[1], ENTRYPOINT_PATH);
    if (primaryLinks.length === 0) {
      continue;
    }

    routes.push({
      topic,
      primary_path: primaryLinks[0].path,
      primary_source_url: primaryLinks[0].source_url,
      supporting_paths: extractLinksFromCell(cells[2], ENTRYPOINT_PATH)
    });
  }

  return routes;
}

function extractSymptomRoutes(content) {
  const routes = [];

  for (const cells of markdownTableRows(content)) {
    if (cells.length < 4) {
      continue;
    }

    const symptom = plainCellText(cells[0]);
    if (!symptom || symptom.toLowerCase().startsWith("symptom")) {
      continue;
    }

    routes.push({
      symptom,
      first_checks: plainCellText(cells[1]),
      route_paths: extractLinksFromCell(cells[2], TROUBLESHOOTING_PATH),
      do_not_guess: plainCellText(cells[3])
    });
  }

  return routes;
}

function markdownTableRows(content) {
  const rows = [];
  for (const line of String(content || "").split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed.startsWith("|") || !trimmed.endsWith("|")) {
      continue;
    }
    if (/^\|\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|$/.test(trimmed)) {
      continue;
    }
    rows.push(splitMarkdownTableRow(trimmed));
  }
  return rows;
}

function splitMarkdownTableRow(row) {
  const body = row.replace(/^\|/, "").replace(/\|$/, "");
  const cells = [];
  let current = "";
  let bracketDepth = 0;
  let parenDepth = 0;
  let inCode = false;

  for (const ch of body) {
    if (ch === "`") {
      inCode = !inCode;
      current += ch;
      continue;
    }
    if (!inCode) {
      if (ch === "[") bracketDepth++;
      if (ch === "]" && bracketDepth > 0) bracketDepth--;
      if (ch === "(") parenDepth++;
      if (ch === ")" && parenDepth > 0) parenDepth--;
      if (ch === "|" && bracketDepth === 0 && parenDepth === 0) {
        cells.push(current.trim());
        current = "";
        continue;
      }
    }
    current += ch;
  }

  cells.push(current.trim());
  return cells;
}

function extractLinksFromCell(cell, basePath) {
  const links = [];
  const seen = new Set();
  const markdownLinkPattern = /\[([^\]]*)\]\(([^)]+)\)/g;
  let match;

  function add(rawPath, text) {
    const path = resolveRepoPath(rawPath, basePath);
    if (!path || !isSafeRepoPath(path) || seen.has(path)) {
      return;
    }
    seen.add(path);
    links.push({
      path,
      source_url: toRawUrl(path),
      text: plainCellText(text || path)
    });
  }

  while ((match = markdownLinkPattern.exec(cell || "")) !== null) {
    add(match[2], match[1]);
  }

  const rawUrlPattern = /https:\/\/raw\.githubusercontent\.com\/N2WQ\/GoCluster\/main\/[^\s)\]>'"]+/g;
  while ((match = rawUrlPattern.exec(cell || "")) !== null) {
    add(match[0], match[0]);
  }

  const pathPattern = /\b[\w.-]+(?:\/[\w.-]+)+\.(?:go|md|yaml|yml|json)\b|\b(?:README\.md|AGENTS\.md|VALIDATION\.md)\b/g;
  while ((match = pathPattern.exec(cell || "")) !== null) {
    if (match[0].startsWith("raw.githubusercontent.com/") || match[0].startsWith("github.com/")) {
      continue;
    }
    add(match[0], match[0]);
  }

  return links;
}

function plainCellText(value) {
  return String(value || "")
    .replace(/\[([^\]]*)\]\(([^)]+)\)/g, "$1")
    .replace(/`([^`]*)`/g, "$1")
    .replace(/<br\s*\/?>/gi, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function extractHeader(content) {
  if (!content) {
    return "";
  }

  const lines = content.split(/\r?\n/);
  const header = [];
  let started = false;

  for (let i = 0; i < Math.min(lines.length, 160); i++) {
    const line = lines[i];
    const trimmed = line.trim();

    if (trimmed === "" && !started) {
      continue;
    }

    if (
      trimmed.startsWith("//") ||
      trimmed.startsWith("#") ||
      trimmed.startsWith("/*") ||
      trimmed.startsWith("*") ||
      trimmed.startsWith("package ")
    ) {
      started = true;
      header.push(line);
      continue;
    }

    if (started) {
      break;
    }

    if (i > 30) {
      break;
    }
  }

  return header.join("\n").trim();
}

function isSafeRepoPath(path) {
  if (!path || path.includes("..") || path.startsWith("/") || path.endsWith("/")) {
    return false;
  }

  const parts = path.split("/");

  if (parts.some((part) => part.startsWith("."))) {
    return false;
  }

  const denyPrefixes = [
    "vendor/",
    "node_modules/",
    "dist/",
    "build/",
    "coverage/",
    "logs/",
    "data/logs/"
  ];

  if (denyPrefixes.some((prefix) => path.startsWith(prefix))) {
    return false;
  }

  const lower = path.toLowerCase();

  const denySuffixes = [
    ".exe",
    ".dll",
    ".so",
    ".dylib",
    ".zip",
    ".tar",
    ".gz",
    ".tgz",
    ".db",
    ".sqlite",
    ".sqlite3",
    ".pem",
    ".key",
    ".crt",
    ".p12",
    ".png",
    ".jpg",
    ".jpeg",
    ".gif",
    ".webp",
    ".ico",
    ".pdf"
  ];

  if (denySuffixes.some((suffix) => lower.endsWith(suffix))) {
    return false;
  }

  const allowSuffixes = [
    ".go",
    ".md",
    ".yaml",
    ".yml"
  ];

  if (allowSuffixes.some((suffix) => lower.endsWith(suffix))) {
    return true;
  }

  if (path === "README" || path === "LICENSE") {
    return true;
  }

  if (path.startsWith("customgpt/") && lower.endsWith(".json")) {
    return true;
  }

  return false;
}

function classifyPath(path) {
  const lower = path.toLowerCase();

  if (lower.endsWith("_test.go")) {
    return "test";
  }

  if (lower.endsWith(".go")) {
    return "source";
  }

  if (lower.endsWith(".yaml") || lower.endsWith(".yml")) {
    return "config";
  }

  if (lower.includes("/decisions/") || lower.includes("decision")) {
    return "decision-doc";
  }

  if (lower.includes("/troubleshooting/") || lower.includes("troubleshooting")) {
    return "troubleshooting-doc";
  }

  if (lower.endsWith(".md")) {
    return "doc";
  }

  if (lower.endsWith(".json")) {
    return "json";
  }

  return "unknown";
}

function trimContent(content, maxChars) {
  if (!maxChars || content.length <= maxChars) {
    return {
      content,
      truncated: false
    };
  }

  return {
    content: content.slice(0, maxChars) + "\n\n[TRUNCATED BY WORKER]",
    truncated: true
  };
}

function toRawUrl(path) {
  return `${RAW_BASE}/${path.split("/").map(encodeURIComponent).join("/")}`;
}

function normalizePath(path) {
  return String(path || "")
    .replace(/\\/g, "/")
    .replace(/^\/+/, "")
    .replace(/\/+/g, "/")
    .trim();
}

function directoryOf(path) {
  const normalized = normalizePath(path);
  const idx = normalized.lastIndexOf("/");

  if (idx === -1) {
    return "";
  }

  return normalized.slice(0, idx);
}

function joinPath(baseDir, relativePath) {
  const parts = [];

  for (const part of `${baseDir}/${relativePath}`.split("/")) {
    if (!part || part === ".") {
      continue;
    }

    if (part === "..") {
      parts.pop();
      continue;
    }

    parts.push(part);
  }

  return parts.join("/");
}

function contextAround(content, index, radius) {
  const start = Math.max(0, index - radius);
  const end = Math.min(content.length, index + radius);

  return content
    .slice(start, end)
    .replace(/\s+/g, " ")
    .trim();
}

function dedupeStrings(values) {
  const seen = new Set();
  const out = [];

  for (const value of values) {
    if (seen.has(value)) {
      continue;
    }

    seen.add(value);
    out.push(value);
  }

  return out;
}

function isAuthenticated(request, env) {
  const expectedToken = env && env[AUTH_SECRET_BINDING];
  if (!expectedToken) {
    return false;
  }

  const header = request.headers.get("authorization") || "";
  const match = header.match(/^Bearer\s+(.+)$/i);
  if (!match) {
    return false;
  }

  return constantTimeEquals(match[1].trim(), String(expectedToken));
}

function constantTimeEquals(actual, expected) {
  const actualText = String(actual || "");
  const expectedText = String(expected || "");
  const maxLength = Math.max(actualText.length, expectedText.length);
  let mismatch = actualText.length ^ expectedText.length;

  for (let i = 0; i < maxLength; i++) {
    const actualCode = i < actualText.length ? actualText.charCodeAt(i) : 0;
    const expectedCode = i < expectedText.length ? expectedText.charCodeAt(i) : 0;
    mismatch |= actualCode ^ expectedCode;
  }

  return mismatch === 0;
}

function unauthorizedResponse() {
  return jsonResponse(
    {
      error: "unauthorized",
      message: "Missing or invalid bearer token"
    },
    401
  );
}

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body, null, 2), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "access-control-allow-origin": "*"
    }
  });
}

function privacyPolicyResponse() {
  const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>GoCluster Repository Retrieval Action Privacy Policy</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body {
      font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      line-height: 1.55;
      max-width: 820px;
      margin: 40px auto;
      padding: 0 20px;
      color: #111827;
    }
    h1, h2 {
      line-height: 1.25;
    }
    code {
      background: #f3f4f6;
      padding: 2px 4px;
      border-radius: 4px;
    }
  </style>
</head>
<body>
  <h1>GoCluster Repository Retrieval Action Privacy Policy</h1>
  <p><strong>Effective date:</strong> May 2, 2026</p>

  <p>
    This privacy policy describes the GoCluster Repository Retrieval Action used by a custom GPT
    to retrieve current public documentation, source files, tests, YAML configuration, and related
    repository context from the <code>N2WQ/GoCluster</code> GitHub repository.
  </p>

  <h2>What this action does</h2>
  <p>
    The action provides read-only access to selected public GoCluster repository files. It retrieves
    individual files from the public GoCluster GitHub repository when requested by the custom GPT.
    Returned files may include related repository paths discovered from Markdown links, source headers,
    YAML comments, and repo-relative references.
  </p>

  <h2>Information processed</h2>
  <p>
    The action receives technical API requests from ChatGPT, including the requested endpoint,
    requested repo path or URL, request headers, timestamp, and standard network metadata such as
    IP address and user agent as processed by Cloudflare.
  </p>

  <p>
    The action does not require users to create an account, provide a name, provide an email address,
    or submit personal information.
  </p>

  <h2>Information users should not submit</h2>
  <p>
    Users should not submit secrets, passwords, API keys, private keys, tokens, private configuration
    files, private logs, private hostnames, or sensitive operational data. The action is intended only
    for retrieving public GoCluster repository evidence.
  </p>

  <h2>How information is used</h2>
  <p>
    Request information may be used to operate the retrieval action, troubleshoot failures, monitor
    abuse, improve reliability, and secure the service.
  </p>

  <h2>Data sharing</h2>
  <p>
    Requests are processed by Cloudflare Workers. The action retrieves public repository files from
    GitHub. No user account data is intentionally sold, rented, or shared for advertising.
  </p>

  <h2>Data retention</h2>
  <p>
    The service itself does not intentionally store user-submitted content. Cloudflare and related
    infrastructure providers may retain operational logs according to their own service policies.
  </p>

  <h2>Security</h2>
  <p>
    Repository retrieval endpoints require a bearer token configured by the GPT owner.
    This privacy page remains public so users can review the policy before using the action.
  </p>

  <h2>Changes</h2>
  <p>This policy may be updated as the retrieval action changes.</p>

  <h2>Contact</h2>
  <p>
    For questions about this retrieval action, contact the GPT owner through the custom GPT listing
    or through the public GoCluster project repository.
  </p>
</body>
</html>`;

  return new Response(html, {
    status: 200,
    headers: {
      "content-type": "text/html; charset=utf-8",
      "access-control-allow-origin": "*"
    }
  });
}
