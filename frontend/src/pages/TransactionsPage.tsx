import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
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
import { PageHeader } from '../components/layout/PageHeader'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { EmptyState } from '../components/states/EmptyState'
import { ErrorState } from '../components/states/ErrorState'
import { LoadingState } from '../components/states/LoadingState'
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

interface MoneyLegDetailsProps {
  title: string
  leg: MoneyLeg | undefined
  displayFiatCode: string | null
}

const DEFAULT_FILTERS: TransactionsFilters = {
  fromDate: '',
  toDate: '',
  source: 'all',
  kind: 'all',
  importId: '',
}

function formatUtcTimestamp(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return `${UTC_DATE_TIME_FORMATTER.format(date)} UTC`
}

function truncateMiddle(value: string | undefined, head = 8, tail = 6): string {
  if (!value) {
    return '—'
  }

  if (value.length <= head + tail + 3) {
    return value
  }

  return `${value.slice(0, head)}...${value.slice(-tail)}`
}

function formatMoneyLegSummary(leg: MoneyLeg | undefined, displayFiatCode: string | null): string {
  if (!leg) {
    return '—'
  }

  const basePart = `${leg.symbol} ${leg.crypto_amount}`

  if (leg.fiat_amount) {
    const fiatPart = displayFiatCode ? `${leg.fiat_amount} ${displayFiatCode}` : leg.fiat_amount
    return `${basePart} | ${fiatPart}`
  }

  if (leg.error?.code) {
    return `${basePart} | ${leg.error.code}`
  }

  return basePart
}

function toUtcRange(filters: TransactionsFilters): { dateFrom?: string; dateTo?: string } {
  const dateFrom = filters.fromDate
    ? new Date(`${filters.fromDate}T00:00:00.000Z`).toISOString()
    : undefined

  const dateTo = filters.toDate
    ? new Date(`${filters.toDate}T00:00:00.000Z`)
    : undefined
  if (dateTo) {
    dateTo.setUTCDate(dateTo.getUTCDate() + 1)
  }

  return {
    dateFrom,
    dateTo: dateTo?.toISOString(),
  }
}

function filtersEqual(left: TransactionsFilters, right: TransactionsFilters): boolean {
  return (
    left.fromDate === right.fromDate &&
    left.toDate === right.toDate &&
    left.source === right.source &&
    left.kind === right.kind &&
    left.importId === right.importId
  )
}

function countActiveFilters(filters: TransactionsFilters): number {
  let count = 0

  if (filters.fromDate) {
    count += 1
  }
  if (filters.toDate) {
    count += 1
  }
  if (filters.importId.trim() !== '') {
    count += 1
  }
  if (filters.source !== 'all') {
    count += 1
  }
  if (filters.kind !== 'all') {
    count += 1
  }

  return count
}

function MoneyLegDetails({ title, leg, displayFiatCode }: MoneyLegDetailsProps) {
  return (
    <section className="money-leg-card">
      <h4>{title}</h4>
      {!leg ? (
        <p>—</p>
      ) : (
        <div className="money-leg-grid">
          <p>
            <strong>Symbol:</strong> {leg.symbol}
          </p>
          <p>
            <strong>Crypto:</strong> {leg.crypto_amount}
          </p>
          <p>
            <strong>Fiat:</strong> {leg.fiat_amount ? `${leg.fiat_amount}${displayFiatCode ? ` ${displayFiatCode}` : ''}` : '—'}
          </p>
          <p>
            <strong>Error code:</strong> {leg.error?.code ?? '—'}
          </p>
          {leg.error?.candidates && leg.error.candidates.length > 0 ? (
            <p className="column-full">
              <strong>Error candidates:</strong>{' '}
              {leg.error.candidates.map((candidate) => `${candidate.coin_id} (${candidate.name})`).join(', ')}
            </p>
          ) : null}
        </div>
      )}
    </section>
  )
}

function FilterIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" aria-hidden="true">
      <path
        d="M2 3.25h12l-4.7 5.18v3.25L6.7 12.9V8.43L2 3.25Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.4"
        strokeLinejoin="round"
      />
    </svg>
  )
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

  const [draftFilters, setDraftFilters] = useState<TransactionsFilters>(DEFAULT_FILTERS)
  const [appliedFilters, setAppliedFilters] = useState<TransactionsFilters>(DEFAULT_FILTERS)
  const [isFilterPanelOpen, setIsFilterPanelOpen] = useState(false)

  const [transactions, setTransactions] = useState<AggregatedTransaction[]>([])
  const [page, setPage] = useState(0)
  const [pageTokens, setPageTokens] = useState<string[]>([''])
  const [nextPageToken, setNextPageToken] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [expandedTxIds, setExpandedTxIds] = useState<Set<string>>(new Set())

  const loadFiatContext = useCallback(async (): Promise<void> => {
    if (!session) {
      return
    }

    setIsFiatLoading(true)
    setFiatError(null)

    try {
      const [currencies, userSettings] = await Promise.all([
        listSupportedFiatCurrencies(),
        getUserSettings(),
      ])

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
        const { dateFrom, dateTo } = toUtcRange(appliedFilters)
        const normalizedImportId = appliedFilters.importId.trim()
        const normalizedSource = appliedFilters.source === 'all' ? undefined : appliedFilters.source
        const normalizedKind = appliedFilters.kind === 'all' ? undefined : appliedFilters.kind

        const response = await listTransactions({
          pageSize: DEFAULT_PAGE_SIZE,
          pageToken,
          dateFrom,
          dateTo,
          importId: normalizedImportId === '' ? undefined : normalizedImportId,
          source: normalizedSource,
          kind: normalizedKind,
          targetFiat: activeUserFiat,
        })

        setTransactions(response.items)
        setNextPageToken(response.nextPageToken ?? '')
        setPage(pageIndex)
        setExpandedTxIds(new Set())
      } catch (fetchError) {
        setError(toErrorMessage(fetchError, 'Failed to load transactions.'))
        setTransactions([])
        setNextPageToken('')
        setPage(0)
      } finally {
        setIsLoading(false)
      }
    },
    [session, activeUserFiat, appliedFilters],
  )

  useEffect(() => {
    if (!session || isFiatLoading || fiatError || !activeUserFiat) {
      return
    }

    setPageTokens([''])
    void fetchTransactions('', 0)
  }, [session, isFiatLoading, fiatError, activeUserFiat, appliedFilters, fetchTransactions])

  const handleReload = async (): Promise<void> => {
    const currentToken = pageTokens[page] ?? ''
    await fetchTransactions(currentToken, page)
  }

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

  const handleApplyFilters = (): void => {
    setAppliedFilters({
      fromDate: draftFilters.fromDate,
      toDate: draftFilters.toDate,
      source: draftFilters.source,
      kind: draftFilters.kind,
      importId: draftFilters.importId.trim(),
    })
  }

  const handleResetFilters = (): void => {
    setDraftFilters(DEFAULT_FILTERS)
    setAppliedFilters(DEFAULT_FILTERS)
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
        setError(toErrorMessage(updateError, 'Failed to update user currency.'))
        notifications.error('Failed to update user currency', toErrorMessage(updateError))
      } finally {
        setIsFiatUpdating(false)
      }
    },
    [session, activeUserTimezone, activeUserFiat, notifications],
  )

  const toggleExpanded = (txId: string): void => {
    setExpandedTxIds((prev) => {
      const next = new Set(prev)
      if (next.has(txId)) {
        next.delete(txId)
      } else {
        next.add(txId)
      }
      return next
    })
  }

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

  const activeFilterCount = useMemo(() => countActiveFilters(appliedFilters), [appliedFilters])
  const displayFiatCode = activeUserFiat
  const filtersDirty = !filtersEqual(draftFilters, appliedFilters)

  return (
    <section className="stack-lg">
      <PageHeader
        title="Transactions"
        description="Latest aggregated transactions in your current valuation fiat, with optional filters on demand."
      />

      {fiatError ? <ErrorState message={fiatError} actionLabel="Retry" onAction={() => void loadFiatContext()} /> : null}
      {error ? <ErrorState message={error} actionLabel="Retry" onAction={() => void handleReload()} /> : null}

      <article className="card">
        <div className="transactions-toolbar">
          <div className="actions-row">
            <button
              type="button"
              className="btn-secondary"
              onClick={() => setIsFilterPanelOpen((prev) => !prev)}
              aria-expanded={isFilterPanelOpen}
              aria-controls="transactions-filter-panel"
            >
              <FilterIcon />
              {activeFilterCount > 0 ? `Filters (${activeFilterCount})` : 'Filters'}
            </button>

            <button type="button" className="btn-secondary" onClick={() => void handleReload()} disabled={isLoading}>
              Refresh
            </button>
          </div>

          <label className="toolbar-field">
            Fiat currency
            <select
              value={activeUserFiat ?? ''}
              onChange={(event) => void handleFiatChange(event.target.value)}
              disabled={isLoading || isFiatLoading || isFiatUpdating || supportedFiat.length === 0}
            >
              {supportedFiat.map((fiat) => (
                <option key={fiat.code} value={fiat.code}>
                  {fiat.code} · {fiat.displayName}
                </option>
              ))}
            </select>
          </label>
        </div>

        {isFilterPanelOpen ? (
          <div id="transactions-filter-panel" className="transactions-filter-panel">
            <div className="form-grid two-columns">
              <label>
                From date (UTC)
                <input
                  type="date"
                  value={draftFilters.fromDate}
                  onChange={(event) => setDraftFilters((prev) => ({ ...prev, fromDate: event.target.value }))}
                />
              </label>

              <label>
                To date (UTC)
                <input
                  type="date"
                  value={draftFilters.toDate}
                  onChange={(event) => setDraftFilters((prev) => ({ ...prev, toDate: event.target.value }))}
                />
              </label>

              <label>
                Exchange
                <select
                  value={draftFilters.source}
                  onChange={(event) => setDraftFilters((prev) => ({ ...prev, source: event.target.value }))}
                >
                  <option value="all">All exchanges</option>
                  {sourceOptions.map((source) => (
                    <option key={source} value={source}>
                      {source}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                Kind
                <select
                  value={draftFilters.kind}
                  onChange={(event) => setDraftFilters((prev) => ({ ...prev, kind: event.target.value }))}
                >
                  <option value="all">All kinds</option>
                  {kindOptions.map((kind) => (
                    <option key={kind} value={kind}>
                      {kind}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                Import ID
                <input
                  value={draftFilters.importId}
                  onChange={(event) => setDraftFilters((prev) => ({ ...prev, importId: event.target.value }))}
                  placeholder="Optional UUID"
                />
              </label>
            </div>

            <div className="actions-row">
              <button type="button" className="btn-primary" onClick={handleApplyFilters} disabled={isLoading || !filtersDirty}>
                Apply filters
              </button>
              <button
                type="button"
                className="btn-secondary"
                onClick={handleResetFilters}
                disabled={isLoading || (activeFilterCount === 0 && !filtersDirty)}
              >
                Reset
              </button>
            </div>
          </div>
        ) : null}

        <p className="hint-text">
          Default view loads the latest transactions automatically. Pagination shows {DEFAULT_PAGE_SIZE} rows per page.
        </p>
        {displayFiatCode ? <p className="hint-text">User valuation currency: {displayFiatCode}.</p> : null}
        <p className="hint-text">Filters are applied server-side with cursor pagination.</p>
      </article>

      {isFiatLoading && !activeUserFiat ? <LoadingState label="Loading fiat currency context..." /> : null}
      {isLoading ? <LoadingState label="Fetching aggregated transactions..." /> : null}

      {!isLoading && !error && transactions.length === 0 ? (
        <EmptyState title="No transactions found" description="Try adjusting the optional filters or load more recent imports." />
      ) : null}

      {!isLoading && !error && transactions.length > 0 ? (
        <article className="card">
          <div className="table-toolbar">
            <p className="table-summary">
              Page: {page + 1} | Displayed rows: {transactions.length}
            </p>
            <div className="pagination-controls">
              <button type="button" className="btn-secondary" onClick={() => void handlePrevPage()} disabled={isLoading || page === 0}>
                Previous
              </button>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => void handleNextPage()}
                disabled={isLoading || nextPageToken === ''}
              >
                Next
              </button>
            </div>
          </div>

          <div className="table-wrapper">
            <table className="data-table tx-table">
              <thead>
                <tr>
                  <th></th>
                  <th>TimeUTC</th>
                  <th>Source</th>
                  <th>Kind</th>
                  <th>InMoney</th>
                  <th>OutMoney</th>
                  <th>FeeMoney</th>
                  <th>ContractSymbol</th>
                  <th>TxHash</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((tx) => {
                  const expanded = expandedTxIds.has(tx.txId)

                  return (
                    <Fragment key={tx.txId}>
                      <tr>
                        <td>
                          <button
                            type="button"
                            className="btn-link expand-toggle"
                            onClick={() => toggleExpanded(tx.txId)}
                            aria-expanded={expanded}
                            aria-label={expanded ? 'Collapse row details' : 'Expand row details'}
                          >
                            {expanded ? '−' : '+'}
                          </button>
                        </td>
                        <td>{formatUtcTimestamp(tx.timeUtc)}</td>
                        <td>{tx.source || '—'}</td>
                        <td>{tx.kind || '—'}</td>
                        <td>{formatMoneyLegSummary(tx.inMoney, displayFiatCode)}</td>
                        <td>{formatMoneyLegSummary(tx.outMoney, displayFiatCode)}</td>
                        <td>{formatMoneyLegSummary(tx.feeMoney, displayFiatCode)}</td>
                        <td>{tx.contractSymbol || '—'}</td>
                        <td className="mono-text">{truncateMiddle(tx.txHash)}</td>
                      </tr>
                      {expanded ? (
                        <tr className="tx-details-row">
                          <td colSpan={9}>
                            <div className="tx-details-panel">
                              <dl className="tx-details-grid">
                                <dt>ID</dt>
                                <dd className="mono-text">{tx.txId}</dd>
                                <dt>ImportID</dt>
                                <dd className="mono-text">{tx.importId}</dd>
                                <dt>DerivativeKind</dt>
                                <dd>{tx.derivativeKind || '—'}</dd>
                                <dt>PositionID</dt>
                                <dd className="mono-text">{tx.positionId || '—'}</dd>
                                <dt>OrderID</dt>
                                <dd className="mono-text">{tx.orderId || '—'}</dd>
                                <dt>Note</dt>
                                <dd>{tx.note || '—'}</dd>
                                <dt>TxFingerprint</dt>
                                <dd className="mono-text">{tx.txFingerprint}</dd>
                                <dt>Full TxHash</dt>
                                <dd className="mono-text">{tx.txHash || '—'}</dd>
                              </dl>

                              <div className="tx-money-details">
                                <MoneyLegDetails title="InMoney" leg={tx.inMoney} displayFiatCode={displayFiatCode} />
                                <MoneyLegDetails title="OutMoney" leg={tx.outMoney} displayFiatCode={displayFiatCode} />
                                <MoneyLegDetails title="FeeMoney" leg={tx.feeMoney} displayFiatCode={displayFiatCode} />
                              </div>
                            </div>
                          </td>
                        </tr>
                      ) : null}
                    </Fragment>
                  )
                })}
              </tbody>
            </table>
          </div>
        </article>
      ) : null}
    </section>
  )
}
