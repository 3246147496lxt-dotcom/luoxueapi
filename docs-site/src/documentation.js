const DOCUMENTATION_ENDPOINT = "/api/v1/public/documentation";
const MAX_TUTORIALS = 24;
const MAX_STEPS_PER_TUTORIAL = 100;
const MAX_ICON_SVG_LENGTH = 32 * 1024;
const SAFE_URL_PROTOCOLS = new Set(["http:", "https:"]);
const KNOWN_ICONS = new Set([
  "key",
  "client",
  "api",
  "wallet",
  "image",
  "help",
  "book",
  "code",
  "terminal",
]);
const BLOCKED_SVG_ELEMENT_PATTERN = /<\s*\/?\s*(?:a|animate|animatemotion|animatetransform|audio|canvas|discard|embed|feimage|foreignobject|iframe|image|link|meta|mpath|object|script|set|style|video)\b/iu;
const ACTIVE_SVG_ATTRIBUTE_PATTERN = /\s(?:on[a-z0-9_-]+|style|src|xml:base)\s*=/iu;

function plainString(value, { maxLength, allowEmpty = false } = {}) {
  if (typeof value !== "string") return null;
  const normalized = value.trim();
  if (!allowEmpty && !normalized) return null;
  if (maxLength && normalized.length > maxLength) return null;
  return normalized;
}

function safeUrl(value) {
  const normalized = plainString(value, { maxLength: 2048 });
  if (!normalized || /[\u0000-\u001F\u007F]/u.test(normalized)) return null;
  if (/^[\\/]{2}/u.test(normalized)) return null;

  try {
    const parsed = new URL(normalized, "https://documentation.invalid/");
    if (!SAFE_URL_PROTOCOLS.has(parsed.protocol)) return null;
    return normalized;
  } catch {
    return null;
  }
}

