import assert from "node:assert/strict";
import test from "node:test";

import {
  loadPublishedDocumentation,
  normalizeDocumentationResponse,
  resolveActiveTutorialId,
  shouldRenderDocumentationImage,
  subscribeToTutorialHashChanges,
} from "../src/documentation.js";

const bundledTutorials = [{ id: "bundled", tabLabel: "本地教程", steps: [] }];

function publishedPayload(overrides = {}) {
  return {
    version: 7,
    published_at: "2026-07-19T10:00:00+08:00",
    content: {
      schema_version: 1,
      tutorials: [
        {
          id: "quick-start",
          tab_label: "快速开始",
          icon: "key",
          description: "从这里开始。",
          steps: [
            {
              title: "创建密钥",
              description: "进入控制台创建一个密钥。",
              note: { tone: "warning", text: "不要公开密钥。" },
              note_placement: "after-image",
              code: { label: "示例", value: "curl https://example.com/v1/models" },
              image: {
                src: "/api/v1/public/documentation/assets/key-upload.png",
                alt: "创建密钥页面",
              },
              link: { label: "进入控制台", href: "https://example.com/keys" },
            },
          ],
        },
      ],
    },
    ...overrides,
  };
}

function responseFor(payload, { ok = true, status = 200 } = {}) {
  return {
    ok,
    status,
    json: async () => payload,
  };
}

test("加载已发布内容并把外部 snake_case 字段映射为渲染字段", async () => {
  let request;
  const result = await loadPublishedDocumentation({
    fallbackTutorials: bundledTutorials,
    fetchImpl: async (url, options) => {
      request = { url, options };
      return responseFor(publishedPayload());
    },
  });

  assert.equal(result.source, "published");
  assert.equal(result.version, 7);
  assert.equal(result.publishedAt, "2026-07-19T10:00:00+08:00");
  assert.equal(result.tutorials[0].tabLabel, "快速开始");
  assert.equal(result.tutorials[0].steps[0].notePlacement, "after-image");
  assert.equal(
    result.tutorials[0].steps[0].image.src,
    "/api/v1/public/documentation/assets/key-upload.png",
  );
  assert.equal(request.url, "/api/v1/public/documentation");
  assert.equal(request.options.credentials, "same-origin");
  assert.equal(request.options.cache, "no-cache");
  assert.deepEqual(request.options.headers, { Accept: "application/json" });
});

test("只渲染已发布内容中的非内置图片", () => {
  assert.equal(shouldRenderDocumentationImage({
    src: "/api/v1/public/documentation/assets/uploaded.png",
  }, "published"), true);
  assert.equal(shouldRenderDocumentationImage({
    src: "/tutorial-docs/assets/old-screen.png?version=2",
  }, "published"), false);
  assert.equal(shouldRenderDocumentationImage({
    src: "/api/v1/public/documentation/assets/uploaded.png",
  }, "bundled"), false);
  assert.equal(shouldRenderDocumentationImage(null, "published"), false);
});

test("兼容 response.Success 的 data 包装", () => {
  const normalized = normalizeDocumentationResponse({ code: 0, data: publishedPayload() });

  assert.ok(normalized);
  assert.equal(normalized.tutorials[0].id, "quick-start");
  assert.equal(normalized.tutorials[0].steps[0].link.href, "https://example.com/keys");
});

test("404 时保留打包内置教程", async () => {
  const result = await loadPublishedDocumentation({
    fallbackTutorials: bundledTutorials,
    fetchImpl: async () => responseFor(null, { ok: false, status: 404 }),
  });

  assert.equal(result.source, "bundled");
  assert.strictEqual(result.tutorials, bundledTutorials);
});

test("网络失败时保留打包内置教程", async () => {
  const result = await loadPublishedDocumentation({
    fallbackTutorials: bundledTutorials,
    fetchImpl: async () => {
      throw new TypeError("Failed to fetch");
    },
  });

  assert.equal(result.source, "bundled");
  assert.strictEqual(result.tutorials, bundledTutorials);
});

test("内容结构无效时保留打包内置教程", async () => {
  const result = await loadPublishedDocumentation({
    fallbackTutorials: bundledTutorials,
    fetchImpl: async () => responseFor({
      ...publishedPayload(),
      content: { schema_version: 2, tutorials: [] },
    }),
  });

  assert.equal(result.source, "bundled");
  assert.strictEqual(result.tutorials, bundledTutorials);
});

