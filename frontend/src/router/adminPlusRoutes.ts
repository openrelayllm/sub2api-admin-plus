import type { RouteLocationGeneric, RouteRecordRaw } from 'vue-router'

export const ADMIN_HOME = '/admin/dashboard'

const adminMeta = (title: string, extra: Record<string, unknown> = {}) => ({
  requiresAuth: true,
  requiresAdmin: true,
  title,
  ...extra
})

const redirectWithQuery = (path: string) => (to: RouteLocationGeneric) => ({
  path,
  query: to.query
})

export const adminPlusRoutes: RouteRecordRaw[] = [
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/setup/SetupWizardView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Setup'
    }
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/LoginView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Login',
      titleKey: 'home.login'
    }
  },
  {
    path: '/',
    redirect: ADMIN_HOME
  },
  {
    path: '/admin',
    redirect: ADMIN_HOME
  },
  {
    path: ADMIN_HOME,
    name: 'AdminDashboard',
    component: () => import('@/views/admin/DashboardView.vue'),
    meta: adminMeta('Admin Dashboard', {
      titleKey: 'admin.dashboard.title',
      descriptionKey: 'admin.dashboard.description'
    })
  },
  {
    path: '/admin/ops',
    name: 'AdminOps',
    component: () => import('@/views/admin/ops/OpsDashboard.vue'),
    meta: adminMeta('Ops Monitoring', {
      titleKey: 'admin.ops.title',
      descriptionKey: 'admin.ops.description'
    })
  },
  {
    path: '/admin/suppliers',
    name: 'AdminPlusSuppliers',
    component: () => import('@/views/admin/operations/SuppliersView.vue'),
    meta: adminMeta('供应商管理')
  },
  {
    path: '/admin/supplier-bindings',
    name: 'AdminPlusSupplierBindings',
    component: () => import('@/views/admin/operations/SupplierAccountsView.vue'),
    meta: adminMeta('账号/Key 绑定')
  },
  {
    path: '/admin/collection/scheduler',
    name: 'AdminPlusCollectionScheduler',
    component: () => import('@/views/admin/operations/SchedulerView.vue'),
    meta: adminMeta('任务调度')
  },
  {
    path: '/admin/collection/plugin-tasks',
    name: 'AdminPlusPluginTasks',
    component: () => import('@/views/admin/operations/SchedulerView.vue'),
    meta: adminMeta('插件任务')
  },
  {
    path: '/admin/collection/sessions',
    name: 'AdminPlusCollectionSessions',
    component: () => import('@/views/admin/operations/SuppliersView.vue'),
    meta: adminMeta('采集会话')
  },
  {
    path: '/admin/monitoring/rates',
    name: 'AdminPlusRates',
    component: () => import('@/views/admin/operations/RatesView.vue'),
    meta: adminMeta('费率')
  },
  {
    path: '/admin/monitoring/balances',
    name: 'AdminPlusBalances',
    component: () => import('@/views/admin/operations/BalancesView.vue'),
    meta: adminMeta('余额')
  },
  {
    path: '/admin/monitoring/health',
    name: 'AdminPlusHealth',
    component: () => import('@/views/admin/operations/HealthView.vue'),
    meta: adminMeta('健康探测')
  },
  {
    path: '/admin/monitoring/account-runtime',
    name: 'AdminPlusAccountRuntime',
    component: () => import('@/views/admin/operations/AccountRuntimeView.vue'),
    meta: adminMeta('并发运行态')
  },
  {
    path: '/admin/monitoring/announcements',
    name: 'AdminPlusAnnouncements',
    component: () => import('@/views/admin/operations/AnnouncementsView.vue'),
    meta: adminMeta('公告')
  },
  {
    path: '/admin/finance/billing',
    name: 'AdminPlusSupplierBilling',
    component: () => import('@/views/admin/operations/BillingReconciliationView.vue'),
    meta: adminMeta('供应商账单')
  },
  {
    path: '/admin/finance/local-usage',
    name: 'AdminPlusLocalUsage',
    component: () => import('@/views/admin/operations/LocalUsageView.vue'),
    meta: adminMeta('本地用量')
  },
  {
    path: '/admin/finance/reconciliation',
    name: 'AdminPlusReconciliation',
    component: () => import('@/views/admin/operations/BillingReconciliationView.vue'),
    meta: adminMeta('对账结果')
  },
  {
    path: '/admin/automation/actions',
    name: 'AdminPlusActions',
    component: () => import('@/views/admin/operations/ActionRecommendationsView.vue'),
    meta: adminMeta('动作建议')
  },
  {
    path: '/admin/automation/notifications',
    name: 'AdminPlusNotifications',
    component: () => import('@/views/admin/operations/NotificationsView.vue'),
    meta: adminMeta('通知记录')
  },
  {
    path: '/admin/automation/audits',
    name: 'AdminPlusExecutionAudits',
    component: () => import('@/views/admin/operations/ActionRecommendationsView.vue'),
    meta: adminMeta('执行审计')
  },
  {
    path: '/admin/operations',
    redirect: '/admin/suppliers'
  },
  {
    path: '/admin/operations/suppliers',
    redirect: redirectWithQuery('/admin/suppliers')
  },
  {
    path: '/admin/operations/supplier-accounts',
    redirect: redirectWithQuery('/admin/supplier-bindings')
  },
  {
    path: '/admin/operations/account-runtime',
    redirect: redirectWithQuery('/admin/monitoring/account-runtime')
  },
  {
    path: '/admin/operations/rates',
    redirect: redirectWithQuery('/admin/monitoring/rates')
  },
  {
    path: '/admin/operations/balances',
    redirect: redirectWithQuery('/admin/monitoring/balances')
  },
  {
    path: '/admin/operations/health',
    redirect: redirectWithQuery('/admin/monitoring/health')
  },
  {
    path: '/admin/operations/announcements',
    redirect: redirectWithQuery('/admin/monitoring/announcements')
  },
  {
    path: '/admin/operations/scheduler',
    redirect: redirectWithQuery('/admin/collection/scheduler')
  },
  {
    path: '/admin/operations/extension-tasks',
    redirect: redirectWithQuery('/admin/collection/plugin-tasks')
  },
  {
    path: '/admin/operations/billing',
    redirect: redirectWithQuery('/admin/finance/reconciliation')
  },
  {
    path: '/admin/operations/actions',
    redirect: redirectWithQuery('/admin/automation/actions')
  },
  {
    path: '/admin/operations/notifications',
    redirect: redirectWithQuery('/admin/automation/notifications')
  },
  {
    path: '/admin/settings',
    name: 'AdminSettings',
    component: () => import('@/views/admin/SettingsView.vue'),
    meta: adminMeta('System Settings', {
      titleKey: 'admin.settings.title',
      descriptionKey: 'admin.settings.description'
    })
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFoundView.vue'),
    meta: {
      requiresAuth: false,
      title: '404 Not Found'
    }
  }
]
