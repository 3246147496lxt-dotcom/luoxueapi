import type { ChatModel } from '@/types/chat'

export const DEFAULT_CHAT_PRODUCT_MODEL_ID = 'gpt-5.6-sol'

/**
 * Web Chat keeps a fixed product order and presentation. Runtime model and
 * reasoning capabilities are deliberately not declared here; ChatView overlays
 * the authenticated `/chat/models` capability response so this file cannot
 * become a conflicting second source of truth.
 */
export const CHAT_PRODUCT_MODELS: readonly Readonly<ChatModel>[] = Object.freeze([
  Object.freeze({
    id: DEFAULT_CHAT_PRODUCT_MODEL_ID,
    display_name: 'GPT-5.6 Sol',
    recommended: true,
    supports_vision: true,
  }),
  Object.freeze({
    id: 'gpt-5.5',
    display_name: 'GPT-5.5',
    recommended: false,
    supports_vision: true,
  }),
  Object.freeze({
    id: 'gpt-5.6-luna',
    display_name: 'GPT-5.6 Luna',
    recommended: false,
    supports_vision: true,
  }),
  Object.freeze({
    id: 'gpt-5.6-terra',
    display_name: 'GPT-5.6 Terra',
    recommended: false,
    supports_vision: true,
  }),
] satisfies ChatModel[])
