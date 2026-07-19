import { readFile, readdir, stat } from "node:fs/promises";
import { extname, join, relative } from "node:path";

const projectRoot = new URL("../", import.meta.url);
const distDir = new URL("dist/", projectRoot);
const expectedBase = "/tutorial-docs/";
const textExtensions = new Set([".html", ".js", ".css", ".json", ".svg"]);

async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) files.push(...await walk(path));
    if (entry.isFile()) files.push(path);
  }

  return files;
}

const distPath = distDir.pathname;
const distStats = await stat(distPath).catch(() => null);
if (!distStats?.isDirectory()) {
  throw new Error("dist/ 不存在，请先运行 npm run build");
}

const files = await walk(distPath);
const textFiles = files.filter((file) => textExtensions.has(extname(file)));
const failures = [];

for (const file of textFiles) {
  const source = await readFile(file, "utf8");
  const displayPath = relative(distPath, file);

  if (/(["'(=])\/assets\//.test(source)) {
    failures.push(`${displayPath} 仍包含根路径 /assets/`);
  }

  if (source.includes("https://docs.luoxueapi.cc")) {
    failures.push(`${displayPath} 仍引用旧文档子域名`);
  }
}

const indexHtml = await readFile(new URL("index.html", distDir), "utf8");
if (!indexHtml.includes(`${expectedBase}assets/`)) {
  failures.push("index.html 未使用 /tutorial-docs/ 资源前缀");
}
if (!indexHtml.includes('href="https://luoxueapi.cc/tutorial-docs/"')) {
  failures.push("index.html 缺少规范文档地址");
}

if (failures.length > 0) {
  throw new Error(`生产构建校验失败：\n- ${failures.join("\n- ")}`);
}

console.log(`生产构建路径校验通过：${files.length} 个文件均使用 ${expectedBase}`);
