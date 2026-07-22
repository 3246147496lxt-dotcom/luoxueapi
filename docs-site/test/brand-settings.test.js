import assert from "node:assert/strict";
import test from "node:test";

import {
  loadPublicBrandSettings,
  normalizePublicBrandSettings,
  sanitizeBrandLogo,
  splitBrandApiSuffix,
} from "../src/brand-settings.js";

const fallback = {
  name: "落雪API",
  logo: "/tutorial-docs/assets/brand-mark.svg",
};

function responseFor(payload, { ok = true } = {}) {
  return {
    ok,
    async json() {
      return payload;
    },
  };
}

test("解析公共设置包装并保留管理员配置的 SVG data URL", () => {
  const logo = "data:image/svg+xml;base64,PHN2Zy8+";
  const result = normalizePublicBrandSettings({
    code: 0,
    data: {
      site_name: "  雪落 API  ",
      site_logo: `  ${logo}  `,
    },
  }, fallback);

  assert.deepEqual(result, { name: "雪落 API", logo });
});

test("Logo URL 规则与主站一致", () => {
  assert.equal(sanitizeBrandLogo("/brand/custom.svg"), "/brand/custom.svg");
  assert.equal(sanitizeBrandLogo("https://cdn.example.com/logo.svg"), "https://cdn.example.com/logo.svg");
  assert.equal(sanitizeBrandLogo("data:image/png;base64,AAAA"), "data:image/png;base64,AAAA");

  for (const unsafe of [
    "//evil.example/logo.svg",
    "javascript:alert(1)",
    "data:text/html,<script>alert(1)</script>",
    "brand/logo.svg",
  ]) {
    assert.equal(sanitizeBrandLogo(unsafe), "");
  }
});

test("仅拆分站点名末尾的 API 品牌后缀", () => {
  assert.deepEqual(splitBrandApiSuffix("落雪API"), { base: "落雪", apiSuffix: "API" });
  assert.deepEqual(splitBrandApiSuffix("落雪 API"), { base: "落雪", apiSuffix: "API" });
  assert.deepEqual(splitBrandApiSuffix("Snow api"), { base: "Snow", apiSuffix: "api" });
  assert.deepEqual(splitBrandApiSuffix("RapidAPI"), { base: "Rapid", apiSuffix: "API" });

  for (const value of ["雪落开发者平台", "API", "API 网关", "GraphAPIary"]) {
    assert.deepEqual(splitBrandApiSuffix(value), { base: value, apiSuffix: "" });
  }
});

test("站点名和 Logo 分别回退，不让单个坏字段污染另一个字段", () => {
  assert.deepEqual(
    normalizePublicBrandSettings({ site_name: "自定义站点", site_logo: "javascript:alert(1)" }, fallback),
    { name: "自定义站点", logo: fallback.logo },
  );
  assert.deepEqual(
    normalizePublicBrandSettings({ site_name: "   ", site_logo: "/brand/custom.svg" }, fallback),
    { name: fallback.name, logo: "/brand/custom.svg" },
  );
});

test("从根路径公共端点读取品牌且禁用缓存", async () => {
  let request;
  const controller = new AbortController();
  const result = await loadPublicBrandSettings({
    fallback,
    signal: controller.signal,
    fetchImpl: async (url, options) => {
      request = { url, options };
      return responseFor({
        code: 0,
        data: { site_name: "落雪API", site_logo: "/brand/live.svg" },
      });
    },
  });

  assert.deepEqual(result, { name: "落雪API", logo: "/brand/live.svg" });
  assert.equal(request.url, "/api/v1/settings/public");
  assert.equal(request.options.method, "GET");
  assert.equal(request.options.credentials, "same-origin");
  assert.equal(request.options.cache, "no-store");
  assert.deepEqual(request.options.headers, { Accept: "application/json" });
  assert.strictEqual(request.options.signal, controller.signal);
});

test("接口异常、失败响应和错误包装都安全回退", async () => {
  const cases = [
    async () => responseFor(null, { ok: false }),
    async () => responseFor({ code: 500, data: { site_name: "错误站点" } }),
    async () => responseFor("not-an-object"),
    async () => { throw new TypeError("Failed to fetch"); },
  ];

  for (const fetchImpl of cases) {
    assert.deepEqual(await loadPublicBrandSettings({ fallback, fetchImpl }), fallback);
  }
});
