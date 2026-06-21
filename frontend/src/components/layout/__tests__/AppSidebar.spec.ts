import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar Admin Plus navigation', () => {
  it('只展示 Admin Plus P4 current 导航入口', () => {
    expect(componentSource).toContain("path: '/admin/dashboard'")
    expect(componentSource).toContain("path: '/admin/ops'")
    expect(componentSource).toContain("path: '/admin/settings'")
    expect(componentSource).toContain("path: '/admin/suppliers'")
    expect(componentSource).toContain("path: '/admin/supplier-bindings'")
    expect(componentSource).toContain("path: '/admin/collection/scheduler'")
    expect(componentSource).toContain("path: '/admin/collection/plugin-tasks'")
    expect(componentSource).toContain("path: '/admin/collection/sessions'")
    expect(componentSource).toContain("path: '/admin/monitoring/rates'")
    expect(componentSource).toContain("path: '/admin/finance/local-usage'")
    expect(componentSource).toContain("path: '/admin/automation/audits'")

    expect(componentSource).not.toContain("path: '/admin/users'")
    expect(componentSource).not.toContain("path: '/admin/accounts'")
    expect(componentSource).not.toContain("path: '/admin/channels'")
    expect(componentSource).not.toContain("path: '/admin/payment'")
    expect(componentSource).not.toContain("path: '/admin/operations/")
    expect(componentSource).not.toContain("path: '/admin/operations/extension-tasks'")
    expect(componentSource).not.toContain("path: '/keys'")
    expect(componentSource).not.toContain("path: '/payment'")
  })

  it('将业务页面收敛到 P4 五组导航下', () => {
    const suppliersGroupMatch = componentSource.match(/label: '供应商',[\s\S]*?children: \[([\s\S]*?)\n {4}\]/)
    const collectionGroupMatch = componentSource.match(/label: '采集监控',[\s\S]*?children: \[([\s\S]*?)\n {4}\]/)
    const monitorGroupMatch = componentSource.match(/label: '运营监控',[\s\S]*?children: \[([\s\S]*?)\n {4}\]/)
    const financeGroupMatch = componentSource.match(/label: '财务对账',[\s\S]*?children: \[([\s\S]*?)\n {4}\]/)
    const automationGroupMatch = componentSource.match(/label: '自动化',[\s\S]*?children: \[([\s\S]*?)\n {4}\]/)

    expect(suppliersGroupMatch?.[1]).toContain("label: '供应商管理'")
    expect(suppliersGroupMatch?.[1]).toContain("label: '账号/Key 绑定'")
    expect(collectionGroupMatch?.[1]).toContain("label: '任务调度'")
    expect(collectionGroupMatch?.[1]).toContain("label: '插件任务'")
    expect(collectionGroupMatch?.[1]).toContain("label: '采集会话'")
    expect(monitorGroupMatch).not.toBeNull()
    expect(monitorGroupMatch?.[1]).toContain("label: t('nav.ops')")
    expect(monitorGroupMatch?.[1]).toContain("label: '费率'")
    expect(monitorGroupMatch?.[1]).toContain("label: '余额'")
    expect(monitorGroupMatch?.[1]).toContain("label: '健康与并发'")
    expect(monitorGroupMatch?.[1]).toContain("label: '公告'")
    expect(financeGroupMatch?.[1]).toContain("label: '供应商账单'")
    expect(financeGroupMatch?.[1]).toContain("label: '本地用量'")
    expect(financeGroupMatch?.[1]).toContain("label: '对账结果'")
    expect(automationGroupMatch?.[1]).toContain("label: '动作建议'")
    expect(automationGroupMatch?.[1]).toContain("label: '通知记录'")
    expect(automationGroupMatch?.[1]).toContain("label: '执行审计'")
    expect(componentSource.match(/path: '\/admin\/ops', label: t\('nav\.ops'\)/g)).toHaveLength(1)
  })

  it('二级导航在同组跳转后保持展开，点击同一级才关闭', () => {
    expect(componentSource).toContain('const expandedGroupPath = ref<string | null>(null)')
    expect(componentSource).toContain('const activeGroupPath = computed(() => {')
    expect(componentSource).toContain('expandedGroupPath.value = nextPath')
    expect(componentSource).toContain('@click="toggleNavGroup(item.path)"')
    expect(componentSource).toContain('isNavGroupExpanded(item.path)')
    expect(componentSource).toContain('expandedGroupPath.value === path')
    expect(componentSource).toContain('expandedGroupPath.value = null')
    expect(componentSource).toContain('@click="handleTopLevelLinkClick"')
    expect(componentSource).toContain('void router.push(path)')
    expect(componentSource).not.toContain('isNavGroupExpanded(item.path) || isNavItemActive(item)')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})
