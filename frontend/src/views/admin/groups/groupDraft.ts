import type {
  AdminGroup,
  CreateGroupRequest,
  GroupPlatform,
  ModelsListConfig,
  OpenAIMessagesDispatchModelConfig,
  SubscriptionType,
  UpdateGroupRequest,
} from "@/types";
import {
  createDefaultMessagesDispatchFormState,
  messagesDispatchConfigToFormState,
  type MessagesDispatchMappingRow,
} from "../groupsMessagesDispatch";
import {
  isProfitControlPlatform,
  profitDecimalToPercent,
  profitPercentToDecimal,
} from "../groupsProfitControl";

export type GroupDraftMode = "create" | "edit";
export type GroupDraftNumber = number | string | null;

/**
 * The one mutable representation used by both create and edit screens.
 * API-specific null/clear semantics belong to the codec below, not templates.
 */
export interface GroupDraft {
  name: string;
  description: string;
  platform: GroupPlatform;
  rate_multiplier: GroupDraftNumber;
  is_exclusive: boolean;
  status: "active" | "inactive";
  subscription_type: SubscriptionType;
  daily_limit_usd: GroupDraftNumber;
  weekly_limit_usd: GroupDraftNumber;
  monthly_limit_usd: GroupDraftNumber;
  allow_image_generation: boolean;
  allow_batch_image_generation: boolean;
  image_rate_independent: boolean;
  image_rate_multiplier: GroupDraftNumber;
  batch_image_discount_multiplier: GroupDraftNumber;
  batch_image_hold_multiplier: GroupDraftNumber;
  image_price_1k: GroupDraftNumber;
  image_price_2k: GroupDraftNumber;
  image_price_4k: GroupDraftNumber;
  video_rate_independent: boolean;
  video_rate_multiplier: GroupDraftNumber;
  video_price_480p: GroupDraftNumber;
  video_price_720p: GroupDraftNumber;
  video_price_1080p: GroupDraftNumber;
  web_search_price_per_call: GroupDraftNumber;
  peak_rate_enabled: boolean;
  peak_start: string;
  peak_end: string;
  peak_rate_multiplier: GroupDraftNumber;
  profit_control_enabled: boolean;
  profit_min_margin_percent: GroupDraftNumber;
  profit_safety_buffer_percent: GroupDraftNumber;
  claude_code_only: boolean;
  fallback_group_id: number | null;
  fallback_group_id_on_invalid_request: number | null;
  allow_messages_dispatch: boolean;
  opus_mapped_model: string;
  sonnet_mapped_model: string;
  haiku_mapped_model: string;
  exact_model_mappings: MessagesDispatchMappingRow[];
  require_oauth_only: boolean;
  require_privacy_set: boolean;
  model_routing_enabled: boolean;
  supported_model_scopes: string[];
  mcp_xml_inject: boolean;
  copy_accounts_from_group_ids: number[];
  rpm_limit: number;
}

export interface GroupDraftCodecContext {
  modelRouting: Record<string, number[]> | null;
  modelsListConfig: ModelsListConfig;
  supportedModelScopes: string[];
  messagesDispatchModelConfig?: OpenAIMessagesDispatchModelConfig;
}

export function createGroupDraft(): GroupDraft {
  const messages = createDefaultMessagesDispatchFormState();
  return {
    name: "",
    description: "",
    platform: "anthropic",
    rate_multiplier: 1,
    is_exclusive: false,
    status: "active",
    subscription_type: "standard",
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    allow_image_generation: false,
    allow_batch_image_generation: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    batch_image_discount_multiplier: 0.5,
    batch_image_hold_multiplier: 0.6,
    image_price_1k: null,
    image_price_2k: null,
    image_price_4k: null,
    video_rate_independent: false,
    video_rate_multiplier: 1,
    video_price_480p: null,
    video_price_720p: null,
    video_price_1080p: null,
    web_search_price_per_call: null,
    peak_rate_enabled: false,
    peak_start: "",
    peak_end: "",
    peak_rate_multiplier: 1,
    profit_control_enabled: false,
    profit_min_margin_percent: 0,
    profit_safety_buffer_percent: 0,
    claude_code_only: false,
    fallback_group_id: null,
    fallback_group_id_on_invalid_request: null,
    allow_messages_dispatch: messages.allow_messages_dispatch,
    opus_mapped_model: messages.opus_mapped_model,
    sonnet_mapped_model: messages.sonnet_mapped_model,
    haiku_mapped_model: messages.haiku_mapped_model,
    exact_model_mappings: [],
    require_oauth_only: false,
    require_privacy_set: false,
    model_routing_enabled: false,
    supported_model_scopes: ["claude", "gemini_text", "gemini_image"],
    mcp_xml_inject: true,
    copy_accounts_from_group_ids: [],
    rpm_limit: 0,
  };
}

