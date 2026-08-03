/**
 * Shell-level destinations shared by workspace navigation, support controls,
 * and the account surface.
 *
 * Keep route ownership here: consumers should select specs by audience and
 * placement instead of repeating destination paths or feature fallback rules.
 */
import type { PersonalSettingsSection } from './personalSettingsRoute'

export const ACCOUNT_DESTINATION_PATHS = Object.freeze({
  pricing: '/pricing',
  subscriptions: '/subscriptions',
  quotaViewer: '/quota-viewer',
  wallet: '/purchase',
  orders: '/orders',
  profile: '/profile',
} as const)

export type AccountDestinationId = keyof typeof ACCOUNT_DESTINATION_PATHS
export type AccountDestinationPath = (typeof ACCOUNT_DESTINATION_PATHS)[AccountDestinationId]

export const ACCOUNT_DESTINATION_PATH_LIST = Object.freeze(
  Object.values(ACCOUNT_DESTINATION_PATHS),
) as readonly AccountDestinationPath[]

export const ACCOUNT_DESTINATION_PATH_SET: ReadonlySet<AccountDestinationPath> = new Set(
  ACCOUNT_DESTINATION_PATH_LIST,
)

export type ShellAudience = 'user' | 'admin'
export type ShellDestinationPlacement = 'account' | 'workspace' | 'support'
export type ShellSimpleVisibility = 'visible' | 'hidden'
export type ShellSimpleAccess = 'allowed' | 'blocked'
export type ShellCapabilityState = 'enabled' | 'disabled' | 'unknown'
export type ShellUnknownCapabilityPolicy = 'allow' | 'deny'
export type ShellCapabilityKey = 'payment' | 'public-model-catalog'
export type ShellConfiguredHrefSource = 'contact' | 'documentation'
export type ShellSupportDestinationId = 'models' | 'contact' | 'documentation'
export type ShellSettingsDestinationId = 'settings'
export type ShellDestinationId =
  | AccountDestinationId
  | ShellSettingsDestinationId
  | ShellSupportDestinationId

export interface ShellSimpleModePolicy {
  readonly visibility: ShellSimpleVisibility
  readonly access: ShellSimpleAccess
}

export interface ShellCapabilityRequirement {
  readonly key: ShellCapabilityKey
  readonly whenUnknown: ShellUnknownCapabilityPolicy
}

export type ShellDestinationTarget =
  | Readonly<{
      kind: 'route'
      path: string
    }>
  | Readonly<{
      kind: 'href'
      href: string
    }>
  | Readonly<{
      kind: 'configured-href'
      source: ShellConfiguredHrefSource
    }>
  | Readonly<{
      kind: 'settings-section'
      section: PersonalSettingsSection
    }>

export interface ShellDestinationDefinition {
  readonly id: ShellDestinationId
  readonly labelKey: string
  readonly placement: ShellDestinationPlacement
  readonly target: ShellDestinationTarget
  readonly simpleMode: ShellSimpleModePolicy
  readonly capability?: ShellCapabilityRequirement
}

export interface ShellDestinationSpec extends ShellDestinationDefinition {
  readonly audience: ShellAudience
}

export type ShellCapabilityStates = Readonly<
  Partial<Record<ShellCapabilityKey, ShellCapabilityState>>
>

export interface ShellDestinationContext {
  readonly audience: ShellAudience
  readonly simpleMode: boolean
  readonly capabilities?: ShellCapabilityStates
}

const VISIBLE_IN_SIMPLE_MODE = Object.freeze({
  visibility: 'visible',
  access: 'allowed',
} as const satisfies ShellSimpleModePolicy)

export const ACCOUNT_DESTINATION_DEFINITIONS = [
  {
    id: 'pricing',
    labelKey: 'accountDock.upgrade',
    placement: 'account',
    target: {
      kind: 'route',
      path: ACCOUNT_DESTINATION_PATHS.pricing,
    },
    simpleMode: {
      visibility: 'hidden',
      access: 'blocked',
    },
    capability: {
      key: 'payment',
      whenUnknown: 'allow',
    },
  },
  {
    id: 'subscriptions',
    labelKey: 'nav.mySubscriptions',
    placement: 'workspace',
    target: {
      kind: 'route',
      path: ACCOUNT_DESTINATION_PATHS.subscriptions,
    },
    simpleMode: {
      visibility: 'hidden',
      access: 'blocked',
    },
  },
  {
    id: 'quotaViewer',
    labelKey: 'quotaViewerLanding.meta.title',
    placement: 'workspace',
    target: {
      kind: 'route',
      path: ACCOUNT_DESTINATION_PATHS.quotaViewer,
    },
    simpleMode: VISIBLE_IN_SIMPLE_MODE,
  },
  {
    id: 'wallet',
    labelKey: 'nav.buySubscription',
    placement: 'workspace',
    target: {
      kind: 'route',
      path: ACCOUNT_DESTINATION_PATHS.wallet,
    },
    simpleMode: VISIBLE_IN_SIMPLE_MODE,
  },
  {
    id: 'settings',
    labelKey: 'accountDock.settings',
    placement: 'account',
    target: {
      kind: 'settings-section',
      section: 'general',
    },
    simpleMode: VISIBLE_IN_SIMPLE_MODE,
  },
  {
    id: 'orders',
    labelKey: 'nav.myOrders',
    placement: 'workspace',
    target: {
      kind: 'route',
      path: ACCOUNT_DESTINATION_PATHS.orders,
    },
    simpleMode: {
      visibility: 'hidden',
      access: 'allowed',
    },
    capability: {
      key: 'payment',
      whenUnknown: 'allow',
    },
  },
  {
    id: 'profile',
    labelKey: 'nav.profile',
    placement: 'account',
    target: {
      kind: 'route',
      path: ACCOUNT_DESTINATION_PATHS.profile,
    },
    simpleMode: VISIBLE_IN_SIMPLE_MODE,
  },
] as const satisfies readonly ShellDestinationDefinition[]

