/**
 * API Client for Sub2API Backend
 * Central export point for all API modules
 */

// Re-export the HTTP client
export { apiClient } from './client'

// Auth API
export { authAPI, isTotp2FARequired, type LoginResponse } from './auth'

// User APIs
export { keysAPI } from './keys'
export { usageAPI } from './usage'
export { userAPI } from './user'
export { redeemAPI, type RedeemHistoryItem } from './redeem'
export { paymentAPI } from './payment'
export { userGroupsAPI } from './groups'
export { userChannelsAPI } from './channels'
export * as batchImageAPI from './batchImage'
export { totpAPI } from './totp'
export { default as announcementsAPI } from './announcements'
export { channelMonitorUserAPI } from './channelMonitor'
export { requestCustomPageLaunch } from './customPages'
export { desktopAPI } from './desktop'
export {
  publicSkillsAPI,
  getPublicSkill,
  getPublicSkillVersions,
  getSkillVersionDownloadURL,
  listPublicSkills,
  type PublicSkill,
  type PublicSkillVersion,
  type SkillCatalogQuery,
  type SkillCatalogResponse,
  type SkillCategory,
  type SkillFileManifestEntry,
} from './skills'

// Admin APIs
export { adminAPI } from './admin'

// Default export
export { default } from './client'
