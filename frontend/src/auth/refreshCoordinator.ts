type RefreshHandler = () => Promise<boolean>

let refreshHandler: RefreshHandler | null = null
let inflightRefresh: Promise<boolean> | null = null

export function setRefreshHandler(handler: RefreshHandler | null): void {
  refreshHandler = handler
}

export async function runRefreshFlow(): Promise<boolean> {
  if (!refreshHandler) {
    return false
  }

  if (inflightRefresh) {
    return inflightRefresh
  }

  inflightRefresh = refreshHandler().finally(() => {
    inflightRefresh = null
  })

  return inflightRefresh
}
