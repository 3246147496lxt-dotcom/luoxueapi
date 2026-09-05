import { describe, expect, it } from "vitest";

import {
  profitDecimalToPercent,
  profitPercentToDecimal,
  validateProfitControlFormState,
  type ProfitControlFormState,
} from "../groupsProfitControl";

const formState = (
  overrides: Partial<ProfitControlFormState> = {},
): ProfitControlFormState => ({
  platform: "openai",
  profit_control_enabled: true,
  profit_min_margin_percent: 30,
  profit_safety_buffer_percent: 0,
  ...overrides,
});

describe("profit control percentage conversion", () => {
  it("converts admin percentages to the backend decimal precision", () => {
    expect(profitPercentToDecimal(30)).toBe(0.3);
    expect(profitPercentToDecimal(33.33)).toBe(0.3333);
    expect(profitPercentToDecimal(33.333)).toBe(0.3333);
    expect(profitPercentToDecimal(0.005)).toBe(0.0001);
  });

  it("converts backend decimals without float-tail noise", () => {
    expect(profitDecimalToPercent(0.3)).toBe(30);
    expect(profitDecimalToPercent(0.3333)).toBe(33.33);
    expect(profitDecimalToPercent(null)).toBe(0);
  });

  it("normalizes empty, invalid, and non-positive input to zero", () => {
    expect(profitPercentToDecimal("")).toBe(0);
    expect(profitPercentToDecimal("invalid")).toBe(0);
    expect(profitPercentToDecimal(-1)).toBe(0);
    expect(profitDecimalToPercent(-0.3)).toBe(0);
  });
});

describe("validateProfitControlFormState", () => {
  it("accepts valid configurations for all supported platforms", () => {
    for (const platform of [
      "openai",
      "anthropic",
      "gemini",
      "grok",
      "zhipu",
      "deepseek",
      "antigravity",
    ]) {
      expect(validateProfitControlFormState(formState({ platform }))).toBeNull();
    }
    expect(
      validateProfitControlFormState(
        formState({
          profit_min_margin_percent: 60,
          profit_safety_buffer_percent: 39.99,
        }),
      ),
    ).toBeNull();
  });

  it("skips disabled and unsupported configurations", () => {
    expect(
      validateProfitControlFormState(
        formState({
          profit_control_enabled: false,
          profit_min_margin_percent: 200,
        }),
      ),
    ).toBeNull();
    expect(
      validateProfitControlFormState(
        formState({ platform: "composite", profit_min_margin_percent: 200 }),
      ),
    ).toBeNull();
  });

  it("rejects out-of-range inputs and totals reaching 100 percent", () => {
    expect(
      validateProfitControlFormState(
        formState({ profit_min_margin_percent: 100 }),
      ),
    ).toBe("marginRangeError");
    expect(
      validateProfitControlFormState(
        formState({ profit_safety_buffer_percent: -0.1 }),
      ),
    ).toBe("bufferRangeError");
    expect(
      validateProfitControlFormState(
        formState({
          profit_min_margin_percent: 60,
          profit_safety_buffer_percent: 40,
        }),
      ),
    ).toBe("sumTooHigh");
    expect(
      validateProfitControlFormState(
        formState({ profit_min_margin_percent: 99.999 }),
      ),
    ).toBe("marginRangeError");
    expect(
      validateProfitControlFormState(
        formState({
          profit_min_margin_percent: 60,
          profit_safety_buffer_percent: 39.999,
        }),
      ),
    ).toBe("sumTooHigh");
  });
});
