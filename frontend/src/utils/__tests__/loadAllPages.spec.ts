import { describe, expect, it, vi } from 'vitest'

import { loadAllPagedItems } from '@/utils/loadAllPages'

describe('loadAllPagedItems', () => {
  it('loads every page until the server reports the last page', async () => {
    const loadPage = vi.fn()
      .mockResolvedValueOnce({ items: [{ id: 1 }], pages: 3 })
      .mockResolvedValueOnce({ items: [{ id: 2 }], pages: 3 })
      .mockResolvedValueOnce({ items: [{ id: 3 }], pages: 3 })

    await expect(loadAllPagedItems(loadPage)).resolves.toEqual([{ id: 1 }, { id: 2 }, { id: 3 }])
    expect(loadPage).toHaveBeenNthCalledWith(1, 1, 1000)
    expect(loadPage).toHaveBeenNthCalledWith(2, 2, 1000)
    expect(loadPage).toHaveBeenNthCalledWith(3, 3, 1000)
  })

  it('supports custom page sizes', async () => {
    const loadPage = vi.fn().mockResolvedValueOnce({ items: [{ id: 1 }], pages: 1 })

    await expect(loadAllPagedItems(loadPage, 200)).resolves.toEqual([{ id: 1 }])
    expect(loadPage).toHaveBeenCalledWith(1, 200)
  })
})