export function groupToDraft(group: AdminGroup): GroupDraft {
  const draft = createGroupDraft();
  const messages = messagesDispatchConfigToFormState(
    group.messages_dispatch_model_config,
  );
  return {
    ...draft,
    name: group.name,
    description: group.description || "",
    platform: group.platform,
    rate_multiplier: group.rate_multiplier,
    is_exclusive: group.is_exclusive,
    status: group.status,
    subscription_type: group.subscription_type || "standard",
    daily_limit_usd: group.daily_limit_usd,
    weekly_limit_usd: group.weekly_limit_usd,
    monthly_limit_usd: group.monthly_limit_usd,
    allow_image_generation: group.allow_image_generation ?? false,
    allow_batch_image_generation: group.allow_batch_image_generation ?? false,
    image_rate_independent: group.image_rate_independent ?? false,
    image_rate_multiplier: group.image_rate_multiplier ?? 1,
    batch_image_discount_multiplier:
      group.batch_image_discount_multiplier ?? 0.5,
    batch_image_hold_multiplier: group.batch_image_hold_multiplier ?? 0.6,
    image_price_1k: group.image_price_1k,
    image_price_2k: group.image_price_2k,
    image_price_4k: group.image_price_4k,
    video_rate_independent: group.video_rate_independent ?? false,
    video_rate_multiplier: group.video_rate_multiplier ?? 1,
    video_price_480p: group.video_price_480p,
    video_price_720p: group.video_price_720p,
    video_price_1080p: group.video_price_1080p,
    web_search_price_per_call: group.web_search_price_per_call ?? null,
    peak_rate_enabled: group.peak_rate_enabled ?? false,
    peak_start: group.peak_start ?? "",
    peak_end: group.peak_end ?? "",
    peak_rate_multiplier: group.peak_rate_multiplier ?? 1,
    profit_control_enabled: group.profit_control_enabled ?? false,
    profit_min_margin_percent: profitDecimalToPercent(
      group.profit_min_margin,
    ),
    profit_safety_buffer_percent: profitDecimalToPercent(
      group.profit_safety_buffer,
    ),
    claude_code_only: group.claude_code_only ?? false,
    fallback_group_id: group.fallback_group_id,
    fallback_group_id_on_invalid_request:
      group.fallback_group_id_on_invalid_request,
    allow_messages_dispatch:
      group.allow_messages_dispatch || messages.allow_messages_dispatch,
    opus_mapped_model: messages.opus_mapped_model,
    sonnet_mapped_model: messages.sonnet_mapped_model,
    haiku_mapped_model: messages.haiku_mapped_model,
    exact_model_mappings: messages.exact_model_mappings,
    require_oauth_only: group.require_oauth_only ?? false,
    require_privacy_set: group.require_privacy_set ?? false,
    model_routing_enabled: group.model_routing_enabled ?? false,
    supported_model_scopes: group.supported_model_scopes
      ? [...group.supported_model_scopes]
      : [...draft.supported_model_scopes],
    mcp_xml_inject: group.mcp_xml_inject ?? true,
    copy_accounts_from_group_ids: [],
    rpm_limit: group.rpm_limit ?? 0,
  };
}

export function replaceGroupDraft(target: GroupDraft, next: GroupDraft): void {
  Object.assign(target, cloneGroupDraft(next));
}

export function cloneGroupDraft(draft: GroupDraft): GroupDraft {
  return {
    ...draft,
    exact_model_mappings: draft.exact_model_mappings.map((row) => ({ ...row })),
    supported_model_scopes: [...draft.supported_model_scopes],
    copy_accounts_from_group_ids: [...draft.copy_accounts_from_group_ids],
  };
}