test("拒绝不安全的链接和图片协议，不把已发布内容换入页面", async () => {
  const payload = publishedPayload();
  payload.content.tutorials[0].steps[0].link.href = "javascript:alert(1)";
  payload.content.tutorials[0].steps[0].image.src = "data:image/svg+xml,<svg/>";

  assert.equal(normalizeDocumentationResponse(payload), null);

  const result = await loadPublishedDocumentation({
    fallbackTutorials: bundledTutorials,
    fetchImpl: async () => responseFor(payload),
  });
  assert.equal(result.source, "bundled");
});

test("重复教程 ID 被视为无效内容", () => {
  const payload = publishedPayload();
  payload.content.tutorials.push(structuredClone(payload.content.tutorials[0]));

  assert.equal(normalizeDocumentationResponse(payload), null);
});

test("支持全部后台图标与说明在图片前后的放置方式", () => {
  for (const icon of ["key", "client", "api", "wallet", "image", "help", "book", "code", "terminal"]) {
    const payload = publishedPayload();
    payload.content.tutorials[0].icon = icon;
    payload.content.tutorials[0].steps[0].note_placement = "before-image";

    const normalized = normalizeDocumentationResponse(payload);
    assert.equal(normalized.tutorials[0].icon, icon);
    assert.equal(normalized.tutorials[0].steps[0].notePlacement, "before-image");
  }
});

test("自定义 SVG 图标优先保留，内置图标继续作为回退", () => {
  const payload = publishedPayload();
  payload.content.tutorials[0].icon = "unknown";
  payload.content.tutorials[0].icon_svg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="url(#ice)" d="M2 2h20v20H2z"/></svg>';

  const normalized = normalizeDocumentationResponse(payload);
  assert.ok(normalized);
  assert.equal(normalized.tutorials[0].icon, "help");
  assert.equal(normalized.tutorials[0].iconSvg, payload.content.tutorials[0].icon_svg);
});

test("危险的自定义 SVG 被忽略且不影响教程与内置回退", () => {
  for (const iconSvg of [
    '<svg onload="alert(1)"><path d="M0 0h1v1z"/></svg>',
    '<svg><script>alert(1)</script></svg>',
    '<svg><foreignObject><div>HTML</div></foreignObject></svg>',
    '<svg><use href="https://evil.example/icon.svg#x"/></svg>',
    '<svg><path fill="url(https://evil.example/fill.svg)"/></svg>',
    '<p>not svg</p>',
  ]) {
    const payload = publishedPayload();
    payload.content.tutorials[0].icon_svg = iconSvg;

    const normalized = normalizeDocumentationResponse(payload);
    assert.ok(normalized);
    assert.equal(normalized.tutorials[0].icon, "key");
    assert.equal(normalized.tutorials[0].iconSvg, undefined);
  }
});

test("拒绝 protocol-relative URL", () => {
  const payload = publishedPayload();
  payload.content.tutorials[0].steps[0].link.href = "//evil.example/phishing";

  assert.equal(normalizeDocumentationResponse(payload), null);
});

test("远端教程载入后按 hash 选择；hash 无效时稳定回到首项", () => {
  const remoteTutorials = [
    { id: "remote-first" },
    { id: "remote-target" },
  ];

  assert.equal(resolveActiveTutorialId(remoteTutorials, "#remote-target"), "remote-target");
  assert.equal(resolveActiveTutorialId(remoteTutorials, "#removed-static-id"), "remote-first");
  assert.equal(resolveActiveTutorialId(remoteTutorials, ""), "remote-first");
});

test("hash 变化时同步当前教程，并在卸载时移除监听", () => {
  const listeners = new Map();
  const target = {
    location: { hash: "#recharge" },
    addEventListener(type, listener) {
      listeners.set(type, listener);
    },
    removeEventListener(type, listener) {
      if (listeners.get(type) === listener) listeners.delete(type);
    },
  };
  const activeIds = [];
  const unsubscribe = subscribeToTutorialHashChanges({
    target,
    tutorialList: [{ id: "quick-start" }, { id: "recharge" }],
    onChange: (id) => activeIds.push(id),
  });

  listeners.get("hashchange")();
  assert.deepEqual(activeIds, ["recharge"]);

  target.location.hash = "#missing";
  listeners.get("hashchange")();
  assert.deepEqual(activeIds, ["recharge", "quick-start"]);

  unsubscribe();
  assert.equal(listeners.has("hashchange"), false);
});
