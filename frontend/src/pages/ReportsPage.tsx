import { Fragment, useCallback, useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { getTaxReportStatus, listTaxReports, startTaxReport, type TaxJob, type TaxPolicy } from '../api/taxService'
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

const TAX_YEAR_MIN = 2000
const TAX_YEAR_MAX = new Date().getUTCFullYear() + 1
const DEFAULT_LIMIT = 20
const PAGE_SIZE_OPTIONS = [20, 50, 100] as const
const JURISDICTION_OPTIONS = ['RU', 'KZ'] as const
const COST_BASIS_OPTIONS: Array<TaxPolicy['costBasisMethod']> = ['FIFO', 'LIFO', 'AVG']

type JurisdictionOption = (typeof JURISDICTION_OPTIONS)[number]

interface CreateReportFormState {
  taxYear: number
  jurisdiction: JurisdictionOption
  costBasisMethod: TaxPolicy['costBasisMethod']
  treatCryptoCryptoAsDisposal: boolean
}

const INITIAL_CREATE_FORM: CreateReportFormState = {
  taxYear: new Date().getUTCFullYear() - 1,
  jurisdiction: 'RU',
  costBasisMethod: 'FIFO',
  treatCryptoCryptoAsDisposal: false,
}

function formatUtcTimestamp(value?: string): string {
  if (!value) {
    return '—'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return `${UTC_DATE_TIME_FORMATTER.format(date)} UTC`
}

function truncateMiddle(value?: string, head = 10, tail = 8): string {
  if (!value || value.trim() === '') {
    return '—'
  }

  if (value.length <= head + tail + 3) {
    return value
  }

  return `${value.slice(0, head)}...${value.slice(-tail)}`
}

function normalizeStatus(status: string): string {
  return status.trim().toLowerCase()
}

function statusBadgeClass(status: string): string {
  switch (normalizeStatus(status)) {
    case 'queued':
      return 'status-badge status-badge-queued'
    case 'running':
      return 'status-badge status-badge-running'
    case 'success':
      return 'status-badge status-badge-success'
    case 'failed':
      return 'status-badge status-badge-failed'
    case 'canceled':
      return 'status-badge status-badge-canceled'
    default:
      return 'status-badge status-badge-neutral'
  }
}

function statusLabel(status: string): string {
  const normalized = normalizeStatus(status)
  if (!normalized) {
    return 'UNKNOWN'
  }

  return normalized.toUpperCase()
}

export function ReportsPage() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [jobs, setJobs] = useState<TaxJob[]>([])
  const [total, setTotal] = useState(0)
  const [limit, setLimit] = useState(DEFAULT_LIMIT)
  const [offset, setOffset] = useState(0)

  const [isLoadingList, setIsLoadingList] = useState(false)
  const [listError, setListError] = useState<string | null>(null)

  const [expandedReportIds, setExpandedReportIds] = useState<Set<string>>(new Set())
  const [detailsByReportId, setDetailsByReportId] = useState<Record<string, TaxJob>>({})
  const [detailsLoadingByReportId, setDetailsLoadingByReportId] = useState<Record<string, boolean>>({})
  const [detailsErrorByReportId, setDetailsErrorByReportId] = useState<Record<string, string>>({})

  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isCreating, setIsCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [createForm, setCreateForm] = useState<CreateReportFormState>(INITIAL_CREATE_FORM)

  const loadList = useCallback(
    async (nextOffset: number): Promise<void> => {
      if (!session) {
        return
      }

      setIsLoadingList(true)
      setListError(null)

      try {
        const response = await listTaxReports({ limit, offset: nextOffset })

        setJobs(response.jobs)
        setTotal(response.total)
        setOffset(nextOffset)
      } catch (error) {
        setListError(toErrorMessage(error, 'Unable to load tax reports.'))
      } finally {
        setIsLoadingList(false)
      }
    },
    [session, limit],
  )

  useEffect(() => {
    void loadList(0)
  }, [loadList])

  const loadDetails = useCallback(
    async (reportId: string, force = false): Promise<void> => {
      if (!session) {
        return
      }

      if (!force && detailsByReportId[reportId]) {
        return
      }

      setDetailsLoadingByReportId((prev) => ({
        ...prev,
        [reportId]: true,
      }))

      setDetailsErrorByReportId((prev) => {
        const next = { ...prev }
        delete next[reportId]
        return next
      })

      try {
        const job = await getTaxReportStatus(reportId)
        setDetailsByReportId((prev) => ({
          ...prev,
          [reportId]: job,
        }))
      } catch (error) {
        const message = toErrorMessage(error, 'Unable to load report details.')

        setDetailsErrorByReportId((prev) => ({
          ...prev,
          [reportId]: message,
        }))

        notifications.error('Unable to load report details', message)
      } finally {
        setDetailsLoadingByReportId((prev) => ({
          ...prev,
          [reportId]: false,
        }))
      }
    },
    [session, detailsByReportId, notifications],
  )

  const toggleExpanded = (reportId: string): void => {
    let shouldLoadDetails = false

    setExpandedReportIds((prev) => {
      const next = new Set(prev)
      if (next.has(reportId)) {
        next.delete(reportId)
      } else {
        next.add(reportId)
        shouldLoadDetails = true
      }

      return next
    })

    if (shouldLoadDetails) {
      void loadDetails(reportId)
    }
  }

  const handleCreateReport = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault()

    if (!session) {
      return
    }

    if (!Number.isInteger(createForm.taxYear) || createForm.taxYear < TAX_YEAR_MIN || createForm.taxYear > TAX_YEAR_MAX) {
      setCreateError(`Tax year must be between ${TAX_YEAR_MIN} and ${TAX_YEAR_MAX}.`)
      return
    }

    setIsCreating(true)
    setCreateError(null)

    try {
      const created = await startTaxReport({
        taxYear: createForm.taxYear,
        taxPolicy: {
          treatCryptoCryptoAsDisposal: createForm.treatCryptoCryptoAsDisposal,
          costBasisMethod: createForm.costBasisMethod,
          jurisdiction: createForm.jurisdiction,
        },
      })

      notifications.success('Report created', `Report ID: ${created.reportId}`)
      setIsCreateModalOpen(false)
      setExpandedReportIds(new Set([created.reportId]))
      await loadList(0)
      void loadDetails(created.reportId, true)
    } catch (error) {
      setCreateError(toErrorMessage(error, 'Unable to create report.'))
    } finally {
      setIsCreating(false)
    }
  }

  const handleCloseCreateModal = (): void => {
    if (isCreating) {
      return
    }

    setIsCreateModalOpen(false)
    setCreateError(null)
  }

  return (
    <section className="stack-lg">
      <PageHeader
        title="Tax Reports"
        description="Review tax report jobs, monitor asynchronous processing, and inspect policy and summary outputs."
        actions={
          <div className="actions-row">
            <button type="button" className="btn-secondary" onClick={() => void loadList(offset)} disabled={isLoadingList}>
              Refresh list
            </button>
            <button
              type="button"
              className="btn-primary"
              onClick={() => {
                setCreateError(null)
                setCreateForm(INITIAL_CREATE_FORM)
                setIsCreateModalOpen(true)
              }}
            >
              Create report
            </button>
          </div>
        }
      />

      {listError ? <ErrorState message={listError} actionLabel="Retry" onAction={() => void loadList(offset)} /> : null}

      {isLoadingList && jobs.length === 0 ? <LoadingState label="Loading tax reports..." /> : null}

      {!isLoadingList && jobs.length === 0 && !listError ? (
        <EmptyState
          title="No tax reports yet"
          description="Create the first report job to start asynchronous tax calculation workflow."
        />
      ) : null}

      {jobs.length > 0 ? (
        <article className="card">
          <div className="table-toolbar">
            <p className="table-summary">
              Total (server): {total} | Current offset: {offset} | Rows: {jobs.length}
            </p>
            <div className="pagination-controls">
              <label>
                Page size
                <select
                  value={limit}
                  onChange={(event) => setLimit(Number(event.target.value) || DEFAULT_LIMIT)}
                  disabled={isLoadingList}
                >
                  {PAGE_SIZE_OPTIONS.map((size) => (
                    <option key={size} value={size}>
                      {size}
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => void loadList(Math.max(0, offset - limit))}
                disabled={isLoadingList || offset === 0}
              >
                Previous
              </button>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => void loadList(offset + limit)}
                disabled={isLoadingList || offset + limit >= total}
              >
                Next
              </button>
            </div>
          </div>

          <div className="table-wrapper">
            <table className="data-table reports-table">
              <thead>
                <tr>
                  <th></th>
                  <th>ID</th>
                  <th>TaxYear</th>
                  <th>Status</th>
                  <th>CreatedAt</th>
                  <th>AuditZipURL</th>
                  <th>ReportURL</th>
                </tr>
              </thead>
              <tbody>
                {jobs.map((job) => {
                  const expanded = expandedReportIds.has(job.reportId)
                  const details = detailsByReportId[job.reportId] ?? job
                  const detailsLoading = detailsLoadingByReportId[job.reportId] ?? false
                  const detailsError = detailsErrorByReportId[job.reportId] ?? null

                  return (
                    <Fragment key={job.reportId}>
                      <tr>
                        <td>
                          <button
                            type="button"
                            className="btn-link expand-toggle"
                            onClick={() => toggleExpanded(job.reportId)}
                            aria-expanded={expanded}
                            aria-label={expanded ? 'Collapse report details' : 'Expand report details'}
                          >
                            {expanded ? '−' : '+'}
                          </button>
                        </td>
                        <td className="mono-text">{truncateMiddle(job.reportId)}</td>
                        <td>{job.taxYear}</td>
                        <td>
                          <span className={statusBadgeClass(job.status)}>{statusLabel(job.status)}</span>
                        </td>
                        <td>{formatUtcTimestamp(job.createdAt)}</td>
                        <td>
                          {job.auditZipUrl ? (
                            <a className="report-link" href={job.auditZipUrl} target="_blank" rel="noreferrer">
                              Download ZIP
                            </a>
                          ) : (
                            '—'
                          )}
                        </td>
                        <td>
                          {job.reportUrl ? (
                            <a className="report-link" href={job.reportUrl} target="_blank" rel="noreferrer">
                              Open report
                            </a>
                          ) : (
                            '—'
                          )}
                        </td>
                      </tr>

                      {expanded ? (
                        <tr className="report-details-row">
                          <td colSpan={7}>
                            <div className="report-details-panel">
                              <div className="report-details-header">
                                <p className="mono-text">{details.reportId}</p>
                                <button
                                  type="button"
                                  className="btn-link"
                                  onClick={() => void loadDetails(job.reportId, true)}
                                  disabled={detailsLoading}
                                >
                                  {detailsLoading ? 'Refreshing...' : 'Refresh details'}
                                </button>
                              </div>

                              {detailsError ? <p className="form-error">{detailsError}</p> : null}

                              <dl className="report-details-grid">
                                <dt>StartedAt</dt>
                                <dd>{formatUtcTimestamp(details.startedAt)}</dd>
                                <dt>FinishedAt</dt>
                                <dd>{formatUtcTimestamp(details.finishedAt)}</dd>
                                <dt>Attempts</dt>
                                <dd>{details.attempts}</dd>
                                <dt>LastErrorCode</dt>
                                <dd className="mono-text">{details.lastErrorCode || '—'}</dd>
                                <dt>LastErrorMessage</dt>
                                <dd>{details.lastErrorMessage || '—'}</dd>
                              </dl>

                              <div className="report-details-sections">
                                <section className="report-details-card">
                                  <h4>PolicySnapshot</h4>
                                  <dl className="report-kv">
                                    <dt>Jurisdiction</dt>
                                    <dd>{details.policySnapshot.jurisdiction}</dd>
                                    <dt>CostBasisMethod</dt>
                                    <dd>{details.policySnapshot.costBasisMethod}</dd>
                                    <dt>TreatCryptoCryptoAsDisposal</dt>
                                    <dd>{details.policySnapshot.treatCryptoCryptoAsDisposal ? 'true' : 'false'}</dd>
                                  </dl>
                                </section>

                                <section className="report-details-card">
                                  <h4>Summary</h4>
                                  {details.summary ? (
                                    <dl className="report-kv">
                                      <dt>TotalIncomeFiat</dt>
                                      <dd>{details.summary.totalIncomeFiat}</dd>
                                      <dt>TotalExpenseFiat</dt>
                                      <dd>{details.summary.totalExpenseFiat}</dd>
                                      <dt>TaxBaseFiat</dt>
                                      <dd>{details.summary.taxBaseFiat}</dd>
                                      <dt>TaxDueFiat</dt>
                                      <dd>{details.summary.taxDueFiat}</dd>
                                    </dl>
                                  ) : (
                                    <p className="hint-text">Summary is not available until processing completes.</p>
                                  )}
                                </section>

                                <section className="report-details-card">
                                  <h4>Artifacts</h4>
                                  <dl className="report-kv">
                                    <dt>AuditZipURL</dt>
                                    <dd>
                                      {details.auditZipUrl ? (
                                        <a className="report-link" href={details.auditZipUrl} target="_blank" rel="noreferrer">
                                          {truncateMiddle(details.auditZipUrl, 18, 14)}
                                        </a>
                                      ) : (
                                        '—'
                                      )}
                                    </dd>
                                    <dt>ReportURL</dt>
                                    <dd>
                                      {details.reportUrl ? (
                                        <a className="report-link" href={details.reportUrl} target="_blank" rel="noreferrer">
                                          {truncateMiddle(details.reportUrl, 18, 14)}
                                        </a>
                                      ) : (
                                        '—'
                                      )}
                                    </dd>
                                  </dl>
                                </section>
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

      {isCreateModalOpen ? (
        <div className="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="create-report-title">
          <article className="modal-card">
            <header className="modal-header">
              <div>
                <h3 id="create-report-title">Create Tax Report</h3>
                <p>New report job will be queued and processed asynchronously by tax-svc.</p>
              </div>
              <button type="button" className="btn-link modal-close" onClick={handleCloseCreateModal} disabled={isCreating}>
                ×
              </button>
            </header>

            <form className="form-grid two-columns" onSubmit={handleCreateReport}>
              <label>
                Tax year
                <input
                  type="number"
                  min={TAX_YEAR_MIN}
                  max={TAX_YEAR_MAX}
                  value={createForm.taxYear}
                  onChange={(event) => {
                    const value = Number(event.target.value)
                    setCreateForm((prev) => ({
                      ...prev,
                      taxYear: Number.isFinite(value) ? value : prev.taxYear,
                    }))
                  }}
                  required
                  disabled={isCreating}
                />
              </label>

              <label>
                Jurisdiction
                <select
                  value={createForm.jurisdiction}
                  onChange={(event) =>
                    setCreateForm((prev) => ({
                      ...prev,
                      jurisdiction: event.target.value as JurisdictionOption,
                    }))
                  }
                  disabled={isCreating}
                >
                  {JURISDICTION_OPTIONS.map((jurisdiction) => (
                    <option key={jurisdiction} value={jurisdiction}>
                      {jurisdiction}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                Cost basis method
                <select
                  value={createForm.costBasisMethod}
                  onChange={(event) =>
                    setCreateForm((prev) => ({
                      ...prev,
                      costBasisMethod: event.target.value as TaxPolicy['costBasisMethod'],
                    }))
                  }
                  disabled={isCreating}
                >
                  {COST_BASIS_OPTIONS.map((method) => (
                    <option key={method} value={method}>
                      {method}
                    </option>
                  ))}
                </select>
              </label>

              <label className="inline-checkbox">
                <input
                  type="checkbox"
                  checked={createForm.treatCryptoCryptoAsDisposal}
                  onChange={(event) =>
                    setCreateForm((prev) => ({
                      ...prev,
                      treatCryptoCryptoAsDisposal: event.target.checked,
                    }))
                  }
                  disabled={isCreating}
                />
                Treat crypto-to-crypto as disposal
              </label>

              <p className="hint-text column-full">Supported tax year range: {TAX_YEAR_MIN} - {TAX_YEAR_MAX}.</p>

              {createError ? <div className="form-error column-full">{createError}</div> : null}

              <div className="column-full modal-actions">
                <button type="button" className="btn-secondary" onClick={handleCloseCreateModal} disabled={isCreating}>
                  Cancel
                </button>
                <button type="submit" className="btn-primary" disabled={isCreating}>
                  {isCreating ? 'Creating...' : 'Create report'}
                </button>
              </div>
            </form>
          </article>
        </div>
      ) : null}
    </section>
  )
}
