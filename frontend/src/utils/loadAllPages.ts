export interface PagedResponse<T> {
  items: T[]
  pages?: number
}

export async function loadAllPagedItems<T>(
  loadPage: (page: number, pageSize: number) => Promise<PagedResponse<T>>,
  pageSize = 1000
): Promise<T[]> {
  const items: T[] = []
  let page = 1

  while (true) {
    const result = await loadPage(page, pageSize)
    items.push(...result.items)

    if (!result.pages || page >= result.pages) {
      break
    }

    page += 1
  }

  return items
}
