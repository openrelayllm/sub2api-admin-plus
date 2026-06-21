import type { RouteRecordRaw } from 'vue-router'
import { describe, expect, it } from 'vitest'
import { adminPlusRoutes } from '@/router/adminPlusRoutes'

function collectRoutePaths(routes: RouteRecordRaw[]): string[] {
  const paths: string[] = []

  const visit = (route: RouteRecordRaw) => {
    paths.push(route.path)
    route.children?.forEach(visit)
  }

  routes.forEach(visit)
  return paths
}

describe('adminPlusRoutes', () => {
  it('只暴露 Admin Plus MVP0 当前路由', () => {
    expect(collectRoutePaths(adminPlusRoutes)).toEqual([
      '/setup',
      '/login',
      '/',
      '/admin',
      '/admin/dashboard',
      '/admin/ops',
      '/admin/suppliers',
      '/admin/supplier-bindings',
      '/admin/collection/scheduler',
      '/admin/collection/plugin-tasks',
      '/admin/collection/sessions',
      '/admin/monitoring/rates',
      '/admin/monitoring/balances',
      '/admin/monitoring/health',
      '/admin/monitoring/account-runtime',
      '/admin/monitoring/announcements',
      '/admin/finance/billing',
      '/admin/finance/local-usage',
      '/admin/finance/reconciliation',
      '/admin/automation/actions',
      '/admin/automation/notifications',
      '/admin/automation/audits',
      '/admin/operations',
      '/admin/operations/suppliers',
      '/admin/operations/supplier-accounts',
      '/admin/operations/account-runtime',
      '/admin/operations/rates',
      '/admin/operations/balances',
      '/admin/operations/health',
      '/admin/operations/announcements',
      '/admin/operations/scheduler',
      '/admin/operations/extension-tasks',
      '/admin/operations/billing',
      '/admin/operations/actions',
      '/admin/operations/notifications',
      '/admin/settings',
      '/:pathMatch(.*)*'
    ])
  })

  it('不回流 Sub2API 用户端、支付、OAuth 和旧后台页面', () => {
    const paths = collectRoutePaths(adminPlusRoutes)
    const deadPaths = [
      '/register',
      '/forgot-password',
      '/reset-password',
      '/verify-email',
      '/oauth/callback',
      '/auth/wechat/callback',
      '/keys',
      '/usage',
      '/profile',
      '/payment',
      '/orders',
      '/channels',
      '/admin/users',
      '/admin/accounts',
      '/admin/channels',
      '/admin/groups',
      '/admin/payment',
      '/admin/subscriptions',
      '/admin/redeem',
      '/admin/backup',
      '/admin/operations/promotions'
    ]

    for (const deadPath of deadPaths) {
      expect(paths).not.toContain(deadPath)
    }
  })

  it('后台业务页面必须要求管理员身份', () => {
    const adminRoutes = adminPlusRoutes.filter((route) =>
      [
      '/admin/dashboard',
      '/admin/ops',
        '/admin/suppliers',
        '/admin/supplier-bindings',
        '/admin/collection/scheduler',
        '/admin/collection/plugin-tasks',
        '/admin/collection/sessions',
        '/admin/monitoring/rates',
        '/admin/monitoring/balances',
        '/admin/monitoring/health',
        '/admin/monitoring/account-runtime',
        '/admin/monitoring/announcements',
        '/admin/finance/billing',
        '/admin/finance/local-usage',
        '/admin/finance/reconciliation',
        '/admin/automation/actions',
        '/admin/automation/notifications',
        '/admin/automation/audits',
        '/admin/settings'
      ].includes(route.path)
    )

    expect(adminRoutes).toHaveLength(19)
    for (const route of adminRoutes) {
      expect(route.meta?.requiresAuth).toBe(true)
      expect(route.meta?.requiresAdmin).toBe(true)
    }
  })

  it('旧 operations 入口只作为兼容重定向', () => {
    const redirects = new Map(
      [
        ['/admin/operations', '/admin/suppliers'],
        ['/admin/operations/suppliers', '/admin/suppliers'],
        ['/admin/operations/supplier-accounts', '/admin/supplier-bindings'],
        ['/admin/operations/account-runtime', '/admin/monitoring/account-runtime'],
        ['/admin/operations/rates', '/admin/monitoring/rates'],
        ['/admin/operations/balances', '/admin/monitoring/balances'],
        ['/admin/operations/health', '/admin/monitoring/health'],
        ['/admin/operations/announcements', '/admin/monitoring/announcements'],
        ['/admin/operations/scheduler', '/admin/collection/scheduler'],
        ['/admin/operations/extension-tasks', '/admin/collection/plugin-tasks'],
        ['/admin/operations/billing', '/admin/finance/reconciliation'],
        ['/admin/operations/actions', '/admin/automation/actions'],
        ['/admin/operations/notifications', '/admin/automation/notifications']
      ]
    )

    for (const [path, target] of redirects) {
      const route = adminPlusRoutes.find((item) => item.path === path)
      expect(route?.component).toBeUndefined()
      if (typeof route?.redirect === 'function') {
        expect(route.redirect({ query: { q: 'abc' } } as never)).toEqual({ path: target, query: { q: 'abc' } })
      } else {
        expect(route?.redirect).toBe(target)
      }
    }
  })
})