export function normalizeOptionalGroupLimit(
  value: GroupDraftNumber | undefined,
): number | null {
  if (value === null || value === undefined || value === "") return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

export function normalizeGroupRateMultiplier(
  value: GroupDraftNumber | undefined,
): number {
  if (value === null || value === undefined || value === "") return 1;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 1;
}

function normalizeOptionalPrice(
  value: GroupDraftNumber,
  mode: GroupDraftMode,
): number | null {
  if (value === "" || value === null) return mode === "edit" ? -1 : null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : mode === "edit" ? -1 : null;
}

function commonPayload(
  draft: GroupDraft,
  mode: GroupDraftMode,
  context: GroupDraftCodecContext,
) {
  const allowBatchImage =
    draft.platform === "gemini" &&
    draft.allow_image_generation &&
    draft.allow_batch_image_generation;

  return {
    name: draft.name,
    description: draft.description,
    platform: draft.platform,
    rate_multiplier: normalizeGroupRateMultiplier(draft.rate_multiplier),
    is_exclusive: draft.is_exclusive,
    subscription_type: draft.subscription_type,
    daily_limit_usd: normalizeOptionalGroupLimit(draft.daily_limit_usd),
    weekly_limit_usd: normalizeOptionalGroupLimit(draft.weekly_limit_usd),
    monthly_limit_usd: normalizeOptionalGroupLimit(draft.monthly_limit_usd),
    allow_image_generation: draft.allow_image_generation,
    allow_batch_image_generation: allowBatchImage,
    image_rate_independent: draft.image_rate_independent,
    image_rate_multiplier: normalizeGroupRateMultiplier(
      draft.image_rate_multiplier,
    ),
    batch_image_discount_multiplier: allowBatchImage
      ? normalizeGroupRateMultiplier(draft.batch_image_discount_multiplier)
      : 0.5,
    batch_image_hold_multiplier: allowBatchImage
      ? normalizeGroupRateMultiplier(draft.batch_image_hold_multiplier)
      : 0.6,
    image_price_1k: normalizeOptionalPrice(draft.image_price_1k, mode),
    image_price_2k: normalizeOptionalPrice(draft.image_price_2k, mode),
    image_price_4k: normalizeOptionalPrice(draft.image_price_4k, mode),
    video_rate_independent: draft.video_rate_independent,
    video_rate_multiplier: normalizeGroupRateMultiplier(
      draft.video_rate_multiplier,
    ),
    video_price_480p: normalizeOptionalPrice(draft.video_price_480p, mode),
    video_price_720p: normalizeOptionalPrice(draft.video_price_720p, mode),
    video_price_1080p: normalizeOptionalPrice(draft.video_price_1080p, mode),
    web_search_price_per_call: normalizeOptionalPrice(
      draft.web_search_price_per_call,
      mode,
    ),
    peak_rate_enabled: draft.peak_rate_enabled,
    peak_start: draft.peak_start,
    peak_end: draft.peak_end,
    peak_rate_multiplier: normalizeGroupRateMultiplier(
      draft.peak_rate_multiplier,
    ),
    profit_control_enabled:
      isProfitControlPlatform(draft.platform) && draft.profit_control_enabled,
    profit_min_margin: profitPercentToDecimal(
      draft.profit_min_margin_percent,
    ),
    profit_safety_buffer: profitPercentToDecimal(
      draft.profit_safety_buffer_percent,
    ),
    claude_code_only: draft.claude_code_only,
    fallback_group_id:
      mode === "edit" && draft.fallback_group_id === null
        ? 0
        : draft.fallback_group_id,
    fallback_group_id_on_invalid_request:
      mode === "edit" && draft.fallback_group_id_on_invalid_request === null
        ? 0
        : draft.fallback_group_id_on_invalid_request,
    allow_messages_dispatch: draft.allow_messages_dispatch,
    messages_dispatch_model_config:
      draft.platform === "openai"
        ? context.messagesDispatchModelConfig
        : undefined,
    model_routing: context.modelRouting,
    model_routing_enabled: draft.model_routing_enabled,
    models_list_config: context.modelsListConfig,
    supported_model_scopes: context.supportedModelScopes,
    mcp_xml_inject: draft.mcp_xml_inject,
    copy_accounts_from_group_ids: [...draft.copy_accounts_from_group_ids],
    rpm_limit: draft.rpm_limit,
    require_oauth_only: draft.require_oauth_only,
    require_privacy_set: draft.require_privacy_set,
  };
}

export function serializeCreateGroupDraft(
  draft: GroupDraft,
  context: GroupDraftCodecContext,
): CreateGroupRequest {
  return commonPayload(draft, "create", context);
}

export function serializeUpdateGroupDraft(
  draft: GroupDraft,
  context: GroupDraftCodecContext,
): UpdateGroupRequest {
  return {
    ...commonPayload(draft, "edit", context),
    status: draft.status,
  };
}
