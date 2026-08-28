import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { resolveLegacyPersonalSettingsRedirect } from '@/navigation/personalSettingsRoute'

export const userRoutes: RouteRecordRaw[] = [
  // ==================== User Routes ====================
  {
    path: '/',
    redirect: '/home'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/views/user/DashboardView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Dashboard',
      titleKey: 'dashboard.title',
      descriptionKey: 'dashboard.welcomeMessage'
    }
  },
  {
    path: '/models',
    name: 'UserModelCatalog',
    component: () => import('@/views/public/ModelCatalogView.vue'),
    props: { embedded: true },
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Model Center',
      titleKey: 'nav.modelCenter',
      descriptionKey: 'modelCatalog.workspaceDescription'
    }
  },
  {
    path: '/desktop/authorize',
    name: 'DesktopAuthorize',
    component: () => import('@/views/user/DesktopAuthorizeView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Desktop Authorization',
      titleKey: 'desktopAuthorization.pageTitle'
    }
  },
  {
    path: '/quota-viewer/authorize',
    name: 'QuotaViewerAuthorize',
    component: () => import('@/views/user/QuotaViewerAuthorizeView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Points Viewer Authorization',
      titleKey: 'quotaViewerAuthorization.pageTitle'
    }
  },
  {
    path: '/quota-viewer/devices',
    name: 'QuotaViewerDevices',
    component: () => import('@/views/user/QuotaViewerDevicesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Points Viewer Devices',
      titleKey: 'quotaViewerDevices.title',
      descriptionKey: 'quotaViewerDevices.description'
    }
  },
  {
    path: '/desktop/devices',
    name: 'DesktopDevices',
    component: () => import('@/views/user/DesktopDevicesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Desktop Devices',
      titleKey: 'desktopDevices.title',
      descriptionKey: 'desktopDevices.description'
    }
  },
  {
    path: '/chat',
    name: 'Chat',
    component: () => import('@/views/user/ChatView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'GPT Chat',
      titleKey: 'chat.title',
      descriptionKey: 'chat.subtitle',
      shellMode: 'chat'
    }
  },
  {
    path: '/library',
    name: 'Library',
    component: () => import('@/views/user/LibraryView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Library',
      titleKey: 'library.title',
      shellMode: 'chat'
    }
  },
  {
    path: '/keys',
    name: 'Keys',
    component: () => import('@/views/user/KeysView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'API Keys',
      titleKey: 'keys.title',
      descriptionKey: 'keys.description'
    }
  },
  {
    path: '/batch-image',
    name: 'BatchImageGuide',
    alias: '/docs/batch-image',
    component: () => import('@/views/user/BatchImageGuideView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Batch Image Guide',
      titleKey: 'batchImageGuide.title',
      descriptionKey: 'batchImageGuide.description'
    }
  },
  {
    path: '/usage',
    name: 'Usage',
    component: () => import('@/views/user/UsageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Usage Records',
      titleKey: 'usage.title',
      descriptionKey: 'usage.description'
    }
  },
  {
    path: '/redeem',
    name: 'Redeem',
    component: () => import('@/views/user/RedeemView.vue'),
    beforeEnter: () => ({ path: '/purchase', hash: '#redeem' }),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Redeem Code',
      titleKey: 'redeem.title',
      descriptionKey: 'redeem.description'
    }
  },
  {
    path: '/affiliate',
    name: 'Affiliate',
    component: () => import('@/views/user/AffiliateView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Affiliate',
      titleKey: 'affiliate.title',
      descriptionKey: 'affiliate.description'
    }
  },
  {
    path: '/available-channels',
    name: 'UserAvailableChannels',
    component: () => import('@/views/user/AvailableChannelsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Available Channels',
      titleKey: 'availableChannels.title',
      descriptionKey: 'availableChannels.description'
    }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: { render: () => null },
    beforeEnter: (to) => resolveLegacyPersonalSettingsRedirect(
      to,
      useAuthStore().isAdmin ? 'admin' : 'user',
      'account',
      'profile'
    ),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Profile',
      titleKey: 'profile.title',
      descriptionKey: 'profile.description'
    }
  },
  {
    path: '/profile/security',
    name: 'ProfileSecurityLegacy',
    component: { render: () => null },
    beforeEnter: (to) => resolveLegacyPersonalSettingsRedirect(
      to,
      useAuthStore().isAdmin ? 'admin' : 'user',
      'security'
    ),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Profile Security',
      titleKey: 'profile.securityTitle'
    }
  },
  {
    path: '/profile/connections',
    name: 'ProfileConnectionsLegacy',
    component: { render: () => null },
    beforeEnter: (to) => resolveLegacyPersonalSettingsRedirect(
      to,
      useAuthStore().isAdmin ? 'admin' : 'user',
      'account',
      'connections'
    ),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Profile Connections',
      titleKey: 'profile.authBindings.title'
    }
  },
  {
    path: '/settings/profile',
    name: 'SettingsProfileLegacy',
    component: { render: () => null },
    beforeEnter: (to) => resolveLegacyPersonalSettingsRedirect(
      to,
      useAuthStore().isAdmin ? 'admin' : 'user',
      'account',
      'connections'
    ),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Profile Connections',
      titleKey: 'profile.authBindings.title'
    }
  },
  {
    path: '/subscriptions',
    name: 'Subscriptions',
    component: () => import('@/views/user/SubscriptionsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Balance & Membership',
      titleKey: 'userSubscriptions.title',
      descriptionKey: 'userSubscriptions.description'
    }
  },
  {
    path: '/purchase',
    name: 'PurchaseSubscription',
    component: () => import('@/views/user/PaymentView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Purchase Subscription',
      titleKey: 'nav.buySubscription',
      descriptionKey: 'purchase.description'
    }
  },
  {
    path: '/pricing',
    name: 'UserSubscriptionPricing',
    component: () => import('@/views/user/PricingView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      requiresPayment: true,
      title: 'Subscription Plans',
      titleKey: 'pricing.title',
      descriptionKey: 'pricing.description'
    }
  },
  {
    path: '/orders',
    name: 'OrderList',
    component: () => import('@/views/user/UserOrdersView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'My Orders',
      titleKey: 'nav.myOrders',
      requiresPayment: true
    }
  },
  {
    path: '/payment/qrcode',
    name: 'PaymentQRCode',
    component: () => import('@/views/user/PaymentQRCodeView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Payment',
      titleKey: 'payment.qr.scanToPay',
      requiresPayment: true
    }
  },
  {
    path: '/payment/result',
    name: 'PaymentResult',
    component: () => import('@/views/user/PaymentResultView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Payment Result',
      titleKey: 'payment.result.success',
      requiresPayment: false
    }
  },
  {
    path: '/payment/stripe',
    name: 'StripePayment',
    component: () => import('@/views/user/StripePaymentView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Stripe Payment',
      titleKey: 'payment.stripePay',
      requiresPayment: false
    }
  },
  {
    path: '/payment/airwallex',
    name: 'AirwallexPayment',
    component: () => import('@/views/user/AirwallexPaymentView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Airwallex Payment',
      titleKey: 'payment.airwallexPay',
      requiresPayment: false
    }
  },
  {
    path: '/payment/stripe-popup',
    name: 'StripePopup',
    component: () => import('@/views/user/StripePopupView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Payment',
      requiresPayment: false
    }
  },
  {
    path: '/custom/:id',
    name: 'CustomPage',
    component: () => import('@/views/user/CustomPageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Custom Page',
      titleKey: 'customPage.title',
    }
  },
  {
    path: '/monitor',
    name: 'ChannelStatus',
    component: () => import('@/views/user/ChannelStatusView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Channel Status',
      titleKey: 'nav.channelStatus'
    }
  },
]
