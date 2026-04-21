import { Fragment, useCallback, useEffect, useMemo, useState, type CSSProperties } from 'react'
import { ChevronDown, ChevronRight, Filter, LoaderCircle, X } from 'lucide-react'
import {
  getUserSettings,
  listSupportedFiatCurrencies,
  listTransactions,
  upsertUserSettings,
  type AggregatedTransaction,
  type MoneyLeg,
  type SupportedFiatCurrency,
} from '../api/aggregationService'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { toErrorMessage } from '../utils/errors'

const UTC_DATE_TIME_FORMATTER = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'UTC',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})

const DEFAULT_PAGE_SIZE = 30

interface TransactionsFilters {
  fromDate: string
  toDate: string
  source: string
  kind: string
  importId: string
}

const DEFAULT_FILTERS: TransactionsFilters = {
  fromDate: '',
  toDate: '',
  source: '',
  kind: '',
  importId: '',
}

function formatUtcTimestamp(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return `${UTC_DATE_TIME_FORMATTER.format(date)} UTC`
}

function toUtcRange(filters: TransactionsFilters): { dateFrom?: string; dateTo?: string } {
  const dateFrom = filters.fromDate ? new Date(`${filters.fromDate}T00:00:00.000Z`).toISOString() : undefined

  const dateTo = filters.toDate ? new Date(`${filters.toDate}T00:00:00.000Z`) : undefined
  if (dateTo) {
    dateTo.setUTCDate(dateTo.getUTCDate() + 1)
  }

  return {
    dateFrom,
    dateTo: dateTo?.toISOString(),
  }
}

function countActiveFilters(filters: TransactionsFilters): number {
  let count = 0
  if (filters.fromDate !== '') count += 1
  if (filters.toDate !== '') count += 1
  if (filters.source !== '') count += 1
  if (filters.kind !== '') count += 1
  if (filters.importId.trim() !== '') count += 1
  return count
}

const KIND_BADGE_PALETTE: Array<{ background: string; color: string }> = [
  { background: 'rgba(59, 130, 246, 0.18)', color: '#93c5fd' },
  { background: 'rgba(16, 185, 129, 0.18)', color: '#86efac' },
  { background: 'rgba(245, 158, 11, 0.18)', color: '#fcd34d' },
  { background: 'rgba(168, 85, 247, 0.18)', color: '#d8b4fe' },
  { background: 'rgba(244, 114, 182, 0.18)', color: '#f9a8d4' },
  { background: 'rgba(34, 197, 94, 0.18)', color: '#bbf7d0' },
  { background: 'rgba(239, 68, 68, 0.18)', color: '#fca5a5' },
  { background: 'rgba(20, 184, 166, 0.18)', color: '#99f6e4' },
]

function kindBadgeStyle(kind: string): CSSProperties {
  const normalized = kind.trim().toUpperCase()
  const hash = Array.from(normalized).reduce((acc, char) => acc + char.charCodeAt(0), 0)
  const palette = KIND_BADGE_PALETTE[hash % KIND_BADGE_PALETTE.length]

  return {
    backgroundColor: palette.background,
    color: palette.color,
  }
}

function renderLeg(leg: MoneyLeg | undefined, tone: 'default' | 'muted' = 'default') {
  if (!leg) {
    return null
  }

  return (
    <div className={tone === 'muted' ? 'text-muted-foreground' : 'text-foreground'}>
      <span className="font-mono">{leg.crypto_amount}</span>
      <span className="ml-1 text-muted-foreground">{leg.symbol}</span>
      {!leg.fiat_amount && leg.error?.code ? <span className="ml-2 text-muted-foreground">| {leg.error.code}</span> : null}
    </div>
  )
}

function resolveValuation(tx: AggregatedTransaction, displayFiatCode: string | null): string {
  const leg = tx.inMoney ?? tx.outMoney ?? tx.feeMoney
  if (!leg?.fiat_amount) {
    return '—'
  }

  return displayFiatCode ? `${leg.fiat_amount} ${displayFiatCode}` : leg.fiat_amount
}