export const SUPPORT_DESTINATION_DEFINITIONS = [
  {
    id: 'documentation',
    labelKey: 'nav.docsTutorial',
    placement: 'support',
    target: {
      kind: 'configured-href',
      source: 'documentation',
    },
    simpleMode: VISIBLE_IN_SIMPLE_MODE,
  },
  {
    id: 'models',
    labelKey: 'nav.modelCatalog',
    placement: 'support',
    target: {
      kind: 'href',
      href: '/models.html',
    },
    simpleMode: VISIBLE_IN_SIMPLE_MODE,
    capability: {
      key: 'public-model-catalog',
      whenUnknown: 'deny',
    },
  },
  {
    id: 'contact',
    labelKey: 'nav.contactUs',
    placement: 'support',
    target: {
      kind: 'configured-href',
      source: 'contact',
    },
    simpleMode: VISIBLE_IN_SIMPLE_MODE,
  },
] as const satisfies readonly ShellDestinationDefinition[]

export const SHELL_DESTINATION_DEFINITIONS: readonly ShellDestinationDefinition[] =
  Object.freeze([
    ...ACCOUNT_DESTINATION_DEFINITIONS,
    ...SUPPORT_DESTINATION_DEFINITIONS,
  ])

function createShellDestinationSpecs(
  audience: ShellAudience,
): readonly ShellDestinationSpec[] {
  const specs: ShellDestinationSpec[] = SHELL_DESTINATION_DEFINITIONS.map(
    (definition) => ({
      ...definition,
      audience,
    }),
  )
  return Object.freeze(specs)
}

function selectPlacement(
  specs: readonly ShellDestinationSpec[],
  placement: ShellDestinationPlacement,
): readonly ShellDestinationSpec[] {
  return Object.freeze(specs.filter((spec) => spec.placement === placement))
}

export const REGULAR_SHELL_DESTINATION_SPECS = createShellDestinationSpecs('user')
export const ADMIN_SHELL_DESTINATION_SPECS = createShellDestinationSpecs('admin')

export const REGULAR_ACCOUNT_DESTINATION_SPECS = selectPlacement(
  REGULAR_SHELL_DESTINATION_SPECS,
  'account',
)

export const ADMIN_ACCOUNT_DESTINATION_SPECS = selectPlacement(
  ADMIN_SHELL_DESTINATION_SPECS,
  'account',
)

export function getAccountDestinationPath(
  id: AccountDestinationId,
): AccountDestinationPath {
  return ACCOUNT_DESTINATION_PATHS[id]
}

export function isAccountDestinationPath(path: string): path is AccountDestinationPath {
  return ACCOUNT_DESTINATION_PATH_SET.has(path as AccountDestinationPath)
}

export function toShellCapabilityState(
  value: boolean | null | undefined,
): ShellCapabilityState {
  if (value === true) return 'enabled'
  if (value === false) return 'disabled'
  return 'unknown'
}

export function resolveShellCapability(
  requirement: ShellCapabilityRequirement | undefined,
  states: ShellCapabilityStates = {},
): boolean {
  if (!requirement) return true

  const state = states[requirement.key] ?? 'unknown'
  if (state === 'enabled') return true
  if (state === 'disabled') return false
  return requirement.whenUnknown === 'allow'
}

export function matchesShellDestinationAudience(
  spec: ShellDestinationSpec,
  audience: ShellAudience,
): boolean {
  return spec.audience === audience
}

export function isShellDestinationVisible(
  spec: ShellDestinationSpec,
  context: ShellDestinationContext,
): boolean {
  if (!matchesShellDestinationAudience(spec, context.audience)) return false
  if (context.simpleMode && spec.simpleMode.visibility === 'hidden') return false
  return resolveShellCapability(spec.capability, context.capabilities)
}

export function isShellDestinationAccessible(
  spec: ShellDestinationSpec,
  context: ShellDestinationContext,
): boolean {
  if (!matchesShellDestinationAudience(spec, context.audience)) return false
  if (context.simpleMode && spec.simpleMode.access === 'blocked') return false
  return resolveShellCapability(spec.capability, context.capabilities)
}

export function getShellDestinationSpecs(
  audience: ShellAudience,
): readonly ShellDestinationSpec[] {
  return audience === 'admin'
    ? ADMIN_SHELL_DESTINATION_SPECS
    : REGULAR_SHELL_DESTINATION_SPECS
}

export function getAccountDestinationSpecs(
  audience: ShellAudience,
): readonly ShellDestinationSpec[] {
  return audience === 'admin'
    ? ADMIN_ACCOUNT_DESTINATION_SPECS
    : REGULAR_ACCOUNT_DESTINATION_SPECS
}

export function selectVisibleShellDestinations(
  specs: readonly ShellDestinationSpec[],
  context: ShellDestinationContext,
  placement?: ShellDestinationPlacement,
): readonly ShellDestinationSpec[] {
  return specs.filter(
    (spec) =>
      (placement === undefined || spec.placement === placement)
      && isShellDestinationVisible(spec, context),
  )
}