function safeSvgIcon(value) {
  const normalized = plainString(value, { maxLength: MAX_ICON_SVG_LENGTH });
  if (!normalized || !/^<svg(?:\s|>)/iu.test(normalized)) return null;
  if (/<!doctype|<\?/iu.test(normalized)) return null;
  if (BLOCKED_SVG_ELEMENT_PATTERN.test(normalized) || ACTIVE_SVG_ATTRIBUTE_PATTERN.test(normalized)) return null;

  for (const match of normalized.matchAll(/\b(?:href|xlink:href)\s*=\s*(["'])(.*?)\1/giu)) {
    if (!match[2].trim().startsWith("#")) return null;
  }
  for (const match of normalized.matchAll(/url\(\s*(["']?)(.*?)\1\s*\)/giu)) {
    if (!match[2].trim().startsWith("#")) return null;
  }
  return normalized;
}

function normalizeNote(note) {
  if (note === undefined || note === null) return undefined;
  if (typeof note !== "object" || Array.isArray(note)) return null;

  const text = plainString(note.text, { maxLength: 4000 });
  if (!text) return null;

  const tone = note.tone === undefined ? undefined : plainString(note.tone, { maxLength: 24 });
  if (tone !== undefined && tone !== "info" && tone !== "warning") return null;

  return tone ? { tone, text } : { text };
}

function normalizeCode(code) {
  if (code === undefined || code === null) return undefined;
  if (typeof code !== "object" || Array.isArray(code)) return null;

  const value = plainString(code.value, { maxLength: 100_000 });
  if (!value) return null;

  const label = code.label === undefined
    ? undefined
    : plainString(code.label, { maxLength: 80 });
  if (code.label !== undefined && !label) return null;

  return label ? { label, value } : { value };
}

function normalizeImage(image) {
  if (image === undefined || image === null) return undefined;
  if (typeof image !== "object" || Array.isArray(image)) return null;

  const src = safeUrl(image.src);
  const alt = plainString(image.alt, { maxLength: 300, allowEmpty: true });
  if (!src || alt === null) return null;

  const caption = image.caption === undefined
    ? undefined
    : plainString(image.caption, { maxLength: 500 });
  if (image.caption !== undefined && !caption) return null;

  return caption ? { src, alt, caption } : { src, alt };
}

function normalizeLink(link) {
  if (link === undefined || link === null) return undefined;
  if (typeof link !== "object" || Array.isArray(link)) return null;

  const label = plainString(link.label, { maxLength: 120 });
  const href = safeUrl(link.href);
  if (!label || !href) return null;

  return { label, href };
}

function normalizeStep(step) {
  if (typeof step !== "object" || step === null || Array.isArray(step)) return null;

  const title = plainString(step.title, { maxLength: 200 });
  const description = plainString(step.description, { maxLength: 10_000 });
  if (!title || !description) return null;

  const note = normalizeNote(step.note);
  const code = normalizeCode(step.code);
  const image = normalizeImage(step.image);
  const link = normalizeLink(step.link);
  if (note === null || code === null || image === null || link === null) return null;

  const notePlacement = step.note_placement === undefined
    ? undefined
    : plainString(step.note_placement, { maxLength: 32 });
  if (notePlacement !== undefined && notePlacement !== "before-image" && notePlacement !== "after-image") {
    return null;
  }
  if (notePlacement && !note) return null;

  return {
    title,
    description,
    ...(note ? { note } : {}),
    ...(notePlacement ? { notePlacement } : {}),
    ...(code ? { code } : {}),
    ...(image ? { image } : {}),
    ...(link ? { link } : {}),
  };
}

function normalizeTutorial(tutorial) {
  if (typeof tutorial !== "object" || tutorial === null || Array.isArray(tutorial)) return null;

  const id = plainString(tutorial.id, { maxLength: 64 });
  const tabLabel = plainString(tutorial.tab_label, { maxLength: 80 });
  const description = plainString(tutorial.description, { maxLength: 4000 });
  if (!id || !/^[a-z0-9][a-z0-9_-]*$/iu.test(id) || !tabLabel || !description) return null;
  if (!Array.isArray(tutorial.steps) || tutorial.steps.length === 0 || tutorial.steps.length > MAX_STEPS_PER_TUTORIAL) {
    return null;
  }

  const steps = tutorial.steps.map(normalizeStep);
  if (steps.some((step) => step === null)) return null;

  const rawIcon = tutorial.icon === undefined
    ? "help"
    : plainString(tutorial.icon, { maxLength: 24 });
  if (!rawIcon) return null;
  const iconSvg = tutorial.icon_svg === undefined
    ? undefined
    : safeSvgIcon(tutorial.icon_svg) || undefined;

  return {
    id,
    tabLabel,
    icon: KNOWN_ICONS.has(rawIcon) ? rawIcon : "help",
    ...(iconSvg ? { iconSvg } : {}),
    description,
    steps,
  };
}

function unwrapResponse(payload) {
  if (typeof payload !== "object" || payload === null || Array.isArray(payload)) return null;
  if (payload.data !== undefined) {
    if (payload.code !== undefined && payload.code !== 0) return null;
    return payload.data;
  }
  return payload;
}

export function normalizeDocumentationResponse(payload) {
  const response = unwrapResponse(payload);
  if (typeof response !== "object" || response === null || Array.isArray(response)) return null;

  const { content } = response;
  if (typeof content !== "object" || content === null || Array.isArray(content)) return null;
  if (content.schema_version !== 1) return null;
  if (!Array.isArray(content.tutorials) || content.tutorials.length === 0 || content.tutorials.length > MAX_TUTORIALS) {
    return null;
  }

  const tutorials = content.tutorials.map(normalizeTutorial);
  if (tutorials.some((tutorial) => tutorial === null)) return null;

  const ids = new Set(tutorials.map((tutorial) => tutorial.id));
  if (ids.size !== tutorials.length) return null;

  const version = Number.isSafeInteger(response.version) && response.version > 0
    ? response.version
    : null;
  const publishedAt = typeof response.published_at === "string"
    ? response.published_at
    : null;

  return { tutorials, version, publishedAt };
}

export function resolveActiveTutorialId(tutorialList, hashValue) {
  if (!Array.isArray(tutorialList) || tutorialList.length === 0) return "";
  const requestedId = typeof hashValue === "string"
    ? hashValue.replace(/^#/, "")
    : "";
  return tutorialList.some((tutorial) => tutorial.id === requestedId)
    ? requestedId
    : tutorialList[0].id;
}

export function subscribeToTutorialHashChanges({
  target = globalThis.window,
  tutorialList,
  onChange,
} = {}) {
  if (
    !target
    || typeof target.addEventListener !== "function"
    || typeof target.removeEventListener !== "function"
    || !Array.isArray(tutorialList)
    || typeof onChange !== "function"
  ) {
    return () => {};
  }

  const handleHashChange = () => {
    onChange(resolveActiveTutorialId(tutorialList, target.location?.hash || ""));
  };

  target.addEventListener("hashchange", handleHashChange);
  return () => target.removeEventListener("hashchange", handleHashChange);
}

function bundledResult(fallbackTutorials) {
  return {
    tutorials: fallbackTutorials,
    source: "bundled",
    version: null,
    publishedAt: null,
  };
}

export async function loadPublishedDocumentation({
  fetchImpl = globalThis.fetch,
  fallbackTutorials,
  signal,
  endpoint = DOCUMENTATION_ENDPOINT,
} = {}) {
  const fallback = bundledResult(fallbackTutorials);
  if (typeof fetchImpl !== "function") return fallback;

  try {
    const response = await fetchImpl(endpoint, {
      method: "GET",
      credentials: "same-origin",
      cache: "default",
      headers: { Accept: "application/json" },
      signal,
    });

    if (!response.ok) return fallback;
    const normalized = normalizeDocumentationResponse(await response.json());
    if (!normalized) return fallback;

    return { ...normalized, source: "published" };
  } catch {
    return fallback;
  }
}
