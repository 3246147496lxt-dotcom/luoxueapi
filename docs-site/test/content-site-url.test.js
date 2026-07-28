import assert from "node:assert/strict";
import test from "node:test";

import {
  DEVELOPMENT_MAIN_SITE_URL,
  PRODUCTION_MAIN_SITE_URL,
  resolveMainSiteUrl,
  tutorials,
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

test("Desktop 教程固定使用公开稳定版下载入口并覆盖 Gatekeeper 恢复路径", () => {
  const desktop = tutorials.find((tutorial) => tutorial.id === "desktop");

  assert.ok(desktop);
  assert.equal(
    desktop.steps[0].link.href,
    `${PRODUCTION_MAIN_SITE_URL}/api/v1/public/desktop/releases/macos/latest/download`,
  );
  assert.match(desktop.steps[2].description, /隐私与安全/);
  assert.match(desktop.steps[2].description, /仍要打开/);
  assert.match(desktop.steps.at(-1).note.text, /恢复并准备卸载/);
});
