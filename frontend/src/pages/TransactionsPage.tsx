import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import {
  getTenantSettings,
  listSupportedFiatCurrencies,
  listTransactionsByImport,
  upsertTenantSettings,
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

const DEFAULT_LIMIT = 50

type SortMode = 'time_desc' | 'time_asc'

interface MoneyLegDetailsProps {
  title: string
  leg: MoneyLeg | undefined
  displayFiatCode: string | null
}

function formatUtcTimestamp(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return `${UTC_DATE_TIME_FORMATTER.format(date)} UTC`
}

function parseTimestamp(value: string): number {
  const date = new Date(value)
  const ms = date.getTime()
  return Number.isNaN(ms) ? 0 : ms
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

export function TransactionsPage() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [importId, setImportId] = useState('')
  const [limit, setLimit] = useState(DEFAULT_LIMIT)
  const [offset, setOffset] = useState(0)

  const [supportedFiat, setSupportedFiat] = useState<SupportedFiatCurrency[]>([])
  const [activeTenantFiat, setActiveTenantFiat] = useState<string | null>(null)
  const [activeTenantTimezone, setActiveTenantTimezone] = useState<string | null>(null)
  const [isFiatLoading, setIsFiatLoading] = useState(false)
  const [isFiatUpdating, setIsFiatUpdating] = useState(false)
  const [fiatError, setFiatError] = useState<string | null>(null)

  const [transactions, setTransactions] = useState<AggregatedTransaction[]>([])
  const [total, setTotal] = useState(0)
  const [hasSearched, setHasSearched] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [sourceFilter, setSourceFilter] = useState('all')
  const [fiatFilter, setFiatFilter] = useState('')
  const [sortMode, setSortMode] = useState<SortMode>('time_desc')
  const [expandedTxIds, setExpandedTxIds] = useState<Set<string>>(new Set())

  const loadFiatContext = useCallback(async (): Promise<void> => {
    if (!session) {
      return
    }

    setIsFiatLoading(true)
    setFiatError(null)

    try {
      const [currencies, tenantSettings] = await Promise.all([
        listSupportedFiatCurrencies(),
        getTenantSettings(),
      ])

      setSupportedFiat(currencies)
      setActiveTenantFiat(tenantSettings.fiatCurrency)
      setActiveTenantTimezone(tenantSettings.timezone)
      setFiatFilter(tenantSettings.fiatCurrency)
    } catch (loadError) {
      setFiatError(toErrorMessage(loadError, 'Failed to load fiat currency context.'))
      setSupportedFiat([])
      setActiveTenantFiat(null)
      setActiveTenantTimezone(null)
      setFiatFilter('')
    } finally {
      setIsFiatLoading(false)
    }
  }, [session])

  useEffect(() => {
    void loadFiatContext()
  }, [loadFiatContext])

  const fetchTransactions = useCallback(
    async (nextOffset: number): Promise<void> => {
      if (!session) {
        return
      }

      setHasSearched(true)
      setIsLoading(true)
      setError(null)

      try {
        const response = await listTransactionsByImport({
          importId,
          limit,
          offset: nextOffset,
        })

        setOffset(nextOffset)
        setTransactions(response.transactions)
        setTotal(response.total)
        setExpandedTxIds(new Set())
      } catch (fetchError) {
        setError(toErrorMessage(fetchError, 'Failed to load transactions.'))
        setTransactions([])
        setTotal(0)
      } finally {
        setIsLoading(false)
      }
    },
    [session, importId, limit],
  )

  const handleSearchSubmit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault()
    await fetchTransactions(0)
  }

  const handleReload = async (): Promise<void> => {
    await fetchTransactions(offset)
  }

  const handlePrevPage = async (): Promise<void> => {
    const nextOffset = Math.max(0, offset - limit)
    await fetchTransactions(nextOffset)
  }

  const handleNextPage = async (): Promise<void> => {
    const nextOffset = offset + limit
    if (nextOffset >= total) {
      return
    }
    await fetchTransactions(nextOffset)
  }

  const sourceOptions = useMemo(() => {
    const set = new Set<string>()
    transactions.forEach((tx) => {
      if (tx.source.trim() !== '') {
        set.add(tx.source)
      }
    })
    return Array.from(set).sort((a, b) => a.localeCompare(b))
  }, [transactions])

  const filteredTransactions = useMemo(() => {
    let result = [...transactions]

    if (sourceFilter !== 'all') {
      result = result.filter((tx) => tx.source === sourceFilter)
    }

    result.sort((a, b) => {
      const left = parseTimestamp(a.timeUtc)
      const right = parseTimestamp(b.timeUtc)
      return sortMode === 'time_desc' ? right - left : left - right
    })

    return result
  }, [transactions, sourceFilter, sortMode])

  const displayFiatCode = useMemo(() => {
    if (fiatFilter) {
      return fiatFilter
    }
    return activeTenantFiat
  }, [fiatFilter, activeTenantFiat])

  const handleFiatChange = useCallback(
    async (nextFiatCode: string): Promise<void> => {
      setFiatFilter(nextFiatCode)
      setError(null)

      if (!session || !activeTenantTimezone || nextFiatCode.trim() === '' || nextFiatCode === activeTenantFiat) {
        return
      }

      setIsFiatUpdating(true)
      try {
        const updatedSettings = await upsertTenantSettings({
          fiatCurrency: nextFiatCode,
          timezone: activeTenantTimezone,
        })

        setActiveTenantFiat(updatedSettings.fiatCurrency)
        setActiveTenantTimezone(updatedSettings.timezone)
        setFiatFilter(updatedSettings.fiatCurrency)
        notifications.success('Tenant currency updated', `Default fiat currency: ${updatedSettings.fiatCurrency}`)

        if (hasSearched && importId.trim() !== '') {
          await fetchTransactions(0)
        }
      } catch (updateError) {
        setError(toErrorMessage(updateError, 'Failed to update tenant currency.'))
        setFiatFilter(activeTenantFiat ?? '')
        notifications.error('Failed to update tenant currency', toErrorMessage(updateError))
      } finally {
        setIsFiatUpdating(false)
      }
    },
    [session, activeTenantTimezone, activeTenantFiat, notifications, hasSearched, importId, fetchTransactions],
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

  return (
    <section className="stack-lg">
      <PageHeader
        title="Transactions"
        description="Review aggregated transactions for an import in a dense accounting-friendly table."
      />

      <article className="card">
        <form onSubmit={handleSearchSubmit} className="form-grid two-columns">
          <label className="column-full">
            Import ID (UUID)
            <input
              required
              value={importId}
              onChange={(event) => setImportId(event.target.value)}
              placeholder="00000000-0000-0000-0000-000000000000"
            />
          </label>

          <label>
            Page size
            <input
              type="number"
              min={1}
              max={200}
              value={limit}
              onChange={(event) => setLimit(Math.min(200, Math.max(1, Number(event.target.value) || DEFAULT_LIMIT)))}
            />
          </label>

          <label>
            Source filter
            <select value={sourceFilter} onChange={(event) => setSourceFilter(event.target.value)} disabled={isLoading}>
              <option value="all">All sources</option>
              {sourceOptions.map((source) => (
                <option key={source} value={source}>
                  {source}
                </option>
              ))}
            </select>
          </label>

          <label>
            Fiat display currency
            <select
              value={fiatFilter}
              onChange={(event) => void handleFiatChange(event.target.value)}
              disabled={isLoading || isFiatLoading || isFiatUpdating}
            >
              {supportedFiat.map((fiat) => (
                <option key={fiat.code} value={fiat.code}>
                  {fiat.code} · {fiat.displayName}
                </option>
              ))}
            </select>
          </label>

          <label>
            Sort
            <select value={sortMode} onChange={(event) => setSortMode(event.target.value as SortMode)} disabled={isLoading}>
              <option value="time_desc">TimeUTC: newest first</option>
              <option value="time_asc">TimeUTC: oldest first</option>
            </select>
          </label>

          <div className="column-full actions-row">
            <button type="submit" className="btn-primary" disabled={isLoading || importId.trim() === ''}>
              {isLoading ? 'Loading...' : 'Load transactions'}
            </button>
            {hasSearched ? (
              <button type="button" className="btn-secondary" onClick={() => void handleReload()} disabled={isLoading}>
                Reload
              </button>
            ) : null}
          </div>
        </form>

        <p className="hint-text">
          Server-side pagination is used (`limit`/`offset`). Sorting and source filter are applied on the current page.
        </p>
        {activeTenantFiat ? (
          <p className="hint-text">
            Tenant valuation currency: {activeTenantFiat}. Changing fiat updates tenant settings via `upsertTenantSettings`.
          </p>
        ) : null}
      </article>

      {isFiatLoading ? <LoadingState label="Loading supported fiat currencies..." /> : null}
      {fiatError ? <ErrorState message={fiatError} actionLabel="Retry" onAction={() => void loadFiatContext()} /> : null}

      {isLoading ? <LoadingState label="Fetching aggregated transactions..." /> : null}
      {error ? <ErrorState message={error} actionLabel="Retry" onAction={() => void handleReload()} /> : null}

      {!isLoading && !error && !hasSearched ? (
        <EmptyState title="Search pending" description="Enter Import ID and load transactions." />
      ) : null}

      {!isLoading && !error && hasSearched && filteredTransactions.length === 0 ? (
        <EmptyState title="No transactions found" description="Try changing Import ID, pagination, or filters." />
      ) : null}

      {!isLoading && !error && filteredTransactions.length > 0 ? (
        <article className="card">
          <div className="table-toolbar">
            <p className="table-summary">
              Total (server): {total} | Current page offset: {offset} | Displayed rows: {filteredTransactions.length}
            </p>
            <div className="pagination-controls">
              <button type="button" className="btn-secondary" onClick={() => void handlePrevPage()} disabled={isLoading || offset === 0}>
                Previous
              </button>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => void handleNextPage()}
                disabled={isLoading || offset + limit >= total}
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
                {filteredTransactions.map((tx) => {
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
