import { describe, expect, it } from 'vitest'
import {
  ACCOUNT_DESTINATION_PATH_LIST,
  ACCOUNT_DESTINATION_PATHS,
  ADMIN_ACCOUNT_DESTINATION_SPECS,
  ADMIN_SHELL_DESTINATION_SPECS,
  REGULAR_ACCOUNT_DESTINATION_SPECS,
  REGULAR_SHELL_DESTINATION_SPECS,
  getAccountDestinationPath,
  getAccountDestinationSpecs,
  getShellDestinationSpecs,
  isAccountDestinationPath,
  isShellDestinationAccessible,
  isShellDestinationVisible,
  selectVisibleShellDestinations,
  toShellCapabilityState,
} from '../shellDestinations'

function findSpec(
  specs: typeof REGULAR_SHELL_DESTINATION_SPECS,
  id: (typeof REGULAR_SHELL_DESTINATION_SPECS)[number]['id'],
) {
  const spec = specs.find((candidate) => candidate.id === id)
  if (!spec) throw new Error(`Missing shell destination: ${id}`)
  return spec
}

describe('shellDestinations', () => {
  it('owns every account path once and exposes the same source to consumers', () => {
    expect(ACCOUNT_DESTINATION_PATHS).toEqual({
      pricing: '/pricing',
      subscriptions: '/subscriptions',
      quotaViewer: '/quota-viewer',
      wallet: '/purchase',
      orders: '/orders',
      profile: '/profile',
    })
    expect(ACCOUNT_DESTINATION_PATH_LIST).toEqual(Object.values(ACCOUNT_DESTINATION_PATHS))
    expect(new Set(ACCOUNT_DESTINATION_PATH_LIST).size).toBe(ACCOUNT_DESTINATION_PATH_LIST.length)
    expect(getAccountDestinationPath('wallet')).toBe('/purchase')
    expect(isAccountDestinationPath('/profile')).toBe(true)
    expect(isAccountDestinationPath('/profile/security')).toBe(false)
  })

  it('builds regular and admin specs from the same ordered definitions', () => {
    expect(REGULAR_SHELL_DESTINATION_SPECS.map(({ id }) => id)).toEqual(
      ADMIN_SHELL_DESTINATION_SPECS.map(({ id }) => id),
    )
    expect(REGULAR_SHELL_DESTINATION_SPECS.every(({ audience }) => audience === 'user')).toBe(true)
    expect(ADMIN_SHELL_DESTINATION_SPECS.every(({ audience }) => audience === 'admin')).toBe(true)

    expect(REGULAR_ACCOUNT_DESTINATION_SPECS.map(({ id }) => id)).toEqual([
      'pricing',
      'settings',
      'profile',
    ])
    expect(ADMIN_ACCOUNT_DESTINATION_SPECS.map(({ id }) => id)).toEqual([
      'pricing',
      'settings',
      'profile',
    ])
    expect(
      REGULAR_SHELL_DESTINATION_SPECS
        .filter(({ placement }) => placement === 'workspace')
        .map(({ id }) => id),
    ).toEqual(['subscriptions', 'quotaViewer', 'wallet', 'orders'])
    expect(
      ADMIN_SHELL_DESTINATION_SPECS
        .filter(({ placement }) => placement === 'workspace')
        .map(({ id }) => id),
    ).toEqual(['subscriptions', 'quotaViewer', 'wallet', 'orders'])
    expect(getShellDestinationSpecs('user')).toBe(REGULAR_SHELL_DESTINATION_SPECS)
    expect(getAccountDestinationSpecs('admin')).toBe(ADMIN_ACCOUNT_DESTINATION_SPECS)
  })

  it('keeps simple-mode visibility separate from route access', () => {
    const context = {
      audience: 'user' as const,
      simpleMode: true,
      capabilities: {
        payment: 'enabled' as const,
      },
    }
    const subscriptions = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'subscriptions')
    const quotaViewer = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'quotaViewer')
    const pricing = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'pricing')
    const wallet = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'wallet')
    const settings = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'settings')
    const orders = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'orders')
    const profile = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'profile')

    expect(isShellDestinationVisible(pricing, context)).toBe(false)
    expect(isShellDestinationAccessible(pricing, context)).toBe(false)
    expect(isShellDestinationVisible(subscriptions, context)).toBe(false)
    expect(isShellDestinationAccessible(subscriptions, context)).toBe(false)
    expect(isShellDestinationVisible(quotaViewer, context)).toBe(true)
    expect(isShellDestinationAccessible(quotaViewer, context)).toBe(true)
    expect(isShellDestinationVisible(orders, context)).toBe(false)
    expect(isShellDestinationAccessible(orders, context)).toBe(true)
    expect(isShellDestinationVisible(wallet, context)).toBe(true)
    expect(isShellDestinationAccessible(wallet, context)).toBe(true)
    expect(isShellDestinationVisible(settings, context)).toBe(true)
    expect(isShellDestinationAccessible(settings, context)).toBe(true)
    expect(settings.target).toEqual({
      kind: 'settings-section',
      section: 'general',
    })
    expect(isShellDestinationVisible(profile, context)).toBe(true)
    expect(isShellDestinationAccessible(profile, context)).toBe(true)
  })

  it('resolves opt-out and opt-in capabilities with explicit unknown behavior', () => {
    const orders = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'orders')
    const pricing = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'pricing')
    const models = findSpec(REGULAR_SHELL_DESTINATION_SPECS, 'models')
    const context = {
      audience: 'user' as const,
      simpleMode: false,
    }

    expect(isShellDestinationVisible(orders, context)).toBe(true)
    expect(isShellDestinationVisible(pricing, context)).toBe(true)
    expect(isShellDestinationVisible(models, context)).toBe(false)
    expect(isShellDestinationVisible(orders, {
      ...context,
      capabilities: { payment: 'disabled' },
    })).toBe(false)
    expect(isShellDestinationVisible(pricing, {
      ...context,
      capabilities: { payment: 'disabled' },
    })).toBe(false)
    expect(isShellDestinationVisible(models, {
      ...context,
      capabilities: { 'public-model-catalog': 'enabled' },
    })).toBe(true)
    expect(toShellCapabilityState(true)).toBe('enabled')
    expect(toShellCapabilityState(false)).toBe('disabled')
    expect(toShellCapabilityState(undefined)).toBe('unknown')
    expect(toShellCapabilityState(null)).toBe('unknown')
  })

  it('filters by audience, placement, simple mode, and capability without conflating them', () => {
    const context = {
      audience: 'user' as const,
      simpleMode: true,
      capabilities: {
        payment: 'enabled' as const,
        'public-model-catalog': 'enabled' as const,
      },
    }

    expect(
      selectVisibleShellDestinations(
        REGULAR_SHELL_DESTINATION_SPECS,
        context,
        'account',
      ).map(({ id }) => id),
    ).toEqual(['settings', 'profile'])
    expect(
      selectVisibleShellDestinations(
        REGULAR_SHELL_DESTINATION_SPECS,
        context,
        'workspace',
      ).map(({ id }) => id),
    ).toEqual(['quotaViewer', 'wallet'])
    expect(
      selectVisibleShellDestinations(
        REGULAR_SHELL_DESTINATION_SPECS,
        context,
        'support',
      ).map(({ id }) => id),
    ).toEqual(['documentation', 'models', 'contact'])

    const adminProfile = findSpec(ADMIN_SHELL_DESTINATION_SPECS, 'profile')
    expect(isShellDestinationVisible(adminProfile, context)).toBe(false)
    expect(isShellDestinationAccessible(adminProfile, context)).toBe(false)
  })
})