export function TransactionsPage() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [supportedFiat, setSupportedFiat] = useState<SupportedFiatCurrency[]>([])
  const [activeUserFiat, setActiveUserFiat] = useState<string | null>(null)
  const [activeUserTimezone, setActiveUserTimezone] = useState<string | null>(null)
  const [isFiatLoading, setIsFiatLoading] = useState(false)
  const [isFiatUpdating, setIsFiatUpdating] = useState(false)
  const [fiatError, setFiatError] = useState<string | null>(null)

  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<TransactionsFilters>(DEFAULT_FILTERS)

  const [transactions, setTransactions] = useState<AggregatedTransaction[]>([])
  const [page, setPage] = useState(0)
  const [pageTokens, setPageTokens] = useState<string[]>([''])
  const [nextPageToken, setNextPageToken] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [expandedRow, setExpandedRow] = useState<string | null>(null)

  const loadFiatContext = useCallback(async (): Promise<void> => {
    if (!session) {
      return
    }

    setIsFiatLoading(true)
    setFiatError(null)

    try {
      const [currencies, userSettings] = await Promise.all([listSupportedFiatCurrencies(), getUserSettings()])
      setSupportedFiat(currencies)
      setActiveUserFiat(userSettings.fiatCurrency)
      setActiveUserTimezone(userSettings.timezone)
    } catch (loadError) {
      setFiatError(toErrorMessage(loadError, 'Failed to load fiat currency context.'))
      setSupportedFiat([])
      setActiveUserFiat(null)
      setActiveUserTimezone(null)
    } finally {
      setIsFiatLoading(false)
    }
  }, [session])

  useEffect(() => {
    void loadFiatContext()
  }, [loadFiatContext])

  const fetchTransactions = useCallback(
    async (pageToken: string, pageIndex: number): Promise<void> => {
      if (!session || !activeUserFiat) {
        return
      }

      setIsLoading(true)
      setError(null)

      try {
        const { dateFrom, dateTo } = toUtcRange(filters)
        const response = await listTransactions({
          pageSize: DEFAULT_PAGE_SIZE,
          pageToken,
          dateFrom,
          dateTo,
          importId: filters.importId.trim() === '' ? undefined : filters.importId.trim(),
          source: filters.source === '' ? undefined : filters.source,
          kind: filters.kind === '' ? undefined : filters.kind,
          targetFiat: activeUserFiat,
        })

        setTransactions(response.items)
        setNextPageToken(response.nextPageToken ?? '')
        setPage(pageIndex)
        setExpandedRow(null)
      } catch (fetchError) {
        setError(toErrorMessage(fetchError, 'Failed to load transactions.'))
        setTransactions([])
        setNextPageToken('')
        setPage(0)
      } finally {
        setIsLoading(false)
      }
    },
    [session, activeUserFiat, filters],
  )

  useEffect(() => {
    if (!session || isFiatLoading || fiatError || !activeUserFiat) {
      return
    }

    setPageTokens([''])
    void fetchTransactions('', 0)
  }, [session, isFiatLoading, fiatError, activeUserFiat, filters, fetchTransactions])

  const handlePrevPage = async (): Promise<void> => {
    if (page === 0) {
      return
    }

    const prevPage = page - 1
    const token = pageTokens[prevPage] ?? ''
    await fetchTransactions(token, prevPage)
  }

  const handleNextPage = async (): Promise<void> => {
    if (nextPageToken === '') {
      return
    }

    const nextPage = page + 1
    setPageTokens((prev) => {
      const next = [...prev]
      next[nextPage] = nextPageToken
      return next
    })
    await fetchTransactions(nextPageToken, nextPage)
  }

  const handleGotoPage = async (pageIndex: number): Promise<void> => {
    if (pageIndex === page) {
      return
    }

    if (pageIndex < pageTokens.length) {
      const token = pageTokens[pageIndex] ?? ''
      await fetchTransactions(token, pageIndex)
      return
    }

    if (pageIndex === page + 1 && nextPageToken !== '') {
      await handleNextPage()
    }
  }

  const handleFiatChange = useCallback(
    async (nextFiatCode: string): Promise<void> => {
      setError(null)

      if (!session || !activeUserTimezone || nextFiatCode.trim() === '' || nextFiatCode === activeUserFiat) {
        return
      }

      setIsFiatUpdating(true)
      try {
        const updatedSettings = await upsertUserSettings({
          fiatCurrency: nextFiatCode,
          timezone: activeUserTimezone,
        })

        setActiveUserFiat(updatedSettings.fiatCurrency)
        setActiveUserTimezone(updatedSettings.timezone)
        notifications.success('User currency updated', `Default fiat currency: ${updatedSettings.fiatCurrency}`)
      } catch (updateError) {
        const message = toErrorMessage(updateError, 'Failed to update user currency.')
        setError(message)
        notifications.error('Failed to update user currency', message)
      } finally {
        setIsFiatUpdating(false)
      }
    },
    [session, activeUserTimezone, activeUserFiat, notifications],
  )

  const sourceOptions = useMemo(() => {
    const set = new Set<string>()
    transactions.forEach((tx) => {
      if (tx.source.trim() !== '') {
        set.add(tx.source)
      }
    })
    return Array.from(set).sort((left, right) => left.localeCompare(right))
  }, [transactions])

  const kindOptions = useMemo(() => {
    const set = new Set<string>()
    transactions.forEach((tx) => {
      if (tx.kind.trim() !== '') {
        set.add(tx.kind)
      }
    })
    return Array.from(set).sort((left, right) => left.localeCompare(right))
  }, [transactions])

  const activeFilterCount = useMemo(() => countActiveFilters(filters), [filters])
  const visiblePageIndices = useMemo(() => pageTokens.map((_, index) => index), [pageTokens])
  const showNextPageButton = nextPageToken !== '' && page === pageTokens.length - 1

  return (
    <div className="max-w-[1400px]">
      <div className="mb-8 flex items-start justify-between">
        <div className="flex flex-col gap-2">
          <h2 className="text-foreground">Transactions</h2>
          <p className="text-muted-foreground text-sm">Review and analyze normalized transaction data across all exchanges</p>
        </div>
      </div>

      {fiatError ? (
        <div className="bg-surface rounded-xl border border-[var(--status-failed)]/30 p-5 mb-6" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <p className="text-sm text-[var(--status-failed)]">{fiatError}</p>
        </div>
      ) : null}

      {error ? (
        <div className="bg-surface rounded-xl border border-[var(--status-failed)]/30 p-5 mb-6" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <p className="text-sm text-[var(--status-failed)]">{error}</p>
        </div>
      ) : null}

      <div className="bg-surface rounded-lg border border-border p-4 mb-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setShowFilters((prev) => !prev)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-all ${
              showFilters || activeFilterCount > 0
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted text-foreground hover:bg-muted/80'
            }`}
          >
            <Filter className="w-4 h-4" />
            Filters
            {activeFilterCount > 0 ? (
              <span className="ml-1 px-1.5 py-0.5 text-xs bg-primary-foreground/20 rounded-full">{activeFilterCount}</span>
            ) : null}
          </button>

          {activeFilterCount > 0 ? (
            <button
              type="button"
              onClick={() => setFilters(DEFAULT_FILTERS)}
              className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1"
            >
              <X className="w-3 h-3" />
              Clear filters
            </button>
          ) : null}
        </div>

        <div className="flex items-center gap-3">
          <label className="text-sm text-muted-foreground">Valuation:</label>
          <select
            value={activeUserFiat ?? ''}
            onChange={(event) => void handleFiatChange(event.target.value)}
            disabled={isLoading || isFiatLoading || isFiatUpdating || supportedFiat.length === 0}
            className="px-3 py-1.5 bg-input-background border border-input-border rounded-lg text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          >
            {supportedFiat.map((fiat) => (
              <option key={fiat.code} value={fiat.code}>
                {fiat.code}
              </option>
            ))}
          </select>
        </div>
      </div>

      {showFilters ? (
        <div className="bg-surface rounded-lg border border-border p-6 mb-6 space-y-4" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4">
            <div>
              <label className="block text-xs text-muted-foreground mb-1.5">Date From</label>
              <input
                type="date"
                value={filters.fromDate}
                onChange={(event) => setFilters((prev) => ({ ...prev, fromDate: event.target.value }))}
                className="w-full px-3 py-2 bg-input-background border border-input-border rounded-lg text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1.5">Date To</label>
              <input
                type="date"
                value={filters.toDate}
                onChange={(event) => setFilters((prev) => ({ ...prev, toDate: event.target.value }))}
                className="w-full px-3 py-2 bg-input-background border border-input-border rounded-lg text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1.5">Exchange</label>
              <select
                value={filters.source}
                onChange={(event) => setFilters((prev) => ({ ...prev, source: event.target.value }))}
                className="w-full px-3 py-2 bg-input-background border border-input-border rounded-lg text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option value="">All</option>
                {sourceOptions.map((source) => (
                  <option key={source} value={source}>
                    {source}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1.5">Transaction Kind</label>
              <select
                value={filters.kind}
                onChange={(event) => setFilters((prev) => ({ ...prev, kind: event.target.value }))}
                className="w-full px-3 py-2 bg-input-background border border-input-border rounded-lg text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option value="">All</option>
                {kindOptions.map((kind) => (
                  <option key={kind} value={kind}>
                    {kind}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1.5">Import ID</label>
              <input
                type="text"
                value={filters.importId}
                onChange={(event) => setFilters((prev) => ({ ...prev, importId: event.target.value }))}
                placeholder="imp_..."
                className="w-full px-3 py-2 bg-input-background border border-input-border rounded-lg text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
          </div>
        </div>
      ) : null}

      {isFiatLoading && !activeUserFiat ? (
        <div className="bg-surface rounded-xl border border-border p-8 mb-6 flex items-center justify-center gap-3" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <LoaderCircle className="w-5 h-5 animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">Loading fiat currency context...</span>
        </div>
      ) : null}

      <div className="bg-surface rounded-xl border border-border overflow-hidden" style={{ boxShadow: 'var(--shadow-md)' }}>
        {isLoading && transactions.length === 0 ? (
          <div className="px-6 py-8 flex items-center justify-center gap-3">
            <LoaderCircle className="w-5 h-5 animate-spin text-primary" />
            <span className="text-sm text-muted-foreground">Fetching aggregated transactions...</span>
          </div>
        ) : null}

        {!isLoading && transactions.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <p className="text-muted-foreground">No transactions found</p>
            <p className="text-sm text-muted-foreground mt-1">Try adjusting the optional filters or load more recent imports.</p>
          </div>
        ) : null}

        {transactions.length > 0 ? (
          <>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[900px]">
                <thead className="bg-surface-secondary border-b border-border">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider w-10"></th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Timestamp</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Exchange</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Kind</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">In</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Out</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Fee</th>
                    <th className="px-4 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">Valuation ({activeUserFiat ?? '—'})</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {transactions.map((tx) => (
                    <Fragment key={tx.txId}>
                      <tr className="hover:bg-surface-secondary transition-colors cursor-pointer" onClick={() => setExpandedRow(expandedRow === tx.txId ? null : tx.txId)}>
                        <td className="px-4 py-3">
                          {expandedRow === tx.txId ? (
                            <ChevronDown className="w-4 h-4 text-muted-foreground" />
                          ) : (
                            <ChevronRight className="w-4 h-4 text-muted-foreground" />
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-foreground font-mono">{formatUtcTimestamp(tx.timeUtc)}</td>
                        <td className="px-4 py-3 text-sm text-foreground">{tx.source}</td>
                        <td className="px-4 py-3">
                          <span className="inline-block px-2 py-0.5 text-xs font-medium rounded-full" style={kindBadgeStyle(tx.kind)}>
                            {tx.kind}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm">{renderLeg(tx.inMoney)}</td>
                        <td className="px-4 py-3 text-sm">{renderLeg(tx.outMoney)}</td>
                        <td className="px-4 py-3 text-sm">{renderLeg(tx.feeMoney, 'muted')}</td>
                        <td className="px-4 py-3 text-sm text-right font-mono text-foreground">{resolveValuation(tx, activeUserFiat)}</td>
                      </tr>

                      {expandedRow === tx.txId ? (
                        <tr>
                          <td colSpan={8} className="px-4 py-4 bg-surface-secondary/50">
                            <div className="grid grid-cols-2 gap-4 text-sm">
                              {tx.txHash ? (
                                <div>
                                  <span className="text-muted-foreground">TX Hash:</span>
                                  <p className="font-mono text-xs text-foreground mt-1 break-all">{tx.txHash}</p>
                                </div>
                              ) : null}
                              <div>
                                <span className="text-muted-foreground">Fingerprint:</span>
                                <p className="font-mono text-xs text-foreground mt-1 break-all">{tx.txFingerprint}</p>
                              </div>
                              <div>
                                <span className="text-muted-foreground">Import ID:</span>
                                <p className="font-mono text-xs text-foreground mt-1 break-all">{tx.importId}</p>
                              </div>
                              {tx.positionId ? (
                                <div>
                                  <span className="text-muted-foreground">Position ID:</span>
                                  <p className="font-mono text-xs text-foreground mt-1 break-all">{tx.positionId}</p>
                                </div>
                              ) : null}
                              {tx.orderId ? (
                                <div>
                                  <span className="text-muted-foreground">Order ID:</span>
                                  <p className="font-mono text-xs text-foreground mt-1 break-all">{tx.orderId}</p>
                                </div>
                              ) : null}
                              {tx.contractSymbol ? (
                                <div>
                                  <span className="text-muted-foreground">Contract Symbol:</span>
                                  <p className="font-mono text-xs text-foreground mt-1 break-all">{tx.contractSymbol}</p>
                                </div>
                              ) : null}
                              {tx.note ? (
                                <div className="col-span-2">
                                  <span className="text-muted-foreground">Notes:</span>
                                  <p className="text-foreground mt-1">{tx.note}</p>
                                </div>
                              ) : null}
                            </div>
                          </td>
                        </tr>
                      ) : null}
                    </Fragment>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="border-t border-border px-6 py-4 flex items-center justify-between bg-surface-secondary/30 gap-4 flex-wrap">
              <div className="text-sm text-muted-foreground">
                Showing <span className="font-medium text-foreground">{transactions.length}</span> transactions on page{' '}
                <span className="font-medium text-foreground">{page + 1}</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => void handlePrevPage()}
                  disabled={page === 0 || isLoading}
                  className="px-3 py-1.5 text-sm border border-border rounded-lg hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Previous
                </button>
                <div className="flex items-center gap-1">
                  {visiblePageIndices.map((pageIndex) => (
                    <button
                      key={pageIndex}
                      type="button"
                      onClick={() => void handleGotoPage(pageIndex)}
                      className={`w-8 h-8 text-sm rounded-lg transition-colors ${
                        page === pageIndex ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
                      }`}
                    >
                      {pageIndex + 1}
                    </button>
                  ))}
                  {showNextPageButton ? (
                    <button
                      type="button"
                      onClick={() => void handleNextPage()}
                      className="w-8 h-8 text-sm rounded-lg transition-colors hover:bg-muted"
                    >
                      {page + 2}
                    </button>
                  ) : null}
                </div>
                <button
                  type="button"
                  onClick={() => void handleNextPage()}
                  disabled={nextPageToken === '' || isLoading}
                  className="px-3 py-1.5 text-sm border border-border rounded-lg hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Next
                </button>
              </div>
            </div>
          </>
        ) : null}
      </div>
    </div>
  )
}
