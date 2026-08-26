// The backend stores margin values as decimal fractions while the admin form
// presents percentages. Round at the storage precision boundary so values do
// not grow floating-point tails across edit/save cycles.
export const profitPercentToDecimal = (
  value: number | string | null | undefined,
): number => {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) return 0;
  return Math.round(parsed * 100) / 10000;
};

export const profitDecimalToPercent = (
  value: number | null | undefined,
): number => {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) return 0;
  return Math.round(parsed * 1e6) / 1e4;
};

export type ProfitControlFormState = {
  platform: string;
  profit_control_enabled: boolean;
  profit_min_margin_percent: number | string | null;
  profit_safety_buffer_percent: number | string | null;
};

const PROFIT_CONTROL_PLATFORMS = new Set([
  "openai",
  "anthropic",
  "gemini",
  "grok",
  "antigravity",
]);

export const isProfitControlPlatform = (platform: string): boolean =>
  PROFIT_CONTROL_PLATFORMS.has(platform);

export const validateProfitControlFormState = (
  form: ProfitControlFormState,
): string | null => {
  if (!isProfitControlPlatform(form.platform) || !form.profit_control_enabled) {
    return null;
  }

  const margin = Number(form.profit_min_margin_percent || 0);
  const buffer = Number(form.profit_safety_buffer_percent || 0);
  const storedMargin = profitPercentToDecimal(margin);
  const storedBuffer = profitPercentToDecimal(buffer);
  if (
    !Number.isFinite(margin) ||
    margin < 0 ||
    margin >= 100 ||
    storedMargin >= 1
  ) {
    return "marginRangeError";
  }
  if (
    !Number.isFinite(buffer) ||
    buffer < 0 ||
    buffer >= 100 ||
    storedBuffer >= 1
  ) {
    return "bufferRangeError";
  }
  if (storedMargin + storedBuffer >= 1) return "sumTooHigh";
  return null;
};
