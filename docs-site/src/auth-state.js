const AUTH_TOKEN_KEY = "auth_token";
const AUTH_USER_KEY = "auth_user";
const DEFAULT_TIMEOUT_MS = 5000;

const guestState = () => ({ status: "guest", user: null });
const unknownState = () => ({ status: "unknown", user: null });
const authenticatedState = (user) => ({ status: "authenticated", user });

function resolveStorage(storage) {
  if (storage) {
    return storage;
  }

  try {
    return globalThis.localStorage;
  } catch {
    return undefined;
  }
}

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function isUserRecord(value) {
  if (!isRecord(value)) {
    return false;
  }

  const hasId =
    (typeof value.id === "number" && Number.isFinite(value.id)) ||
    (typeof value.id === "string" && value.id.trim().length > 0);
  const hasKnownProfileField = [value.username, value.email, value.display_name, value.nickname]
    .some((field) => typeof field === "string" && field.trim().length > 0);

  return hasId || hasKnownProfileField;
}

function readStoredAuthSnapshot(storage) {
  const target = resolveStorage(storage);
  if (!target || typeof target.getItem !== "function") {
    return { status: "unknown", user: null, token: null };
  }

  let tokenValue;
  let userValue;
  try {
    tokenValue = target.getItem(AUTH_TOKEN_KEY);
    userValue = target.getItem(AUTH_USER_KEY);
  } catch {
    return { status: "unknown", user: null, token: null };
  }

  const token = typeof tokenValue === "string" ? tokenValue.trim() : "";
  const hasToken = token.length > 0;
  const hasUser = typeof userValue === "string" && userValue.trim().length > 0;

  if (!hasToken && !hasUser) {
    return { status: "guest", user: null, token: null };
  }

  if (!hasToken || !hasUser) {
    return { status: "unknown", user: null, token: hasToken ? token : null };
  }

  try {
    const user = JSON.parse(userValue);
    if (!isUserRecord(user)) {
      return { status: "unknown", user: null, token };
    }
    return { status: "authenticated", user, token };
  } catch {
    return { status: "unknown", user: null, token };
  }
}

/**
 * Read the main application's cached authentication state without mutating it.
 * Only auth_token and auth_user are accessed; refresh_token is intentionally
 * outside this module's responsibility.
 */
export function readStoredAuth(storage) {
  const { status, user } = readStoredAuthSnapshot(storage);
  return { status, user };
}

function extractResponseUser(payload) {
  if (!isRecord(payload)) {
    return null;
  }

  const candidate = Object.hasOwn(payload, "data") ? payload.data : payload;
  return isUserRecord(candidate) ? candidate : null;
}

function userIdentitiesConflict(cachedUser, verifiedUser) {
  if (cachedUser.id === undefined || verifiedUser.id === undefined) {
    return false;
  }
  return String(cachedUser.id) !== String(verifiedUser.id);
}

function didTokenChange(storage, expectedToken) {
  const current = readStoredAuthSnapshot(storage);
  return current.status !== "authenticated" || current.token !== expectedToken;
}

function isAbortError(error) {
  return error?.name === "AbortError";
}

/**
 * Verify the cached same-origin session with GET /api/v1/auth/me.
 *
 * @param {object} [options]
 * @param {Storage} [options.storage] Storage-compatible object.
 * @param {typeof fetch} [options.fetchImpl] Injectable fetch for tests.
 * @param {number} [options.timeoutMs=5000] Verification timeout.
 * @param {AbortSignal} [options.signal] Optional caller cancellation signal.
 * @returns {Promise<{status: "authenticated"|"guest"|"unknown", user: object|null}>}
 */
export async function verifyAuthState({
  storage,
  fetchImpl = globalThis.fetch,
  timeoutMs = DEFAULT_TIMEOUT_MS,
  signal,
} = {}) {
  const targetStorage = resolveStorage(storage);
  const cached = readStoredAuthSnapshot(targetStorage);

  if (cached.status === "guest") {
    return guestState();
  }
  if (cached.status !== "authenticated" || !cached.token) {
    return unknownState();
  }

  if (typeof fetchImpl !== "function") {
    return authenticatedState(cached.user);
  }

  const controller = new AbortController();
  let timedOut = false;
  const normalizedTimeout = Number.isFinite(timeoutMs) && timeoutMs > 0
    ? timeoutMs
    : DEFAULT_TIMEOUT_MS;
  const timeoutId = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, normalizedTimeout);

  const forwardAbort = () => controller.abort(signal?.reason);
  if (signal?.aborted) {
    clearTimeout(timeoutId);
    return unknownState();
  }
  signal?.addEventListener("abort", forwardAbort, { once: true });

  try {
    let response;
    try {
      response = await fetchImpl("/api/v1/auth/me", {
        method: "GET",
        headers: {
          Accept: "application/json",
          Authorization: `Bearer ${cached.token}`,
        },
        credentials: "same-origin",
        cache: "no-store",
        signal: controller.signal,
      });
    } catch (error) {
      if (timedOut || controller.signal.aborted || isAbortError(error)) {
        return unknownState();
      }

      if (didTokenChange(targetStorage, cached.token)) {
        return unknownState();
      }
      return authenticatedState(cached.user);
    }

    const status = Number(response?.status);
    const responseIsOk = typeof response?.ok === "boolean"
      ? response.ok
      : status >= 200 && status < 300;

    if (status === 401 || !responseIsOk || typeof response?.json !== "function") {
      return unknownState();
    }

    let responsePayload;
    try {
      responsePayload = await response.json();
    } catch {
      return unknownState();
    }

    const verifiedUser = extractResponseUser(responsePayload);
    if (
      !verifiedUser ||
      userIdentitiesConflict(cached.user, verifiedUser) ||
      didTokenChange(targetStorage, cached.token)
    ) {
      return unknownState();
    }

    return authenticatedState(verifiedUser);
  } finally {
    clearTimeout(timeoutId);
    signal?.removeEventListener("abort", forwardAbort);
  }
}
