const PUBLIC_SETTINGS_ENDPOINT = "/api/v1/settings/public";

function fallbackBrand(fallback) {
  return {
    name: fallback.name,
    logo: fallback.logo,
  };
}

function unwrapResponse(payload) {
  if (typeof payload !== "object" || payload === null || Array.isArray(payload)) return null;
  if (payload.data === undefined) return payload;
  if (payload.code !== undefined && payload.code !== 0) return null;
  return payload.data;
}

function normalizeSiteName(value, fallback) {
  if (typeof value !== "string") return fallback;
  const normalized = value.trim();
  return normalized || fallback;
}

export function sanitizeBrandLogo(value) {
  if (typeof value !== "string") return "";
  const normalized = value.trim();
  if (!normalized) return "";

  if (normalized.startsWith("/") && !normalized.startsWith("//")) return normalized;
  if (normalized.startsWith("data:image/")) return normalized;
  if (!/^https?:\/\//iu.test(normalized)) return "";

  try {
    const parsed = new URL(normalized);
    return parsed.protocol === "http:" || parsed.protocol === "https:" ? parsed.toString() : "";
  } catch {
    return "";
  }
}

export function splitBrandApiSuffix(value) {
  const normalized = typeof value === "string" ? value.trim() : "";
  const match = /^(.*?)(?:\s*)(api)$/iu.exec(normalized);

  if (!match?.[1]?.trim()) return { base: normalized, apiSuffix: "" };

  return {
    base: match[1].trimEnd(),
    apiSuffix: match[2],
  };
}

export function normalizePublicBrandSettings(payload, fallback) {
  const response = unwrapResponse(payload);
  if (typeof response !== "object" || response === null || Array.isArray(response)) {
    return fallbackBrand(fallback);
  }

  return {
    name: normalizeSiteName(response.site_name, fallback.name),
    logo: sanitizeBrandLogo(response.site_logo) || fallback.logo,
  };
}

export async function loadPublicBrandSettings({
  fetchImpl = globalThis.fetch,
  fallback,
  signal,
  endpoint = PUBLIC_SETTINGS_ENDPOINT,
} = {}) {
  const safeFallback = fallbackBrand(fallback);
  if (typeof fetchImpl !== "function") return safeFallback;

  try {
    const response = await fetchImpl(endpoint, {
      method: "GET",
      credentials: "same-origin",
      cache: "no-store",
      headers: { Accept: "application/json" },
      signal,
    });

    if (!response.ok) return safeFallback;
    return normalizePublicBrandSettings(await response.json(), safeFallback);
  } catch {
    return safeFallback;
  }
}
