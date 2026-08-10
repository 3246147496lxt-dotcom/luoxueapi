import type { ChatModel } from '@/types/chat'

export const DEFAULT_CHAT_PRODUCT_MODEL_ID = 'gpt-5.6-sol'

/**
 * Web Chat exposes a deliberately fixed product catalog. Runtime availability,
 * account access and channel health are validated by the completion endpoint
 * only after the user sends a message.
 */
export const CHAT_PRODUCT_MODELS: readonly Readonly<ChatModel>[] = Object.freeze([
  Object.freeze({
    id: DEFAULT_CHAT_PRODUCT_MODEL_ID,
    display_name: 'GPT-5.6 Sol',
    recommended: true,
    supports_vision: true,
    supports_reasoning_slider: true,
  }),
  Object.freeze({
    id: 'gpt-5.5',
    display_name: 'GPT-5.5',
    recommended: false,
    supports_vision: true,
    supports_reasoning_slider: true,
  }),
  Object.freeze({
    id: 'gpt-5.6-luna',
    display_name: 'GPT-5.6 Luna',
    recommended: false,
    supports_vision: true,
    supports_reasoning_slider: true,
  }),
  Object.freeze({
    id: 'gpt-5.6-terra',
    display_name: 'GPT-5.6 Terra',
    recommended: false,
    supports_vision: true,
    supports_reasoning_slider: true,
  }),
] satisfies ChatModel[])
