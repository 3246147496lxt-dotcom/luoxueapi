import assert from "node:assert/strict";
import test from "node:test";

import {
  DEVELOPMENT_MAIN_SITE_URL,
  PRODUCTION_MAIN_SITE_URL,
  resolveMainSiteUrl,
} from "../src/content.js";

test("开发环境默认返回本地主站，生产环境继续返回正式域名", () => {
  assert.equal(resolveMainSiteUrl({ DEV: true }), DEVELOPMENT_MAIN_SITE_URL);
  assert.equal(resolveMainSiteUrl({ DEV: false }), PRODUCTION_MAIN_SITE_URL);
  assert.equal(resolveMainSiteUrl(), PRODUCTION_MAIN_SITE_URL);
});

test("VITE_MAIN_SITE_URL 可以覆盖开发和生产默认值", () => {
  for (const DEV of [true, false]) {
    assert.equal(
      resolveMainSiteUrl({ DEV, VITE_MAIN_SITE_URL: " http://localhost:5173/ " }),
      "http://localhost:5173",
    );
  }
});

test("无效覆盖值不会生成损坏或不安全的反向链接", () => {
  for (const VITE_MAIN_SITE_URL of [
    "javascript:alert(1)",
    "https://user:secret@example.com",
    "https://example.com/?redirect=1",
  ]) {
    assert.equal(
      resolveMainSiteUrl({ DEV: true, VITE_MAIN_SITE_URL }),
      DEVELOPMENT_MAIN_SITE_URL,
    );
  }
});
