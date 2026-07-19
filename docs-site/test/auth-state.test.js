import assert from "node:assert/strict";
import test from "node:test";

import { readStoredAuth, verifyAuthState } from "../src/auth-state.js";

function createStorage(entries = {}) {
  const values = new Map(Object.entries(entries));
  const reads = [];

  return {
    reads,
    getItem(key) {
      reads.push(key);
      return values.has(key) ? values.get(key) : null;
    },
    setItem(key, value) {
      values.set(key, String(value));
    },
  };
}

const cachedUser = {
  id: 42,
  username: "luoxue-user",
  email: "user@example.com",
};

test("无缓存状态时返回 guest，且不发请求", async () => {
  const storage = createStorage();
  let fetchCalled = false;

  assert.deepEqual(readStoredAuth(storage), { status: "guest", user: null });
  assert.deepEqual(
    await verifyAuthState({
      storage,
      fetchImpl: async () => {
        fetchCalled = true;
      },
    }),
    { status: "guest", user: null },
  );
  assert.equal(fetchCalled, false);
  assert.equal(storage.reads.includes("refresh_token"), false);
});

test("有效缓存通过同源接口验证后返回最新用户", async () => {
  const storage = createStorage({
    auth_token: "test-token",
    auth_user: JSON.stringify(cachedUser),
    refresh_token: "must-not-be-read",
  });
  const verifiedUser = { ...cachedUser, username: "latest-name" };
  let request;

  const result = await verifyAuthState({
    storage,
    fetchImpl: async (url, options) => {
      request = { url, options };
      return {
        ok: true,
        status: 200,
        json: async () => ({ code: 0, data: verifiedUser }),
      };
    },
  });

  assert.deepEqual(result, { status: "authenticated", user: verifiedUser });
  assert.equal(request.url, "/api/v1/auth/me");
  assert.equal(request.options.headers.Authorization, "Bearer test-token");
  assert.equal(request.options.credentials, "same-origin");
  assert.equal(storage.reads.includes("refresh_token"), false);
});

test("auth_user JSON 损坏时返回 unknown，且不发请求", async () => {
  const storage = createStorage({
    auth_token: "test-token",
    auth_user: "{not-json",
  });
  let fetchCalled = false;

  assert.deepEqual(readStoredAuth(storage), { status: "unknown", user: null });
  assert.deepEqual(
    await verifyAuthState({
      storage,
      fetchImpl: async () => {
        fetchCalled = true;
      },
    }),
    { status: "unknown", user: null },
  );
  assert.equal(fetchCalled, false);
});

test("接口返回 401 时返回 unknown", async () => {
  const storage = createStorage({
    auth_token: "expired-token",
    auth_user: JSON.stringify(cachedUser),
  });

  const result = await verifyAuthState({
    storage,
    fetchImpl: async () => ({ ok: false, status: 401 }),
  });

  assert.deepEqual(result, { status: "unknown", user: null });
});

test("网络错误时保留有效的缓存用户", async () => {
  const storage = createStorage({
    auth_token: "offline-token",
    auth_user: JSON.stringify(cachedUser),
  });

  const result = await verifyAuthState({
    storage,
    fetchImpl: async () => {
      throw new TypeError("Failed to fetch");
    },
  });

  assert.deepEqual(result, { status: "authenticated", user: cachedUser });
});

test("验证响应不是合法 JSON 时返回 unknown", async () => {
  const storage = createStorage({
    auth_token: "test-token",
    auth_user: JSON.stringify(cachedUser),
  });

  const result = await verifyAuthState({
    storage,
    fetchImpl: async () => ({
      ok: true,
      status: 200,
      json: async () => {
        throw new SyntaxError("invalid JSON");
      },
    }),
  });

  assert.deepEqual(result, { status: "unknown", user: null });
});

test("验证期间 auth_token 变化时丢弃旧请求结果", async () => {
  const storage = createStorage({
    auth_token: "first-token",
    auth_user: JSON.stringify(cachedUser),
  });

  const result = await verifyAuthState({
    storage,
    fetchImpl: async () => {
      storage.setItem("auth_token", "second-token");
      return {
        ok: true,
        status: 200,
        json: async () => ({ code: 0, data: cachedUser }),
      };
    },
  });

  assert.deepEqual(result, { status: "unknown", user: null });
});

test("验证超时会中止请求并返回 unknown", async () => {
  const storage = createStorage({
    auth_token: "slow-token",
    auth_user: JSON.stringify(cachedUser),
  });
  let requestWasAborted = false;

  const result = await verifyAuthState({
    storage,
    timeoutMs: 10,
    fetchImpl: (_url, { signal }) => new Promise((_resolve, reject) => {
      signal.addEventListener("abort", () => {
        requestWasAborted = true;
        const error = new Error("aborted");
        error.name = "AbortError";
        reject(error);
      }, { once: true });
    }),
  });

  assert.deepEqual(result, { status: "unknown", user: null });
  assert.equal(requestWasAborted, true);
});
