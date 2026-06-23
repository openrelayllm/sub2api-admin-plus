import type { ExtensionTaskType } from '@/api/admin/adminPlus'

export type SupplierAutomationAction =
  | 'full_collect'
  | 'login_session'
  | 'fetch_balance'
  | 'fetch_groups'
  | 'fetch_rates'
  | 'reconcile_costs'
  | 'check_channels'

const supplierActionTasks: Record<SupplierAutomationAction, ExtensionTaskType[]> = {
  full_collect: ['fetch_balance', 'fetch_groups', 'fetch_rates', 'fetch_announcements', 'fetch_usage_costs'],
  fetch_balance: ['fetch_balance'],
  fetch_groups: ['fetch_groups'],
  fetch_rates: ['fetch_rates'],
  reconcile_costs: ['fetch_balance', 'fetch_announcements', 'fetch_usage_costs'],
  check_channels: ['check_supplier_channels'],
  login_session: []
}

export function supplierActionTaskTypes(action: SupplierAutomationAction): ExtensionTaskType[] {
  return supplierActionTasks[action] || []
}

export function supplierActionLabel(action: SupplierAutomationAction): string {
  return {
    full_collect: '一键采集',
    login_session: '直登会话',
    fetch_balance: '余额同步',
    fetch_groups: '分组同步',
    fetch_rates: '倍率同步',
    reconcile_costs: '成本对账',
    check_channels: '渠道检测'
  }[action] || action
}

export function supplierActionKey(supplierId: number, action: SupplierAutomationAction): string {
  return `${supplierId}:${action}`
}

export function checklistActionForKey(key: string): SupplierAutomationAction | null {
  const action = {
    session: 'login_session',
    balance: 'fetch_balance',
    groups: 'fetch_groups',
    rates: 'fetch_rates',
    recharge_rate: 'fetch_rates',
    recharge_entry: 'reconcile_costs',
    billing: 'reconcile_costs',
    channels: 'check_channels'
  }[key] as SupplierAutomationAction | undefined
  return action || null
}

export function checklistManualActionLabel(key: string): string {
  return {
    basic: '去供应商管理补充',
    url: '去供应商管理补充',
    schedule: '在供应商渠道中加入调度'
  }[key] || '暂无自动动作'
}
