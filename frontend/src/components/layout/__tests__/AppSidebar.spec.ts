import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar Admin Plus navigation', () => {
  it('只保留 Admin Plus MVP0 后台导航入口', () => {
    expect(componentSource).toContain("path: '/admin/dashboard'")
    expect(componentSource).toContain("path: '/admin/ops'")
    expect(componentSource).toContain("path: '/admin/settings'")

    expect(componentSource).not.toContain("path: '/admin/users'")
    expect(componentSource).not.toContain("path: '/admin/accounts'")
    expect(componentSource).not.toContain("path: '/admin/channels'")
    expect(componentSource).not.toContain("path: '/admin/payment'")
    expect(componentSource).not.toContain("path: '/keys'")
    expect(componentSource).not.toContain("path: '/payment'")
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
